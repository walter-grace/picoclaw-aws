# OpenRouter Setup (Recommended for AWS Tools)

OpenRouter gives access to powerful models (Claude, GPT-4, etc.) with strong tool-calling support—ideal for pico-aws AWS MCP integration.

## 1. Get an API Key

1. Go to [https://openrouter.ai/keys](https://openrouter.ai/keys)
2. Sign up or log in
3. Add credits (required for usage)
4. Create an API key

## 2. Configure

**Option A: Environment variable (recommended)**

Add to `.env` in the picoclaw directory:

```
PICOCLAW_PROVIDERS_OPENROUTER_API_KEY=sk-or-v1-your-key-here
```

**Option B: Config file**

Edit `~/.picoclaw/config.json` and add your key:

```json
"providers": {
  "openrouter": {
    "api_key": "sk-or-v1-your-key-here",
    "api_base": "https://openrouter.ai/api/v1"
  }
}
```

## 3. Model

The config uses `openrouter/moonshotai/kimi-k2.5` (Kimi K2.5) by default—strong agentic tool-calling, 262K context. Other options:

- `openrouter/anthropic/claude-3.5-sonnet` – Claude 3.5 Sonnet
- `openrouter/openai/gpt-4o` – GPT-4o
- `openrouter/openai/gpt-4o-mini` – Cheaper, still capable
- `openrouter/anthropic/claude-3.5-haiku` – Faster, cheaper
- `openrouter/google/gemini-pro-1.5` – Gemini

Change in `~/.picoclaw/config.json` under `agents.defaults.model`.

## 4. Run

```bash
cd /Users/bigneek/Desktop/aws-pico/picoclaw
PATH="$HOME/.local/bin:$PATH" ./build/picoclaw-darwin-arm64 agent -m "What AWS regions are available?"
```

## Fallback to Ollama

If OpenRouter fails (no key or no credits), you can switch back to Ollama:

In `~/.picoclaw/config.json`:
```json
"model": "ollama/llama3.2:3b"
```

And ensure Ollama has `api_key: "ollama"` and `api_base: "http://localhost:11434/v1"`.
