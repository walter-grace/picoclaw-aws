// Package codemode generates a TypeScript-style API from MCP (and other) tools
// for use in Code Mode: the LLM sees an API and uses run_code to call it.
package codemode

import (
	"fmt"
	"strings"

	"github.com/bigneek/claw-cubed/pkg/tools"
)

const awsToolPrefix = "aws__"

// GenerateTypeScriptAPI produces a TypeScript API declaration string from tools
// in the registry. Only tools whose names start with awsToolPrefix (MCP) are
// included. Each tool becomes an async function with JSDoc from description
// and a simple (input: object) => Promise<unknown> signature.
func GenerateTypeScriptAPI(registry *tools.ToolRegistry) string {
	names := registry.List()
	var sb strings.Builder
	sb.WriteString("/**\n * Code Mode API: call these from your run_code script.\n * Use console.log() to output results to the agent.\n */\n\n")
	sb.WriteString("declare const api: {\n")
	for _, name := range names {
		if name == "run_code" {
			continue
		}
		// Include AWS MCP tools and shutdown_instance (instance self-management)
		if !strings.HasPrefix(name, awsToolPrefix) && name != "shutdown_instance" {
			continue
		}
		tool, ok := registry.Get(name)
		if !ok {
			continue
		}
		desc := tool.Description()
		params := tool.Parameters()
		// TypeScript-safe function name (replace - with _)
		jsName := strings.ReplaceAll(name, "-", "_")
		sb.WriteString(fmt.Sprintf("  /** %s */\n", escapeJSDoc(desc)))
		sb.WriteString(fmt.Sprintf("  %s(input: Record<string, unknown>): Promise<unknown>;\n", jsName))
		_ = params // could refine param types from JSON Schema later
	}
	sb.WriteString("};\n")
	return sb.String()
}

func escapeJSDoc(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "*/", "* /"), "\n", " ")
}
