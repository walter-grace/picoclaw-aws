package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bigneek/claw-cubed/pkg/config"
	"github.com/bigneek/claw-cubed/pkg/tools"
)

// ListMemoryBucketTool lets the agent list the current S3 memory bucket contents
// (memory files and top-level folders/projects) so it can answer questions like
// "how many projects are in our S3?"
type ListMemoryBucketTool struct {
	cfg *config.Config
}

// NewListMemoryBucketTool creates a tool that lists the configured memory bucket.
func NewListMemoryBucketTool(cfg *config.Config) *ListMemoryBucketTool {
	return &ListMemoryBucketTool{cfg: cfg}
}

func (t *ListMemoryBucketTool) Name() string {
	return "list_memory_bucket"
}

func (t *ListMemoryBucketTool) Description() string {
	return "List contents of your S3 memory bucket: memory files (MEMORY.md, daily notes) and top-level folders/projects in the bucket. Use this when the user asks how many projects or folders are in the bucket, or what's in your memory bucket."
}

func (t *ListMemoryBucketTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"include_folders": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, also list top-level folders (projects) in the bucket. Default true when user asks about projects or folders.",
			},
		},
		"required": []string{},
	}
}

func (t *ListMemoryBucketTool) Execute(ctx context.Context, args map[string]interface{}) *tools.ToolResult {
	t.cfg.RLock()
	backend := strings.ToLower(t.cfg.Memory.Backend)
	bucket := t.cfg.Memory.S3.Bucket
	if bucket == "" && len(t.cfg.Memory.S3.Buckets) > 0 {
		bucket = t.cfg.Memory.S3.Buckets[0]
	}
	t.cfg.RUnlock()

	if backend != "s3" || bucket == "" {
		return tools.NewToolResult("Memory is not using S3, or no bucket is configured. There is no S3 memory bucket to list.")
	}

	store, err := NewS3MemoryStore(S3MemoryStoreConfig{
		Bucket: bucket,
		Prefix: t.cfg.Memory.S3.Prefix,
		Region: t.cfg.Memory.S3.Region,
	})
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("Failed to connect to S3 memory bucket: %v", err))
	}

	runCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var out strings.Builder
	out.WriteString(fmt.Sprintf("**Memory bucket:** %s\n\n", bucket))

	// List memory files (MEMORY.md, daily notes)
	memObjs, err := store.ListMemoryObjects(runCtx)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("Failed to list memory files: %v", err))
	}
	out.WriteString("**Memory files:**\n")
	if len(memObjs) == 0 {
		out.WriteString("- (none yet)\n")
	} else {
		for _, o := range memObjs {
			out.WriteString(fmt.Sprintf("- %s (%d bytes)\n", o.Key, o.Size))
		}
	}

	includeFolders := true
	if v, ok := args["include_folders"].(bool); ok {
		includeFolders = v
	}
	if includeFolders {
		folders, err := store.ListTopLevelPrefixes(runCtx)
		if err != nil {
			out.WriteString(fmt.Sprintf("\n**Top-level folders:** (failed to list: %v)\n", err))
		} else {
			out.WriteString(fmt.Sprintf("\n**Top-level folders/projects in bucket:** %d\n", len(folders)))
			for _, name := range folders {
				out.WriteString(fmt.Sprintf("- %s\n", name))
			}
		}
	}

	return tools.NewToolResult(out.String())
}
