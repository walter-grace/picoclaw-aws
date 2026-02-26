package channels

import "context"

// SubagentTaskInfo is a minimal view of a subagent task for /status display.
type SubagentTaskInfo struct {
	Label  string
	Status string
	Task   string
}

// TelegramAgentDeps provides the Telegram channel with agent capabilities
// (ProcessDirect, subagent status, model override). Injected after creation
// to avoid circular imports with the agent package.
type TelegramAgentDeps struct {
	// ProcessDirect runs a prompt through the agent and returns the response.
	ProcessDirect func(ctx context.Context, content, sessionKey, channel, chatID string) (string, error)
	// ListTasksByChat returns subagent tasks for the given channel/chat.
	ListTasksByChat func(channel, chatID string) []SubagentTaskInfo
	// SetModelOverride sets the LLM model for a channel:chatID key.
	SetModelOverride func(key, model string)
	// GetModelOverride returns the overridden model for a key.
	GetModelOverride func(key string) string
	// DefaultModel returns the config default model (for /model display).
	DefaultModel func() string
}
