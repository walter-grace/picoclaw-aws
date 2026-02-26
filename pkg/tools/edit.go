package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// EditFileTool edits a file by replacing old_text with new_text.
// The old_text must exist exactly in the file.
type EditFileTool struct {
	allowedDir  string
	restrict    bool
	memoryStore MemoryStore
}

// NewEditFileTool creates a new EditFileTool with optional directory restriction.
func NewEditFileTool(allowedDir string, restrict bool) *EditFileTool {
	return &EditFileTool{allowedDir: allowedDir, restrict: restrict, memoryStore: nil}
}

// NewEditFileToolWithMemory creates an EditFileTool that delegates memory path edits to MemoryStore.
func NewEditFileToolWithMemory(allowedDir string, restrict bool, memoryStore MemoryStore) *EditFileTool {
	return &EditFileTool{allowedDir: allowedDir, restrict: restrict, memoryStore: memoryStore}
}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Description() string {
	return "Edit a file by replacing old_text with new_text. The old_text must exist exactly in the file."
}

func (t *EditFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The file path to edit",
			},
			"old_text": map[string]interface{}{
				"type":        "string",
				"description": "The exact text to find and replace",
			},
			"new_text": map[string]interface{}{
				"type":        "string",
				"description": "The text to replace with",
			},
		},
		"required": []string{"path", "old_text", "new_text"},
	}
}

func (t *EditFileTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	oldText, ok := args["old_text"].(string)
	if !ok {
		return ErrorResult("old_text is required")
	}

	newText, ok := args["new_text"].(string)
	if !ok {
		return ErrorResult("new_text is required")
	}

	resolvedPath, err := validatePath(path, t.allowedDir, t.restrict)
	if err != nil {
		return ErrorResult(err.Error())
	}

	var contentStr string
	if t.memoryStore != nil {
		if relPath, isMemory := getMemoryPathRel(t.allowedDir, resolvedPath); isMemory {
			if relPath == "memory/MEMORY.md" {
				contentStr = t.memoryStore.ReadLongTerm()
			} else {
				date := parseMemoryDailyDate(relPath)
				contentStr = t.memoryStore.ReadDailyNote(date)
			}
			if contentStr == "" {
				return ErrorResult(fmt.Sprintf("file not found: %s", path))
			}
			if !strings.Contains(contentStr, oldText) {
				return ErrorResult("old_text not found in file. Make sure it matches exactly")
			}
			if strings.Count(contentStr, oldText) > 1 {
				return ErrorResult("old_text appears multiple times. Please provide more context to make it unique")
			}
			newContent := strings.Replace(contentStr, oldText, newText, 1)
			if relPath == "memory/MEMORY.md" {
				if err := t.memoryStore.WriteLongTerm(newContent); err != nil {
					return ErrorResult(fmt.Sprintf("failed to write memory: %v", err))
				}
			} else {
				date := parseMemoryDailyDate(relPath)
				if err := t.memoryStore.WriteDailyNote(date, newContent); err != nil {
					return ErrorResult(fmt.Sprintf("failed to write daily note: %v", err))
				}
			}
			return SilentResult(fmt.Sprintf("File edited: %s", path))
		}
	}

	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return ErrorResult(fmt.Sprintf("file not found: %s", path))
	}

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read file: %v", err))
	}

	contentStr = string(content)

	if !strings.Contains(contentStr, oldText) {
		return ErrorResult("old_text not found in file. Make sure it matches exactly")
	}

	count := strings.Count(contentStr, oldText)
	if count > 1 {
		return ErrorResult(fmt.Sprintf("old_text appears %d times. Please provide more context to make it unique", count))
	}

	newContent := strings.Replace(contentStr, oldText, newText, 1)

	if err := os.WriteFile(resolvedPath, []byte(newContent), 0644); err != nil {
		return ErrorResult(fmt.Sprintf("failed to write file: %v", err))
	}

	return SilentResult(fmt.Sprintf("File edited: %s", path))
}

type AppendFileTool struct {
	workspace   string
	restrict    bool
	memoryStore MemoryStore
}

func NewAppendFileTool(workspace string, restrict bool) *AppendFileTool {
	return &AppendFileTool{workspace: workspace, restrict: restrict, memoryStore: nil}
}

func NewAppendFileToolWithMemory(workspace string, restrict bool, memoryStore MemoryStore) *AppendFileTool {
	return &AppendFileTool{workspace: workspace, restrict: restrict, memoryStore: memoryStore}
}

func (t *AppendFileTool) Name() string {
	return "append_file"
}

func (t *AppendFileTool) Description() string {
	return "Append content to the end of a file"
}

func (t *AppendFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The file path to append to",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The content to append",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *AppendFileTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return ErrorResult("content is required")
	}

	resolvedPath, err := validatePath(path, t.workspace, t.restrict)
	if err != nil {
		return ErrorResult(err.Error())
	}

	if t.memoryStore != nil {
		if relPath, isMemory := getMemoryPathRel(t.workspace, resolvedPath); isMemory {
			if relPath == "memory/MEMORY.md" {
				existing := t.memoryStore.ReadLongTerm()
				newContent := existing + "\n" + content
				if err := t.memoryStore.WriteLongTerm(newContent); err != nil {
					return ErrorResult(fmt.Sprintf("failed to append to memory: %v", err))
				}
			} else {
				date := parseMemoryDailyDate(relPath)
				if date == time.Now().Format("20060102") {
					if err := t.memoryStore.AppendToday(content); err != nil {
						return ErrorResult(fmt.Sprintf("failed to append to daily note: %v", err))
					}
				} else {
					existing := t.memoryStore.ReadDailyNote(date)
					newContent := existing + "\n" + content
					if err := t.memoryStore.WriteDailyNote(date, newContent); err != nil {
						return ErrorResult(fmt.Sprintf("failed to append to daily note: %v", err))
					}
				}
			}
			return SilentResult(fmt.Sprintf("Appended to %s", path))
		}
	}

	f, err := os.OpenFile(resolvedPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to open file: %v", err))
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return ErrorResult(fmt.Sprintf("failed to append to file: %v", err))
	}

	return SilentResult(fmt.Sprintf("Appended to %s", path))
}
