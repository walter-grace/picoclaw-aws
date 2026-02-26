package codemode

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/bigneek/claw-cubed/pkg/logger"
	"github.com/bigneek/claw-cubed/pkg/tools"
)

const runCodeTimeout = 30 * time.Second
const runCodeName = "run_code"

// RunCodeTool runs user-provided JavaScript in a goja VM. The VM has an `api`
// object whose methods map to registry tools (excluding run_code). Results
// are returned via console.log().
type RunCodeTool struct {
	registry *tools.ToolRegistry
}

// NewRunCodeTool creates a run_code tool that dispatches API calls to the given registry.
func NewRunCodeTool(registry *tools.ToolRegistry) *RunCodeTool {
	return &RunCodeTool{registry: registry}
}

// Name returns the tool name.
func (t *RunCodeTool) Name() string {
	return runCodeName
}

// Description returns the tool description for the LLM.
func (t *RunCodeTool) Description() string {
	return "Run JavaScript in a sandbox. You have an `api` object: each key is a tool name (e.g. api.aws__list_buckets). Call with an object of arguments; returns a Promise with the result. Use console.log() to output results. No network or filesystem—only the api.* methods are available."
}

// Parameters returns the JSON Schema for the tool.
func (t *RunCodeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"code": map[string]interface{}{
				"type":        "string",
				"description": "JavaScript code to run. Use api.<tool_name>(args) for tool calls and console.log() for output.",
			},
		},
		"required": []string{"code"},
	}
}

// Execute runs the provided code in a goja VM and returns captured console output.
func (t *RunCodeTool) Execute(ctx context.Context, args map[string]interface{}) *tools.ToolResult {
	code, _ := args["code"].(string)
	code = strings.TrimSpace(code)
	if code == "" {
		return tools.ErrorResult("run_code requires a non-empty 'code' string")
	}

	runCtx, cancel := context.WithTimeout(ctx, runCodeTimeout)
	defer cancel()

	vm := goja.New()
	var logLines []string
	// Capture console.log
	logFn := func(call goja.FunctionCall) goja.Value {
		var parts []string
		for _, v := range call.Arguments {
			parts = append(parts, valueToString(v))
		}
		logLines = append(logLines, strings.Join(parts, " "))
		return goja.Undefined()
	}
	consoleObj := vm.NewObject()
	_ = consoleObj.Set("log", logFn)
	_ = vm.Set("console", consoleObj)

	// Expose api.* as calls to the tool registry (excluding run_code)
	apiObj := vm.NewObject()
	for _, name := range t.registry.List() {
		if name == runCodeName {
			continue
		}
		toolName := name
		fn := func(call goja.FunctionCall) goja.Value {
			argsObj := call.Argument(0)
			argsMap := make(map[string]interface{})
			if argsObj != nil && !goja.IsNull(argsObj) && !goja.IsUndefined(argsObj) {
				if o, ok := argsObj.Export().(map[string]interface{}); ok {
					argsMap = o
				} else if o := argsObj.ToObject(vm); o != nil {
					for _, k := range o.Keys() {
						v := o.Get(k)
						argsMap[k] = exportValue(v)
					}
				}
			}
			result := t.registry.Execute(runCtx, toolName, argsMap)
			// Return result (including error message) so the script can log or check it
			return vm.ToValue(result.ForLLM)
		}
		jsName := strings.ReplaceAll(toolName, "-", "_")
		_ = apiObj.Set(jsName, fn)
	}
	_ = vm.Set("api", apiObj)

	// Run user code (wrap in async IIFE if we want to support await later)
	script := code
	if !strings.Contains(script, "await") {
		script = "(function() { " + code + "\n })();"
	} else {
		script = "(async function() { " + code + "\n })();"
	}
	_, err := vm.RunString(script)
	if err != nil {
		logger.WarnCF("codemode", "run_code execution failed",
			map[string]interface{}{"error": err.Error()})
		return tools.ErrorResult(fmt.Sprintf("run_code error: %v", err))
	}

	output := strings.Join(logLines, "\n")
	if output == "" {
		output = "(no console output)"
	}
	return tools.SilentResult(output)
}

func valueToString(v goja.Value) string {
	if v == nil || goja.IsNull(v) || goja.IsUndefined(v) {
		return ""
	}
	return v.String()
}

func exportValue(v goja.Value) interface{} {
	if v == nil || goja.IsNull(v) || goja.IsUndefined(v) {
		return nil
	}
	return v.Export()
}
