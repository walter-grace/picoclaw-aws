// Package mcp provides MCP (Model Context Protocol) client integration for pico-aws.
package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/bigneek/claw-cubed/pkg/logger"
)

const (
	connectTimeout = 30 * time.Second
	callTimeout    = 120 * time.Second
)

// AWSMCPClient connects to the AWS MCP Server via mcp-proxy-for-aws.
type AWSMCPClient struct {
	proxyCommand string
	proxyArgs    []string
	client       *mcp.Client
	session      *mcp.ClientSession
	mu           sync.Mutex
}

// AWSMCPClientConfig configures the AWS MCP client.
type AWSMCPClientConfig struct {
	ProxyCommand string
	ProxyArgs    []string
	Region       string
}

// DefaultAWSMCPConfig returns the default configuration for AWS MCP.
func DefaultAWSMCPConfig(region string) AWSMCPClientConfig {
	if region == "" {
		region = "us-east-1"
	}
	return AWSMCPClientConfig{
		ProxyCommand: "uvx",
		ProxyArgs: []string{
			"mcp-proxy-for-aws@latest",
			"https://aws-mcp.us-east-1.api.aws/mcp",
			"--metadata", "AWS_REGION=" + region,
		},
		Region: region,
	}
}

// NewAWSMCPClient creates a new AWS MCP client.
func NewAWSMCPClient(cfg AWSMCPClientConfig) *AWSMCPClient {
	proxyCommand := cfg.ProxyCommand
	if proxyCommand == "" {
		proxyCommand = "uvx"
	}
	proxyArgs := cfg.ProxyArgs
	if len(proxyArgs) == 0 {
		region := cfg.Region
		if region == "" {
			region = "us-east-1"
		}
		proxyArgs = []string{
			"mcp-proxy-for-aws@latest",
			"https://aws-mcp.us-east-1.api.aws/mcp",
			"--metadata", "AWS_REGION=" + region,
		}
	}
	return &AWSMCPClient{
		proxyCommand: proxyCommand,
		proxyArgs:    proxyArgs,
	}
}

// Connect establishes a connection to the AWS MCP server via the proxy.
func (c *AWSMCPClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil {
		return nil
	}

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	cmd := exec.Command(c.proxyCommand, c.proxyArgs...)
	transport := &mcp.CommandTransport{Command: cmd}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "pico-aws-mcp-client",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(connectCtx, transport, nil)
	if err != nil {
		logger.ErrorCF("mcp", "Failed to connect to AWS MCP",
			map[string]interface{}{
				"error": err.Error(),
				"hint":  "Ensure uv/uvx is installed: curl -LsSf https://astral.sh/uv/install.sh | sh",
			})
		return fmt.Errorf("connect to AWS MCP: %w", err)
	}

	c.client = client
	c.session = session
	logger.InfoCF("mcp", "Connected to AWS MCP server", nil)
	return nil
}

// ListTools returns the list of tools available from the AWS MCP server.
func (c *AWSMCPClient) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}

	c.mu.Lock()
	session := c.session
	c.mu.Unlock()

	if session == nil {
		return nil, fmt.Errorf("not connected to AWS MCP")
	}

	callCtx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	result, err := session.ListTools(callCtx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	return result.Tools, nil
}

// CallTool invokes a tool on the AWS MCP server.
func (c *AWSMCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, bool, error) {
	if err := c.Connect(ctx); err != nil {
		return "", false, err
	}

	c.mu.Lock()
	session := c.session
	c.mu.Unlock()

	if session == nil {
		return "", false, fmt.Errorf("not connected to AWS MCP")
	}

	callCtx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	params := &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	}

	result, err := session.CallTool(callCtx, params)
	if err != nil {
		return "", false, fmt.Errorf("call tool %s: %w", name, err)
	}

	content := extractTextContent(result.Content)
	if result.IsError {
		return content, true, nil
	}
	return content, false, nil
}

// extractTextContent concatenates text from Content items.
func extractTextContent(content []mcp.Content) string {
	if len(content) == 0 {
		return ""
	}
	var parts []string
	for _, c := range content {
		if tc, ok := c.(*mcp.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	result := ""
	for _, p := range parts {
		if result != "" {
			result += "\n\n"
		}
		result += p
	}
	return result
}

// Close closes the connection to the AWS MCP server.
func (c *AWSMCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil {
		err := c.session.Close()
		c.session = nil
		c.client = nil
		return err
	}
	return nil
}
