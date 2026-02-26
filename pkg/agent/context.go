package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bigneek/claw-cubed/pkg/logger"
	"github.com/bigneek/claw-cubed/pkg/providers"
	"github.com/bigneek/claw-cubed/pkg/skills"
	"github.com/bigneek/claw-cubed/pkg/tools"
	"github.com/bigneek/claw-cubed/pkg/tools/codemode"
)

type ContextBuilder struct {
	workspace    string
	skillsLoader *skills.SkillsLoader
	memory       tools.MemoryStore
	tools        *tools.ToolRegistry // Direct reference to tool registry
	codeMode     bool                 // When true, prompt shows TypeScript API + run_code only
}

func getGlobalConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".picoclaw")
}

func NewContextBuilder(workspace string) *ContextBuilder {
	// builtin skills: skills directory in current project
	// Use the skills/ directory under the current working directory
	wd, _ := os.Getwd()
	builtinSkillsDir := filepath.Join(wd, "skills")
	globalSkillsDir := filepath.Join(getGlobalConfigDir(), "skills")

	return &ContextBuilder{
		workspace:    workspace,
		skillsLoader: skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir),
		memory:       NewFilesystemMemoryStore(workspace),
	}
}

// NewContextBuilderWithMemory creates a ContextBuilder with a custom MemoryStore (e.g. S3-backed).
func NewContextBuilderWithMemory(workspace string, memory tools.MemoryStore) *ContextBuilder {
	wd, _ := os.Getwd()
	builtinSkillsDir := filepath.Join(wd, "skills")
	globalSkillsDir := filepath.Join(getGlobalConfigDir(), "skills")

	return &ContextBuilder{
		workspace:    workspace,
		skillsLoader: skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir),
		memory:       memory,
	}
}

// SetToolsRegistry sets the tools registry for dynamic tool summary generation.
func (cb *ContextBuilder) SetToolsRegistry(registry *tools.ToolRegistry) {
	cb.tools = registry
}

// SetCodeMode enables or disables Code Mode (MCP as TypeScript API + single run_code tool).
func (cb *ContextBuilder) SetCodeMode(enabled bool) {
	cb.codeMode = enabled
}

