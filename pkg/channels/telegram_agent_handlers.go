package channels

import (
	"context"
	"fmt"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

const spawnParallelPrompt = `Spawn 2 subagents in parallel:
1. Subagent A: List the workspace root (list_dir)
2. Subagent B: Count lines in workspace README or main.go if present

Use spawn for both. I'll get two separate messages when each completes.`

func (c *TelegramChannel) handleStatus(ctx context.Context, message *telego.Message) error {
	if c.agentDeps == nil || c.agentDeps.ListTasksByChat == nil {
		return c.sendPlain(ctx, message.Chat.ID, "Subagent status is not available.")
	}
	chatIDStr := fmt.Sprintf("%d", message.Chat.ID)
	tasks := c.agentDeps.ListTasksByChat("telegram", chatIDStr)
	if len(tasks) == 0 {
		return c.sendPlain(ctx, message.Chat.ID, "No subagent tasks.")
	}
	var sb strings.Builder
	sb.WriteString("📋 <b>Subagent tasks</b>\n\n")
	running := 0
	for _, t := range tasks {
		icon := "✅"
		if t.Status == "running" {
			icon = "🔄"
			running++
		} else if t.Status == "failed" {
			icon = "❌"
		}
		label := t.Label
		if label == "" {
			label = t.Task
		}
		sb.WriteString(fmt.Sprintf("%s <b>%s</b> — %s\n", icon, label, t.Status))
	}
	if running > 0 {
		sb.WriteString(fmt.Sprintf("\n%d running.", running))
	}
	return c.sendHTML(ctx, message.Chat.ID, sb.String())
}

func (c *TelegramChannel) handleSpawn(ctx context.Context, message *telego.Message) error {
	if c.agentDeps == nil || c.agentDeps.ProcessDirect == nil {
		return c.sendPlain(ctx, message.Chat.ID, "Spawn is not available.")
	}
	return c.processAgentPrompt(ctx, message, spawnParallelPrompt)
}

func (c *TelegramChannel) handleCustomSpawn(ctx context.Context, message *telego.Message) error {
	c.customSpawnMu.Lock()
	c.customSpawnMap[message.Chat.ID] = &customSpawnState{Tasks: nil}
	c.customSpawnMu.Unlock()

	text := "✏️ <b>Custom Subagents</b>\n\n" +
		"Send your tasks, <b>one per line</b>.\n\n" +
		"Example:\n" +
		"<code>List files in workspace/</code>\n" +
		"<code>Count lines in main.go</code>\n\n" +
		"Then send <b>/go</b> to spawn, or <b>/cancel</b> to abort."
	return c.sendHTML(ctx, message.Chat.ID, text)
}

func (c *TelegramChannel) handleGo(ctx context.Context, message *telego.Message) error {
	c.customSpawnMu.Lock()
	state := c.customSpawnMap[message.Chat.ID]
	delete(c.customSpawnMap, message.Chat.ID)
	c.customSpawnMu.Unlock()

	if state == nil || len(state.Tasks) == 0 {
		return c.sendPlain(ctx, message.Chat.ID, "No tasks. Send your tasks first (one per line), then /go.")
	}
	if c.agentDeps == nil || c.agentDeps.ProcessDirect == nil {
		return c.sendPlain(ctx, message.Chat.ID, "Spawn is not available.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Spawn %d subagents in parallel:\n", len(state.Tasks)))
	for i, t := range state.Tasks {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, t))
	}
	sb.WriteString("\nUse spawn for all. I'll get separate messages when each completes.")
	return c.processAgentPrompt(ctx, message, sb.String())
}

func (c *TelegramChannel) handleCancel(ctx context.Context, message *telego.Message) error {
	c.customSpawnMu.Lock()
	delete(c.customSpawnMap, message.Chat.ID)
	c.customSpawnMu.Unlock()

	return c.sendPlain(ctx, message.Chat.ID, "Cancelled.")
}

func (c *TelegramChannel) handleModel(ctx context.Context, message *telego.Message) error {
	if c.agentDeps == nil {
		return c.sendPlain(ctx, message.Chat.ID, "Model override is not available.")
	}
	text := strings.TrimSpace(message.Text)
	arg := strings.TrimSpace(strings.TrimPrefix(text, "/model"))

	chatIDStr := fmt.Sprintf("%d", message.Chat.ID)
	key := "telegram:" + chatIDStr

	if arg == "" {
		model := c.agentDeps.GetModelOverride(key)
		if model == "" && c.agentDeps.DefaultModel != nil {
			model = c.agentDeps.DefaultModel()
		}
		if model == "" {
			model = "default"
		}
		msg := fmt.Sprintf("🤖 <b>Model</b>: <code>%s</code>\n\nUse /model &lt;id&gt; to change, /model default to reset.", model)
		return c.sendHTML(ctx, message.Chat.ID, msg)
	}
	if strings.EqualFold(arg, "default") {
		c.agentDeps.SetModelOverride(key, "")
		model := "default"
		if c.agentDeps.DefaultModel != nil {
			model = c.agentDeps.DefaultModel()
		}
		return c.sendHTML(ctx, message.Chat.ID, fmt.Sprintf("Model reset to <code>%s</code>", model))
	}
	c.agentDeps.SetModelOverride(key, arg)
	return c.sendHTML(ctx, message.Chat.ID, fmt.Sprintf("Model set to <code>%s</code>. Next messages will use this model.", arg))
}

