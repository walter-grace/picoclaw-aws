# Code Mode: MCP via Code Instead of Tool Calls

Code Mode is inspired by [Cloudflare's "Code Mode"](https://blog.cloudflare.com/code-mode-the-better-way-to-use-mcp): instead of exposing every MCP tool directly to the LLM as a tool, we convert the MCP tools into a **TypeScript/JavaScript API** and give the agent a single **run_code** tool. The LLM writes code that calls that API; the code runs in a **sandbox** whose only outbound access is through the MCP-backed API.

## Why Code Mode?

- **LLMs are better at writing code than at choosing tool calls** – They have seen huge amounts of real-world TypeScript/JavaScript; tool-call formats are more synthetic and narrow.
- **Efficiency** – Chaining multiple MCP calls no longer requires feeding each result back through the model; the script runs to completion and returns only the final `console.log` output.
- **Scale** – Many tools (e.g. large AWS APIs) fit into a compact API surface in code form instead of blowing up the tool list and context.

## How It Works Here

1. **MCP tools → TypeScript API**  
   We take the registered MCP tools (e.g. `aws__*`) and generate a TypeScript-style API: function names, JSDoc from descriptions, and parameter types from JSON Schema. This string is injected into the system prompt.

2. **Single tool: run_code**  
   The agent is presented with one tool: `run_code(code: string)`. The LLM outputs a call to `run_code` with a JavaScript snippet that uses the generated API.

3. **Sandbox**  
   The snippet runs in a [goja](https://github.com/dop251/goja) VM inside the process. The VM has:
   - No network or filesystem access.
   - A global object whose methods map to MCP (and optionally other) tools; each call is dispatched to the tool registry.
   - `console.log(...)` overridden to capture output; that output is returned as the tool result to the LLM.

So: **memory = one bucket; other buckets for media** is unchanged. Code Mode only changes **how** the agent uses MCP (and optionally other tools)—via code in a sandbox instead of direct tool calls.

## Config

```json
"tools": {
  "aws_mcp": { "enabled": true, "region": "us-east-1" },
  "code_mode": { "enabled": true }
}
```

- **code_mode.enabled** – When `true` and AWS MCP is enabled, the agent uses Code Mode: prompt includes the generated TypeScript API and only the `run_code` tool is sent to the LLM; MCP tools are still registered so they can be invoked from inside the sandbox.

## Security

- The sandbox is **isolated**: no network, no host filesystem. Only the injected API (tool dispatcher) is available.
- Execution is **time-limited** and **single-threaded** inside the VM.
- API keys and credentials stay in the host process; the script cannot access them except via the tool-call interface.

## Comparison to Cloudflare

| Aspect | Cloudflare | pico-aws (this) |
|--------|------------|-------------------|
| Sandbox | Workers / V8 isolates | goja VM in-process |
| API shape | TypeScript from MCP schema | TypeScript-style API from tool schemas |
| Single tool | Execute TypeScript | run_code(code) |
| MCP | Wrapped as TS API | Same; tools exposed as TS API, backed by MCP |

We use goja instead of Workers because we run as a single Go binary and avoid external runtimes (Node/containers) for simplicity and portability.