func (cb *ContextBuilder) getIdentity() string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	workspacePath, _ := filepath.Abs(filepath.Join(cb.workspace))
	runtime := fmt.Sprintf("%s %s, Go %s", runtime.GOOS, runtime.GOARCH, runtime.Version())

	// Build tools section dynamically
	toolsSection := cb.buildToolsSection()

	// Detect S3, R2, or filesystem memory backend for accurate self-description
	usesS3 := false
	usesR2 := false
	memoryBucketName := ""
	switch m := cb.memory.(type) {
	case *S3MemoryStore:
		usesS3 = true
		memoryBucketName = m.BucketName()
	case *R2MemoryStore:
		usesR2 = true
		memoryBucketName = m.BucketName()
	case *ConfigurableMemoryStore:
		usesS3 = m.UsesS3()
		usesR2 = m.UsesR2()
		memoryBucketName = m.BucketName()
	}

	var identity, memoryDesc, memoryRule string
	if usesR2 {
		identity = `# Pico Claw AWS 🦞

You are Pico Claw AWS, an AI assistant with memory backed by Cloudflare R2. Your memory is durable, shared, and cloud-backed.

**When asked about your memory:** You MUST answer that your long-term memory is stored in Cloudflare R2 (object storage), not on local disk.`
		bucketLine := "Your memory is stored in a Cloudflare R2 bucket."
		if memoryBucketName != "" {
			bucketLine = fmt.Sprintf("Your memory bucket is: **%s** (Cloudflare R2).", memoryBucketName)
		}
		memoryDesc = fmt.Sprintf(`## Memory (Cloudflare R2)
%s
- Long-term: memory/MEMORY.md (in the bucket)
- Daily notes: memory/YYYYMM/YYYYMMDD.md (in the bucket)
- Skills: %s/skills/{skill-name}/SKILL.md (local)`, bucketLine, workspacePath)
		memoryRule = "4. **Memory** - When remembering something, write to memory/MEMORY.md (the tools will store it in R2)."
	} else if usesS3 {
		identity = `# pico-aws 🧊

You are pico-aws, an AWS-native AI assistant. Your memory lives in an S3 bucket—durable, shared, and cloud-backed.

**When asked about your memory or S3 access:** You MUST answer that your long-term memory is stored in an S3 bucket (cloud), not on local disk. Do not say "local files" or "filesystem"—you use S3.`
		bucketLine := "Your memory is stored in an S3 bucket."
		if memoryBucketName != "" {
			bucketLine = fmt.Sprintf("Your memory bucket is: **%s**. (When you list buckets via AWS, this one is your memory bucket.)", memoryBucketName)
		}
		memoryDesc = fmt.Sprintf(`## Memory (S3)
%s
- Long-term: memory/MEMORY.md (in the bucket)
- Daily notes: memory/YYYYMM/YYYYMMDD.md (in the bucket)
- Skills: %s/skills/{skill-name}/SKILL.md (local)
Use the list_memory_bucket tool to list bucket contents or count folders/projects.`, bucketLine, workspacePath)
		memoryRule = "4. **Memory** - When remembering something, write to memory/MEMORY.md (the tools will store it in S3)."
	} else {
		identity = `# pico-aws 🦞

You are pico-aws, a helpful AI assistant.`
		memoryDesc = fmt.Sprintf(`## Workspace
Your workspace is at: %s
- Memory: %s/memory/MEMORY.md
- Daily Notes: %s/memory/YYYYMM/YYYYMMDD.md
- Skills: %s/skills/{skill-name}/SKILL.md`, workspacePath, workspacePath, workspacePath, workspacePath)
		memoryRule = fmt.Sprintf("4. **Memory** - When remembering something, write to %s/memory/MEMORY.md", workspacePath)
	}

	return fmt.Sprintf(`%s

## Current Time
%s

## Runtime
%s

%s

%s

## Important Rules

1. **ALWAYS use tools** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), you MUST call the appropriate tool. Do NOT just say you'll do it or pretend to do it.

2. **Be helpful and accurate** - When using tools, briefly explain what you're doing.

3. **Code changes execute immediately** - When asked to fix or change code, use read_file then edit_file/write_file immediately. Don't ask "want me to do it?" or wait for confirmation—just do it. Run shell to verify (e.g. go build). You have limited iterations; use them for actions.

%s`,
		identity, now, runtime, memoryDesc, toolsSection, memoryRule)
}

