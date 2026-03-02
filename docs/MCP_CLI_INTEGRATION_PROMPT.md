# Detailed Prompt: Integrating CLI Access Into Your MCP Server (Boomi Example)

Use this prompt to replicate the **pico-aws pattern**: an agent that has **both** MCP server tools (e.g. Boomi) **and** CLI tools (files, shell, web, etc.), with **graceful fallback** when the MCP server is unavailable.

---

## 1. Goal

- **Primary**: Expose your **Boomi MCP server** to an AI agent so it can call Boomi tools (APIs, flows, etc.).
- **Fallback**: When the Boomi MCP server is down or unreachable, the agent should still run in **CLI mode**—using only local/CLI tools (file read/write, shell exec, web fetch, etc.) and clearly inform the user: e.g. *"Boomi MCP unavailable; using CLI mode (CLI tools only)."*
- **Same UX**: One entrypoint (e.g. `my-agent agent` or your CLI command). No separate “MCP-only” vs “CLI-only” binaries; the same process tries MCP first, then continues with CLI if MCP fails.

Reference implementation: **pico-aws** ([picoclaw-aws](https://github.com/walter-grace/picoclaw-aws)), which does this for the **AWS MCP** server.

---

## 2. High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  CLI / Agent process (your app)                                 │
│  - Single entrypoint: e.g. "my-agent agent" or "boomi-agent"     │
└─────────────────────────────────────────────────────────────────┘
         │
         │ 1. Register CLI tools first (always available)
         │    - read_file, write_file, edit_file, list_dir
         │    - exec / shell (sandboxed)
         │    - web_search, web_fetch
         │    - message (if you have channels)
         │    - optional: spawn, cron, etc.
         │
         │ 2. If config says MCP enabled:
         │    - Create MCP client (stdio/SSE/command transport)
         │    - Connect to Boomi MCP server (e.g. npx boomi-mcp-server or your server URL)
         │    - List tools from MCP → register each as a proxy tool with prefix (e.g. boomi__)
         │    - On connection/list failure: log warning + "MCP unavailable, using CLI mode"
         │    - Do NOT exit; agent continues with CLI tools only
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│  Tool registry (single list for the LLM)                         │
│  - CLI tools: read_file, exec, web_fetch, ...                     │
│  - MCP proxy tools (if connected): boomi__invoke_flow, ...       │
└─────────────────────────────────────────────────────────────────┘
         │
         │ When user sends a message:
         │ - LLM chooses tools from registry
         │ - MCP tools → proxy calls through MCP client to Boomi server
         │ - CLI tools → run locally
         │
         ▼
┌──────────────────┐     ┌────────────────────────────────────────┐
│  Boomi MCP server │     │  Local execution (files, shell, web)   │
│  (when available)│     │  (always available)                    │
└──────────────────┘     └────────────────────────────────────────┘
```

---

## 3. Implementation Checklist (Replicate pico-aws Pattern)

### 3.1 Config

- Add a **Boomi MCP** (or generic “external MCP”) config block, similar to `tools.aws_mcp` in pico-aws:
  - **enabled** (bool): whether to try connecting to the Boomi MCP server.
  - **transport**: how to connect:
    - **stdio**: run a subprocess (e.g. `npx boomi-mcp-server`); MCP over stdin/stdout.
    - **command + args**: e.g. `proxy_command: "npx", proxy_args: ["boomi-mcp-server", "--config", "..."].`
  - Optional: **url** (if your Boomi MCP server is HTTP/SSE); then use SSE transport instead of stdio.

Example JSON config:

```json
{
  "tools": {
    "boomi_mcp": {
      "enabled": true,
      "proxy_command": "npx",
      "proxy_args": ["boomi-mcp-server", "--env", "production"]
    }
  }
}
```

- Env vars (optional): e.g. `BOOMI_MCP_ENABLED`, `BOOMI_MCP_PROXY_COMMAND`, `BOOMI_MCP_PROXY_ARGS` so you can override without editing config.

### 3.2 MCP Client (Generic or Boomi-Specific)

- **Connect**: Start the process (or open SSE URL), establish MCP session with timeout (e.g. 30s).
- **ListTools**: Call MCP `tools/list`; return list of tools with names, descriptions, input schemas.
- **CallTool**: Call MCP `tools/call` with tool name and arguments; return content (and whether it was an error).
- **On Connect/ListTools failure**: Return error; do **not** crash the process. The caller will log and continue without MCP tools.

Use the **Model Context Protocol** SDK for your language (e.g. Go: `github.com/modelcontextprotocol/go-sdk`, TypeScript: `@modelcontextprotocol/sdk`). Transport can be:

- **Stdio**: `exec.Command(proxyCommand, proxyArgs...)` with stdin/stdout.
- **SSE**: if Boomi MCP server exposes an HTTP endpoint.

### 3.3 Tool Registry and “CLI Tools First”

- Maintain a **single** tool registry that the LLM sees.
- **Registration order**:
  1. **Always** register CLI tools first: file ops, exec, web, message, etc.
  2. **Then**, if `tools.boomi_mcp.enabled`:
     - Create the Boomi MCP client.
     - Call `ListTools` (which may call `Connect` internally).
     - If error: log warning + one clear info line: *"Boomi MCP unavailable; using CLI mode (CLI tools only)."* Do **not** register any MCP tools; skip to step 3.
     - If success: for each MCP tool, register a **proxy tool** that implements your platform’s `Tool` interface and forwards `Execute` to `MCPClient.CallTool(name, args)`.
  3. Optionally register more tools (e.g. Code Mode, or other integrations).

- **Tool interface** (same idea as pico-aws): each tool has:
  - **Name**: e.g. `boomi__invoke_flow` (use a prefix like `boomi__` to avoid clashes with CLI tools).
  - **Description**: e.g. `"[Boomi] " + mcpTool.Description`.
  - **Parameters**: JSON Schema from MCP tool’s `inputSchema`.
  - **Execute(ctx, args)**: call `mcpClient.CallTool(ctx, mcpTool.Name, args)`; map result to your `ToolResult` (success/error, content for LLM/user).

### 3.4 Graceful MCP Failure (Default to CLI)

- **Where**: In the same place you register tools (e.g. “agent loop” or “gateway” startup), right after you try to register MCP tools.
- **If** `RegisterBoomiMCPTools(...)` (or equivalent) returns an error:
  - Log a **warning** with error and hint (e.g. “Ensure Boomi MCP server is running and reachable”).
  - Log a **single info line** that the user will see: *"Boomi MCP unavailable; using CLI mode (CLI tools only)."*
  - **Do not** exit, **do not** clear already-registered CLI tools. The agent continues with the existing (CLI-only) registry.
- Optional: track “MCP connected” in a flag so that optional features (e.g. “Code Mode” that exposes MCP as a TypeScript API) are only enabled when MCP is actually available.

### 3.5 CLI Entrypoint and Logging

- One command to run the agent (e.g. `my-agent agent` or `boomi-agent`).
- At startup, after tool registration:
  - Log that the agent is ready, and optionally how many tools are available (CLI + MCP if any).
- In interactive mode, show a clear banner, e.g. *"Boomi agent — Interactive mode (Ctrl+C to exit)."*
- If MCP failed, the user already saw *"MCP unavailable; using CLI mode"* so they know they only have CLI tools.

---

## 4. Code-Level Pattern (from pico-aws)

### 4.1 Registration (pseudo-code)

```text
// 1. CLI tools (always)
registry.Register(NewReadFileTool(...))
registry.Register(NewWriteFileTool(...))
registry.Register(NewExecTool(...))
registry.Register(NewWebSearchTool(...))
registry.Register(NewWebFetchTool(...))
// ... message, spawn, etc.

// 2. Boomi MCP (optional; on failure → CLI only)
if config.Tools.BoomiMCP.Enabled {
    client := NewBoomiMCPClient(config.Tools.BoomiMCP)
    if err := RegisterBoomiMCPTools(ctx, client, registry); err != nil {
        logger.Warn("Boomi MCP tools not available (connection failed)", "error", err)
        logger.Info("Boomi MCP unavailable; using CLI mode (CLI tools only)")
    }
}

// 3. Optional: Code Mode / other integrations that depend on MCP
if config.Tools.CodeMode.Enabled && boomiMCPConnected {
    registry.Register(NewRunCodeTool(registry))
}
```

### 4.2 MCP Client Contract

- **NewBoomiMCPClient(cfg)** → client.
- **client.Connect(ctx)** → error (e.g. timeout, subprocess failed, or SSE failed).
- **client.ListTools(ctx)** → ([]Tool, error); may call Connect if not yet connected.
- **client.CallTool(ctx, name string, args map[string]interface{})** → (content string, isError bool, err error).
- **client.Close()** (optional): close transport/session.

### 4.3 Proxy Tool (per MCP tool)

- **Name**: `boomi__` + mcpTool.Name.
- **Description**: `"[Boomi] " + mcpTool.Description`.
- **Parameters**: mcpTool.InputSchema (JSON Schema).
- **Execute(ctx, args)**:
  - Call `client.CallTool(ctx, mcpTool.Name, args)`.
  - If err != nil → return ErrorResult(err).
  - If isError → return ErrorResult(content).
  - Else → return SuccessResult(content) (or SilentResult for LLM-only).

---

## 5. Boomi-Specific Notes

- **Boomi MCP server**: Ensure it speaks standard MCP (tools/list, tools/call). If it’s a custom API, add a thin MCP adapter that exposes Boomi operations as MCP tools.
- **Transport**: If Boomi provides an MCP server as a Node app, use `proxy_command: "npx", proxy_args: ["your-boomi-mcp-package"]`. If it’s HTTP/SSE, use the MCP SDK’s SSE client and set `url` in config.
- **Auth**: Pass credentials via env vars or config (e.g. `BOOMI_CLIENT_ID`, `BOOMI_CLIENT_SECRET`) so the Boomi MCP server (or your proxy) can authenticate; the CLI agent only runs the process or connects to the URL.
- **Prefix**: Use a consistent prefix (e.g. `boomi__`) so the LLM and logs clearly show which tools are from Boomi vs CLI.

---

## 6. Summary Table (AWS vs Boomi)

| Aspect              | pico-aws (AWS MCP)                    | Your Boomi integration                |
|---------------------|----------------------------------------|----------------------------------------|
| Config block        | `tools.aws_mcp`                        | `tools.boomi_mcp` (or `tools.mcp`)     |
| Transport           | stdio via `uvx mcp-proxy-for-aws@...` | stdio (npx/boomi-mcp-server) or SSE   |
| Tool prefix         | `aws__`                                | `boomi__`                              |
| CLI tools           | file, exec, web, message, spawn, etc.  | Same idea; register first               |
| On MCP failure      | Log warning + “MCP unavailable, using CLI mode” | Same                    |
| Single entrypoint   | `pico-aws agent`                       | e.g. `boomi-agent agent` or your CLI   |

---

## 7. Quick Reference: pico-aws Files

- **Tool registration and MCP fallback**: `pkg/agent/loop.go` — `registerSharedTools()`: CLI tools first, then `if cfg.Tools.AWSMCP.Enabled { ... RegisterAWSMCPTools(...); on err log and continue }`.
- **MCP client**: `pkg/tools/mcp/client.go` — Connect (stdio), ListTools, CallTool.
- **MCP proxy tools**: `pkg/tools/mcp/aws_tool.go` — NewAWSMCPToolProxy, RegisterAWSMCPTools (list tools, register each with prefix).
- **Config**: `pkg/config/config.go` — `AWSMCPConfig`; `config.example.json` — `tools.aws_mcp`.
- **CLI entrypoint**: `cmd/picoclaw/main.go` — `agent` command, banner “pico-aws agent — Interactive mode”.

Using this prompt, you can integrate your Boomi MCP server with CLI fallback in the same way pico-aws integrates the AWS MCP server.
