# Self-Building Repo Agent

pico-aws can run as a **self-building agent**: one repo and one **repo agent** whose workspace is this repository, with **AWS MCP** and file/shell tools enabled. The agent can modify the codebase and use AWS (APIs, docs, infra) to extend itself.

## Concept

- **Unique repo**: This repository is the agent’s workspace.
- **Unique repo agent**: A dedicated agent (e.g. `id: "repo"`) with `workspace: "."` and the **self-builder** skill.
- **Tools**: File tools (read/write/list/edit/append), `exec` (sandboxed to workspace), `spawn`, and when enabled, AWS MCP tools (`aws__search_documentation`, `aws__call_api`, etc.).

Run the process **from repo root** so that workspace `"."` resolves to this directory.

## Prerequisites

- **Go** (to build/run and for `go test` / `go build`).
- **AWS credentials**: `aws configure` or `aws login`; IAM should allow `aws-mcp:InvokeMcp`, `aws-mcp:CallReadOnlyTool`, `aws-mcp:CallReadWriteTool` as needed.
- **uv/uvx**: For AWS MCP proxy. Install: `curl -LsSf https://astral.sh/uv/install.sh | sh`.
- **LLM provider**: Configure at least one provider (e.g. OpenRouter, Anthropic) in `config.json` with a valid API key.

## Setup

1. **Config**: Use an agent that has this repo as its workspace and AWS MCP enabled.
   - Add `agents.list` with one agent, e.g. `id: "repo"`, `default: true`, `workspace: "."`, `skills: ["self-builder"]`.
   - Set `tools.aws_mcp.enabled: true` and set `tools.aws_mcp.region` (and optional `proxy_command` / `proxy_args`) as needed.
   - You can copy from `config/config.example.json` (which includes the repo agent and AWS MCP) or merge the snippet from `config/self-builder.example.json`.

2. **Run from repo root** so `"."` is the repo:
   - `./picoclaw agent` or `go run ./cmd/picoclaw agent` (from the repo root).

3. **Optional – two agents**: Keep a general `main` agent (default workspace) and add a second agent `repo` with `workspace: "."` and the self-builder skill. Use **bindings** to route a specific channel or CLI context to `repo` when you want self-building.

## Safety

- With **workspace = repo root**, the agent can change any file and run commands inside the repo. Use a **dedicated clone or branch** so you can review or revert changes.
- **`restrict_to_workspace: true`** (default) limits file and exec tools to the workspace directory; the agent cannot escape the repo. No code change is required—just run with this default.

## See also

- [PICOCLAW_ARCHITECTURE.md](PICOCLAW_ARCHITECTURE.md) §11 – AWS MCP integration.
- [workspace/TOOLS.md](workspace/TOOLS.md) – Built-in and AWS MCP tool list.
- [workspace/skills/self-builder/SKILL.md](workspace/skills/self-builder/SKILL.md) – Self-builder skill text.