func (cb *ContextBuilder) buildToolsSection() string {
	if cb.tools == nil {
		return ""
	}

	if cb.codeMode {
		apiDoc := codemode.GenerateTypeScriptAPI(cb.tools)
		return "## Available Tools (Code Mode)\n\n" +
			"You have **one** tool: **run_code**. Use it to run JavaScript that calls the API below. Use `console.log()` to output results.\n\n" +
			"**API (call these from your script via `api.<name>(args)`):**\n\n```typescript\n" + apiDoc + "```\n\n" +
			"**CRITICAL**: To use AWS or other MCP capabilities, call run_code with a script that uses the `api` object and logs the result.\n"
	}

	summaries := cb.tools.GetSummaries()
	if len(summaries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Available Tools\n\n")
	sb.WriteString("**CRITICAL**: You MUST use tools to perform actions. Do NOT pretend to execute commands or schedule tasks.\n\n")
	sb.WriteString("You have access to the following tools:\n\n")
	for _, s := range summaries {
		sb.WriteString(s)
		sb.WriteString("\n")
	}

	return sb.String()
}

func (cb *ContextBuilder) BuildSystemPrompt() string {
	parts := []string{}

	// Core identity section
	parts = append(parts, cb.getIdentity())

	// Bootstrap files
	bootstrapContent := cb.LoadBootstrapFiles()
	if bootstrapContent != "" {
		parts = append(parts, bootstrapContent)
	}

	// Skills - show summary, AI can read full content with read_file tool
	skillsSummary := cb.skillsLoader.BuildSkillsSummary()
	if skillsSummary != "" {
		parts = append(parts, fmt.Sprintf(`# Skills

The following skills extend your capabilities. To use a skill, read its SKILL.md file using the read_file tool.

%s`, skillsSummary))
	}

	// Memory context
	memoryContext := cb.memory.GetMemoryContext()
	if memoryContext != "" {
		parts = append(parts, "# Memory\n\n"+memoryContext)
	}

	// Join with "---" separator
	return strings.Join(parts, "\n\n---\n\n")
}

func (cb *ContextBuilder) LoadBootstrapFiles() string {
	usesS3 := false
	switch m := cb.memory.(type) {
	case *S3MemoryStore:
		usesS3 = true
	case *ConfigurableMemoryStore:
		usesS3 = m.UsesS3()
	}

	bootstrapFiles := []string{
		"AGENTS.md",
		"SOUL.md",
		"USER.md",
		"IDENTITY.md",
	}

	// When using S3 (pico-aws), skip IDENTITY.md and SOUL.md to avoid branding override
	if usesS3 {
		bootstrapFiles = []string{"AGENTS.md", "USER.md"}
	}

	var result string
	for _, filename := range bootstrapFiles {
		filePath := filepath.Join(cb.workspace, filename)
		if data, err := os.ReadFile(filePath); err == nil {
			result += fmt.Sprintf("## %s\n\n%s\n\n", filename, string(data))
		}
	}

	// Inject pico-aws identity when S3 (replaces IDENTITY.md + SOUL.md)
	if usesS3 {
		result += `## pico-aws Identity

You are pico-aws 🧊 — an AWS-native AI assistant. Your memory lives in S3. Use 🧊 when signing off or referring to yourself.`
	}

	return result
}

func (cb *ContextBuilder) BuildMessages(history []providers.Message, summary string, currentMessage string, media []string, channel, chatID string) []providers.Message {
	messages := []providers.Message{}

	systemPrompt := cb.BuildSystemPrompt()

	// Add Current Session info if provided
	if channel != "" && chatID != "" {
		systemPrompt += fmt.Sprintf("\n\n## Current Session\nChannel: %s\nChat ID: %s", channel, chatID)
	}

	// Log system prompt summary for debugging (debug mode only)
	logger.DebugCF("agent", "System prompt built",
		map[string]interface{}{
			"total_chars":   len(systemPrompt),
			"total_lines":   strings.Count(systemPrompt, "\n") + 1,
			"section_count": strings.Count(systemPrompt, "\n\n---\n\n") + 1,
		})

	// Log preview of system prompt (avoid logging huge content)
	preview := systemPrompt
	if len(preview) > 500 {
		preview = preview[:500] + "... (truncated)"
	}
	logger.DebugCF("agent", "System prompt preview",
		map[string]interface{}{
			"preview": preview,
		})

	if summary != "" {
		systemPrompt += "\n\n## Summary of Previous Conversation\n\n" + summary
	}

	//This fix prevents the session memory from LLM failure due to elimination of toolu_IDs required from LLM
	// --- INICIO DEL FIX ---
	//Diegox-17
	for len(history) > 0 && (history[0].Role == "tool") {
		logger.DebugCF("agent", "Removing orphaned tool message from history to prevent LLM error",
			map[string]interface{}{"role": history[0].Role})
		history = history[1:]
	}
	//Diegox-17
	// --- FIN DEL FIX ---

	messages = append(messages, providers.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	messages = append(messages, history...)

	messages = append(messages, providers.Message{
		Role:    "user",
		Content: currentMessage,
	})

	return messages
}

func (cb *ContextBuilder) AddToolResult(messages []providers.Message, toolCallID, toolName, result string) []providers.Message {
	messages = append(messages, providers.Message{
		Role:       "tool",
		Content:    result,
		ToolCallID: toolCallID,
	})
	return messages
}

func (cb *ContextBuilder) AddAssistantMessage(messages []providers.Message, content string, toolCalls []map[string]interface{}) []providers.Message {
	msg := providers.Message{
		Role:    "assistant",
		Content: content,
	}
	// Always add assistant message, whether or not it has tool calls
	messages = append(messages, msg)
	return messages
}

func (cb *ContextBuilder) loadSkills() string {
	allSkills := cb.skillsLoader.ListSkills()
	if len(allSkills) == 0 {
		return ""
	}

	var skillNames []string
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}

	content := cb.skillsLoader.LoadSkillsForContext(skillNames)
	if content == "" {
		return ""
	}

	return "# Skill Definitions\n\n" + content
}

// GetSkillsInfo returns information about loaded skills.
func (cb *ContextBuilder) GetSkillsInfo() map[string]interface{} {
	allSkills := cb.skillsLoader.ListSkills()
	skillNames := make([]string, 0, len(allSkills))
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}
	return map[string]interface{}{
		"total":     len(allSkills),
		"available": len(allSkills),
		"names":     skillNames,
	}
}