func (c *TelegramChannel) handleCallbackQuery(thCtx *telegohandler.Context, q telego.CallbackQuery) error {
	ctx := thCtx.Context()
	_ = c.bot.AnswerCallbackQuery(ctx, tu.CallbackQuery(q.ID))

	if q.Message == nil {
		return nil
	}
	chat := q.Message.GetChat()
	chatIDInt := chat.ID

	switch q.Data {
	case "spawn_parallel":
		msg := &telego.Message{Chat: chat, From: &q.From}
		return c.processAgentPrompt(ctx, msg, spawnParallelPrompt)
	case "custom_spawn":
		c.customSpawnMu.Lock()
		c.customSpawnMap[chat.ID] = &customSpawnState{Tasks: nil}
		c.customSpawnMu.Unlock()
		text := "✏️ <b>Custom Subagents</b>\n\nSend your tasks, <b>one per line</b>. Then /go or /cancel."
		return c.sendHTML(ctx, chatIDInt, text)
	case "custom_spawn_go":
		msg := &telego.Message{Chat: chat, From: &q.From}
		c.customSpawnMu.Lock()
		state := c.customSpawnMap[chat.ID]
		delete(c.customSpawnMap, chat.ID)
		c.customSpawnMu.Unlock()
		if state == nil || len(state.Tasks) == 0 {
			return c.sendPlain(ctx, chatIDInt, "No tasks. Send tasks first (one per line), then /go.")
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Spawn %d subagents:\n", len(state.Tasks)))
		for i, t := range state.Tasks {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, t))
		}
		sb.WriteString("\nUse spawn for all.")
		return c.processAgentPrompt(ctx, msg, sb.String())
	case "custom_spawn_cancel":
		c.customSpawnMu.Lock()
		delete(c.customSpawnMap, chat.ID)
		c.customSpawnMu.Unlock()
		return c.sendPlain(ctx, chatIDInt, "Cancelled.")
	default:
		return nil
	}
}

func (c *TelegramChannel) processAgentPrompt(ctx context.Context, message *telego.Message, prompt string) error {
	if c.agentDeps == nil || c.agentDeps.ProcessDirect == nil {
		return c.sendPlain(ctx, message.Chat.ID, "Agent is not available.")
	}
	userCtx := fmt.Sprintf("[User: %s (id: %d)", message.From.Username, message.From.ID)
	if message.From.FirstName != "" {
		userCtx += fmt.Sprintf(", name: %s", message.From.FirstName)
	}
	userCtx += "] " + prompt

	_, _ = c.bot.SendMessage(ctx, tu.Message(tu.ID(message.Chat.ID), "💭 Thinking..."))

	sessionKey := fmt.Sprintf("telegram:%d", message.Chat.ID)
	chatIDStr := fmt.Sprintf("%d", message.Chat.ID)
	reply, err := c.agentDeps.ProcessDirect(ctx, userCtx, sessionKey, "telegram", chatIDStr)
	if err != nil {
		return c.sendPlain(ctx, message.Chat.ID, fmt.Sprintf("Error: %v", err))
	}
	if reply == "" {
		reply = "(no response)"
	}
	return c.sendHTML(ctx, message.Chat.ID, reply)
}

func (c *TelegramChannel) handleStart(thCtx *telegohandler.Context, message *telego.Message) error {
	ctx := thCtx.Context()
	if c.agentDeps != nil {
		text := "👋 <b>pico-aws</b> — AWS-native agent with subagents.\n\n" +
			"• <b>Spawn 2</b> — Quick: list workspace + count lines\n" +
			"• <b>Custom</b> — Enter your own tasks (one per line)"
		btn1 := tu.InlineKeyboardButton("🚀 Spawn 2").WithCallbackData("spawn_parallel")
		btn2 := tu.InlineKeyboardButton("✏️ Custom Subagents").WithCallbackData("custom_spawn")
		markup := tu.InlineKeyboard(tu.InlineKeyboardRow(btn1, btn2))
		params := tu.Message(tu.ID(message.Chat.ID), text)
		params.ParseMode = telego.ModeHTML
		params.ReplyMarkup = markup
		_, err := c.bot.SendMessage(ctx, params)
		return err
	}
	return c.commands.Start(ctx, *message)
}

func (c *TelegramChannel) getAndSetCustomTasks(chatID int64, text string) (handled bool, tasks []string) {
	c.customSpawnMu.Lock()
	defer c.customSpawnMu.Unlock()

	state := c.customSpawnMap[chatID]
	if state == nil {
		return false, nil
	}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			tasks = append(tasks, line)
		}
	}

	if len(state.Tasks) > 0 {
		tasks = append(state.Tasks, tasks...)
	}
	if len(tasks) == 0 {
		return true, nil
	}

	c.customSpawnMap[chatID] = &customSpawnState{Tasks: tasks}
	return true, tasks
}

func (c *TelegramChannel) sendPlain(ctx context.Context, chatID int64, text string) error {
	_, err := c.bot.SendMessage(ctx, tu.Message(tu.ID(chatID), text))
	return err
}

func (c *TelegramChannel) sendHTML(ctx context.Context, chatID int64, html string) error {
	params := tu.Message(tu.ID(chatID), html)
	params.ParseMode = telego.ModeHTML
	_, err := c.bot.SendMessage(ctx, params)
	if err != nil {
		_, err = c.bot.SendMessage(ctx, tu.Message(tu.ID(chatID), html))
	}
	return err
}
