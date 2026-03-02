package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/walter-grace/picoclaw-aws/pkg/config"
	"github.com/walter-grace/picoclaw-aws/pkg/logger"
	"github.com/walter-grace/picoclaw-aws/pkg/tools"
)

// NewAWSMCPClientFromConfig creates an AWS MCP client from pico-aws config.
func NewAWSMCPClientFromConfig(cfg *config.AWSMCPConfig) *AWSMCPClient {
	mcpCfg := AWSMCPClientConfig{
		ProxyCommand: cfg.ProxyCommand,
		ProxyArgs:    cfg.ProxyArgs,
		Region:       cfg.Region,
	}
	return NewAWSMCPClient(mcpCfg)
}

const awsToolPrefix = "aws__"

// AWSMCPToolProxy is a pico-aws tool that proxies to an AWS MCP server tool.
type AWSMCPToolProxy struct {
	mcpTool   *mcp.Tool
	client    *AWSMCPClient
	picoName  string
}

// NewAWSMCPToolProxy creates a tool proxy for a single MCP tool.
func NewAWSMCPToolProxy(mcpTool *mcp.Tool, client *AWSMCPClient) *AWSMCPToolProxy {
	name := awsToolPrefix + mcpTool.Name
	return &AWSMCPToolProxy{
		mcpTool:  mcpTool,
		client:   client,
		picoName: name,
	}
}

// Name returns the pico-aws tool name (aws__<mcp_name>).
func (t *AWSMCPToolProxy) Name() string {
	return t.picoName
}

// Description returns the tool description.
func (t *AWSMCPToolProxy) Description() string {
	if t.mcpTool.Description != "" {
		return "[AWS] " + t.mcpTool.Description
	}
	return "[AWS] " + t.mcpTool.Name
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *AWSMCPToolProxy) Parameters() map[string]interface{} {
	if t.mcpTool.InputSchema == nil {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	// InputSchema can be map[string]any, *jsonschema.Schema, or json.RawMessage
	switch s := t.mcpTool.InputSchema.(type) {
	case map[string]interface{}:
		return s
	case nil:
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	default:
		// Marshal and unmarshal to get map[string]interface{}
		data, err := json.Marshal(s)
		if err != nil {
			return map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}
		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			return map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}
		return m
	}
}

// Execute runs the tool via the AWS MCP client.
func (t *AWSMCPToolProxy) Execute(ctx context.Context, args map[string]interface{}) *tools.ToolResult {
	content, isError, err := t.client.CallTool(ctx, t.mcpTool.Name, args)
	if err != nil {
		logger.ErrorCF("mcp", "AWS MCP tool call failed",
			map[string]interface{}{
				"tool":  t.picoName,
				"error": err.Error(),
			})
		return tools.ErrorResult(fmt.Sprintf("AWS MCP tool %s failed: %v", t.mcpTool.Name, err))
	}

	if isError {
		return tools.ErrorResult(content)
	}

	return tools.SilentResult(content)
}

// RegisterAWSMCPTools connects to the AWS MCP server, lists tools, and registers
// each as a PicoClaw tool with the aws__ prefix.
func RegisterAWSMCPTools(ctx context.Context, client *AWSMCPClient, registry *tools.ToolRegistry) error {
	mcpTools, err := client.ListTools(ctx)
	if err != nil {
		return err
	}

	count := 0
	for _, mcpTool := range mcpTools {
		if mcpTool.Name == "" {
			continue
		}
		proxy := NewAWSMCPToolProxy(mcpTool, client)
		registry.Register(proxy)
		count++
		logger.DebugCF("mcp", "Registered AWS MCP tool",
			map[string]interface{}{
				"name": proxy.Name(),
				"mcp":  mcpTool.Name,
			})
	}

	logger.InfoCF("mcp", "Registered AWS MCP tools",
		map[string]interface{}{
			"count": count,
		})
	return nil
}
