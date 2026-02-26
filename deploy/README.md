# Deploy: Attach This Folder to a Launched Instance

Use this when your app **launches an EC2 (or other) instance** and **attaches the pico-aws repo** to it. On the instance, run from the repo root to start the **agent only** (CLI). No web UI—minimal footprint; the agent can self-build later (e.g. add a UI or API).

## What to attach

Attach the **whole pico-aws folder** (this repo) to the instance—e.g. copy the repo, or mount it, or unpack a tarball at a path like `/opt/pico-aws` or `$HOME/pico-aws`. The run script must be executed from the **repo root** (the directory that contains `cmd/`, `config/`, `deploy/`, `workspace/`).

## Prerequisites on the instance

- **Go 1.21+** – to build the binary (or pre-build and ship the binary so run.sh only runs it).
- **AWS credentials** – `aws configure`, IAM role, or env vars (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`).
- **uv/uvx** – for AWS MCP: `curl -LsSf https://astral.sh/uv/install.sh | sh`.
- **LLM API key** – set via env or in config (OpenRouter, Anthropic, OpenAI, etc.).

## Config

Config is read from **`~/.picoclaw/config.json`**. On first run:

1. Create the dir: `mkdir -p ~/.picoclaw`
2. Copy and edit: `cp config/config.example.json ~/.picoclaw/config.json`
3. Set your providers, S3 bucket, and `tools.aws_mcp.enabled: true` as needed.

For the **self-building repo agent**, use the repo agent + self-builder skill and set `workspace` to the repo path on the instance (e.g. `"."` when running from repo root). See [SELF_BUILDER.md](../SELF_BUILDER.md).

## Env vars (production)

Set these (in `.env`, systemd, or your launcher):

- **AWS**: `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` (or use IAM role).
- **S3 memory**: `PICOCLAW_MEMORY_S3_BUCKET=your-bucket` (and optional `PICOCLAW_MEMORY_S3_PREFIX`, `PICOCLAW_MEMORY_S3_REGION`).
- **Cloudflare R2 memory**: When using `memory.backend: "r2"`, set `CLOUDFLARE_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, and `memory.r2.bucket` in config.
- **LLM**: e.g. `PICOCLAW_PROVIDERS_OPENROUTER_API_KEY=sk-or-v1-...` or provider-specific keys.

See [../.env.example](../.env.example) and [.env.example](.env.example) for a full list.

## Deploy Telegram bot with Cloudflare R2 memory

To run the Telegram bot with memory backed by Cloudflare R2 (instead of S3 or filesystem):

1. **Create R2 bucket** – In Cloudflare dashboard: R2 → Create bucket (e.g. `pico-flare`). Or use PicoFlare's `mcp-test` to create via MCP.
2. **Create R2 API token** – R2 → Manage R2 API Tokens → Create token with Object Read & Write.
3. **Config** – In `~/.picoclaw/config.json`:
   ```json
   {
     "memory": {
       "backend": "r2",
       "r2": {
         "account_id": "YOUR_CLOUDFLARE_ACCOUNT_ID",
         "bucket": "pico-flare",
         "prefix": ""
       }
     },
     "channels": {
       "telegram": {
         "enabled": true,
         "token": "YOUR_TELEGRAM_BOT_TOKEN",
         "allow_from": ["YOUR_USER_ID"]
       }
     }
   }
   ```
4. **Env vars** – Set `CLOUDFLARE_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY` (or put in config).
5. **Run** – `./deploy/run.sh gateway` (for HTTP API) or `./deploy/run.sh agent` (CLI + Telegram polling).

## EC2 shutdown (optional)

To let the agent shut down the instance it runs on (e.g. "please shut down this instance"):

1. **IAM** – Attach a policy with `ec2:StopInstances` to the instance role (or credentials).
2. **Config** – In `~/.picoclaw/config.json`:
   ```json
   "tools": {
     "ec2_shutdown": { "enabled": true }
   }
   ```
3. The agent will use IMDS to discover its instance ID. For testing off-EC2, set `PICOCLAW_EC2_INSTANCE_ID=i-xxxxx`.

## Run

From the **repo root** (e.g. after `cd /opt/pico-aws`):

```bash
chmod +x deploy/run.sh
./deploy/run.sh          # agent only (default)
# or
./deploy/run.sh gateway  # agent + HTTP API (for channels / future UI)
```

- **agent** (default): CLI-only. Interactive or `pico-aws agent -m "Build yourself"`. Smallest footprint.
- **gateway**: agent plus HTTP API (health, /api/chat, etc.) for Telegram/Discord or a future UI you add when self-building.

## Summary for your launcher app

1. Attach this folder (pico-aws repo) to the launched instance at a known path.
2. Ensure Go, AWS creds, uv, and LLM keys are available (env or config).
3. Create `~/.picoclaw/config.json` from `config/config.example.json` and edit.
4. From repo root: `./deploy/run.sh` (agent only).

After that, prompt the agent via CLI (e.g. “build yourself”); it can extend the repo and add a UI or API later.
