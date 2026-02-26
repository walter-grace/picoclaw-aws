//go:build ignore
// +build ignore

// Benchmark Code Mode vs normal tool-call mode: same prompt, compare tokens and latency.
//
// Usage:
//
//  1. Start gateway with code_mode OFF (in config: "tools": { "code_mode": { "enabled": false }, "aws_mcp": { "enabled": true } }):
//     ./picoclaw-aws gateway  (or go run ./cmd/picoclaw gateway)
//  2. Run: go run scripts/benchmark_code_mode.go --label tool_calls
//  3. Stop gateway, set code_mode.enabled to true, start gateway again.
//  4. Run: go run scripts/benchmark_code_mode.go --label code_mode
//  5. Compare duration_ms, total_tokens, iterations, and response length.
//
// To save both for comparison:
//   go run scripts/benchmark_code_mode.go --label tool_calls --json > tool_calls.json
//   go run scripts/benchmark_code_mode.go --label code_mode --json > code_mode.json
//
// Run from the repo root. Gateway must be running on port 18790.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
)

func main() {
	url := flag.String("url", "http://localhost:18790", "Gateway base URL")
	label := flag.String("label", "", "Label for this run (e.g. 'tool_calls' or 'code_mode')")
	prompt := flag.String("prompt", "List my S3 buckets and tell me how many there are. Be brief.", "User prompt to send")
	outputJSON := flag.Bool("json", false, "Output machine-readable JSON only")
	flag.Parse()

	if *label == "" {
		fmt.Fprintf(os.Stderr, "Usage: go run scripts/benchmark_code_mode.go --label <tool_calls|code_mode> [--url URL] [--prompt PROMPT]\n")
		fmt.Fprintf(os.Stderr, "  Run once with code_mode off (--label tool_calls), then with code_mode on (--label code_mode), then compare.\n")
		os.Exit(1)
	}

	body := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": *prompt},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", *url+"/api/chat", bytes.NewReader(bodyBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	// Unique session per benchmark run - prevents history from contaminating token counts
	req.Header.Set("X-Session-ID", fmt.Sprintf("bench-%s-%d", *label, rand.Int63()))

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	clientDuration := time.Since(start)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result struct {
		Content    string `json:"content"`
		DurationMs int64  `json:"duration_ms"`
		Iterations int    `json:"iterations"`
		Usage      struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
			// Per-category breakdowns (OpenRouter / OpenAI)
			PromptTokensDetails struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
			CompletionTokensDetails struct {
				ReasoningTokens int `json:"reasoning_tokens"`
			} `json:"completion_tokens_details"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode, result.Content)
		os.Exit(1)
	}

	if *outputJSON {
		out := map[string]interface{}{
			"label":                   *label,
			"duration_ms":             result.DurationMs,
			"client_wall_ms":          clientDuration.Milliseconds(),
			"prompt_tokens":           result.Usage.PromptTokens,
			"completion_tokens":       result.Usage.CompletionTokens,
			"total_tokens":            result.Usage.TotalTokens,
			"cached_prompt_tokens":    result.Usage.PromptTokensDetails.CachedTokens,
			"reasoning_tokens":        result.Usage.CompletionTokensDetails.ReasoningTokens,
			"iterations":              result.Iterations,
			"response_length":         len(result.Content),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
		return
	}

	fmt.Println("--- Code Mode benchmark ---")
	fmt.Printf("Label:    %s\n", *label)
	fmt.Printf("Prompt:   %s\n", *prompt)
	fmt.Printf("Duration (server): %d ms\n", result.DurationMs)
	fmt.Printf("Duration (client): %d ms\n", clientDuration.Milliseconds())
	fmt.Printf("Tokens:   prompt=%d  completion=%d  total=%d\n",
		result.Usage.PromptTokens, result.Usage.CompletionTokens, result.Usage.TotalTokens)
	if result.Usage.PromptTokensDetails.CachedTokens > 0 || result.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
		fmt.Printf("          cached_prompt=%d  reasoning=%d\n",
			result.Usage.PromptTokensDetails.CachedTokens,
			result.Usage.CompletionTokensDetails.ReasoningTokens)
	}
	if result.Usage.TotalTokens == 0 {
		fmt.Println("  [!] total_tokens=0 — provider did not return usage; check model/OpenRouter plan")
	}
	fmt.Printf("Iterations: %d\n", result.Iterations)
	fmt.Printf("Response length: %d chars\n", len(result.Content))
	fmt.Println("----------------------------")
	fmt.Println("Run again with the other mode (code_mode on/off) and compare duration_ms, total_tokens, iterations.")
}
