# pico-aws

**The AWS-native tiny agent.** An improved variant of [PicoClaw](https://github.com/sipeed/picoclaw) built for the cloud. Agent-only by default (no web UI)—minimal footprint; run on an instance and prompt it to self-build (e.g. add a UI) later.

## What Makes It Different

| PicoClaw | pico-aws |
|----------|----------|
| Local `MEMORY.md` file | **S3 bucket** – cloud-backed memory |
| AWS MCP optional | **AWS MCP enabled by default** |
| Single instance | Multi-instance, shared memory |
| Local-first | Cloud-first, serverless-ready |

## Quick Start

1. **Create an S3 bucket** for memory:
   ```bash
   aws s3 mb s3://your-pico-aws-memory --region us-east-1
   ```

2. **Set your bucket** and credentials:
   ```bash
   export PICOCLAW_MEMORY_S3_BUCKET=your-pico-aws-memory
   export AWS_ACCESS_KEY_ID=...
   export AWS_SECRET_ACCESS_KEY=...
   export AWS_REGION=us-east-1
   ```

3. **Build and run**:
   ```bash
   go build -o pico-aws ./cmd/picoclaw
   ./pico-aws agent
   ```

Memory goes to S3. No local `MEMORY.md`. The agent also has AWS MCP tools (API calls, docs search) enabled by default.

## New in this release

- **Code Mode** – Use MCP via a single `run_code` tool: the LLM gets a TypeScript API and writes JavaScript that runs in a sandbox (goja). Fewer round-trips and lower latency for multi-step AWS tasks. Enable in config: `"tools": { "code_mode": { "enabled": true } }`. See [CODE_MODE.md](CODE_MODE.md).
- **Chat API with metrics** – `POST /api/chat` returns `content`, `usage` (prompt/completion/total tokens), `duration_ms`, and `iterations` for each request (useful for benchmarking and monitoring).
- **Code Mode benchmark** – Compare Code Mode vs normal tool-call mode: run `./scripts/run_benchmark_test.sh` (starts the gateway twice with code_mode off/on and prints duration and iteration comparison). Or run `go run scripts/benchmark_code_mode.go --label tool_calls` and again with `--label code_mode` after toggling config.
- **Config path override** – Set `PICOCLAW_CONFIG_PATH` to use a specific config file (e.g. for tests or multiple environments).
- **Test env** – `.env.test` documents test env and gateway URL (port 18790). Source it or copy from `.env` when running benchmarks.

## Requirements

- Go 1.21+
- AWS credentials (`aws configure` or env vars)
- LLM API key (OpenRouter, OpenAI, Anthropic, etc.)
- For AWS MCP: `uvx` (pip/uv) for the proxy

## Docs

- [ONE_PAGER.md](ONE_PAGER.md) – Positioning and vision
- [CODE_MODE.md](CODE_MODE.md) – Code Mode: MCP as TypeScript API + `run_code` sandbox
- [SELF_BUILDER.md](SELF_BUILDER.md) – Run as a self-building repo agent (e.g. on EC2)
- [deploy/README.md](deploy/README.md) – **Attach this folder to a launched instance** (production run script and setup)
- [TEST_MEMORY.md](TEST_MEMORY.md) – Memory testing guide
- [docs/AWS_DEPLOYMENT_ONEPAGER.md](docs/AWS_DEPLOYMENT_ONEPAGER.md) – Deploy on AWS

## License

MIT
