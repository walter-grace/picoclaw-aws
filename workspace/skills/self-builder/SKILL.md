---
name: self-builder
description: This agent's workspace is the Claw Cubed repo; use file tools, exec, and AWS MCP to extend the codebase and operate on AWS. Use when building, improving, or deploying this repo.
---

# Self-Builder

Your workspace is the **Claw Cubed repo** (this codebase). You can read/write code, run builds and tests via `exec`, and use AWS MCP tools (`aws__search_documentation`, `aws__call_api`, etc.) to extend the agent and operate on AWS.

## Guidelines

- **Tests and builds**: Before applying or suggesting large code changes, run tests (e.g. `go test ./...`) or build from repo root. Prefer small, verifiable steps.
- **AWS**: Use `aws__*` tools for infrastructure, APIs, and docs. Search documentation when needed; use `aws__call_api` for provisioning and operations.
- **Long or parallel work**: Use the `spawn` tool for time-consuming or independent tasks so the main loop stays responsive.
- **Structure**: Respect existing layout—`pkg/`, `cmd/`, `config/`, `workspace/`—and follow patterns already in the repo.

## Scope

Workspace = repo root. You may modify any file under the repo. `restrict_to_workspace` keeps exec and file tools scoped to this tree.
