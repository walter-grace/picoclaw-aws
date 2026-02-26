package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

const shutdownInstanceName = "shutdown_instance"

// ShutdownInstanceTool stops the EC2 instance the agent is running on.
// Uses IMDS to discover the instance ID when on EC2, or PICOCLAW_EC2_INSTANCE_ID env var.
// Requires IAM permission: ec2:StopInstances.
type ShutdownInstanceTool struct{}

// NewShutdownInstanceTool creates a tool that can stop the current EC2 instance.
func NewShutdownInstanceTool() *ShutdownInstanceTool {
	return &ShutdownInstanceTool{}
}

// Name returns the tool name.
func (t *ShutdownInstanceTool) Name() string {
	return shutdownInstanceName
}

// Description returns the tool description for the LLM.
func (t *ShutdownInstanceTool) Description() string {
	return "Stop (shut down) the EC2 instance this agent is running on. Use only when the user explicitly asks to shut down, stop, or power off the instance. The instance will stop; you can start it again later from the AWS console. Requires running on EC2 with IAM permission ec2:StopInstances."
}

// Parameters returns the JSON Schema for the tool.
func (t *ShutdownInstanceTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"confirm": map[string]interface{}{
				"type":        "string",
				"description": "Must be exactly 'yes' to confirm shutdown. Prevents accidental execution.",
			},
		},
		"required": []string{"confirm"},
	}
}

// Execute stops the current EC2 instance.
func (t *ShutdownInstanceTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	confirm, _ := args["confirm"].(string)
	if confirm != "yes" {
		return ErrorResult("shutdown_instance requires confirm='yes' to prevent accidental shutdown. Ask the user to confirm first.")
	}

	instanceID := os.Getenv("PICOCLAW_EC2_INSTANCE_ID")
	if instanceID == "" {
		// Try IMDS (only works on EC2)
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to load AWS config: %v", err))
		}
		imdsClient := imds.NewFromConfig(cfg)
		doc, err := imdsClient.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
		if err != nil {
			return ErrorResult(fmt.Sprintf("not running on EC2 or IMDS unavailable: %v. Set PICOCLAW_EC2_INSTANCE_ID for testing.", err))
		}
		instanceID = doc.InstanceID
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to load AWS config: %v", err))
	}
	ec2Client := ec2.NewFromConfig(cfg)

	_, err = ec2Client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return ErrorResult(fmt.Sprintf("StopInstances failed: %v. Ensure IAM has ec2:StopInstances.", err))
	}

	msg := fmt.Sprintf("Instance %s is stopping. It may take a minute to fully stop. You can start it again from the AWS console.", instanceID)
	return UserResult(msg)
}
