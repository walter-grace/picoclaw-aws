//go:build ignore
// +build ignore

// Test S3 memory connectivity for pico-aws.
// Run: go run scripts/test_s3_memory.go
// Or: cd picoclaw-aws && go run ./scripts/test_s3_memory.go
//
// Requires the gateway to be running for the agent memory check.
// Start with: ./pico-aws gateway

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/walter-grace/picoclaw-aws/pkg/agent"
	"github.com/walter-grace/picoclaw-aws/pkg/config"
)

func main() {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".picoclaw", "config.json")

	// Load .env from same locations as gateway (AWS credentials)
	_ = godotenv.Load(filepath.Join(home, ".picoclaw", ".env"))
	_ = godotenv.Load(".env")

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load config from %s: %v\n", configPath, err)
		os.Exit(1)
	}

	memCfg := &cfg.Memory
	workspace := cfg.WorkspacePath()

	fmt.Println("🧊 pico-aws S3 Memory Test")
	fmt.Println("============================")
	fmt.Printf("Config: %s\n", configPath)
	fmt.Printf("Workspace: %s\n", workspace)
	fmt.Printf("Memory backend: %s\n", memCfg.Backend)
	if memCfg.S3.Bucket != "" {
		fmt.Printf("S3 bucket: %s\n", memCfg.S3.Bucket)
	} else if len(memCfg.S3.Buckets) > 0 {
		fmt.Printf("S3 buckets: %v (using %s for writes)\n", memCfg.S3.Buckets, memCfg.S3.Buckets[0])
	}
	fmt.Println()

	if memCfg.Backend != "s3" {
		fmt.Println("⚠️  Memory backend is not 's3'. Set memory.backend to 's3' and configure a bucket.")
		os.Exit(1)
	}

	store, err := agent.NewMemoryStoreFromConfig(workspace, memCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create S3 memory store: %v\n", err)
		fmt.Println("\nCheck:")
		fmt.Println("  1. memory.s3.bucket or memory.s3.buckets[0] is set in config")
		fmt.Println("  2. AWS credentials: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY (or ~/.picoclaw/.env)")
		fmt.Println("  3. Bucket exists and you have s3:GetObject, s3:PutObject permissions")
		os.Exit(1)
	}

	// Type assert to verify it's S3
	if _, ok := store.(*agent.S3MemoryStore); !ok {
		fmt.Println("❌ Expected S3MemoryStore but got filesystem fallback. Check config.")
		os.Exit(1)
	}

	fmt.Println("✓ S3 memory store created")

	// Test ReadLongTerm
	content := store.ReadLongTerm()
	fmt.Printf("✓ ReadLongTerm: %d bytes\n", len(content))
	if content != "" {
		preview := content
		if len(preview) > 120 {
			preview = preview[:120] + "..."
		}
		fmt.Printf("  Preview: %s\n", preview)
	} else {
		fmt.Println("  (empty - agent will have no long-term memory until you tell it to remember something)")
	}

	// Test WriteLongTerm round-trip (read, append test, write, verify, restore)
	original := content
	testMarker := "\n\n<!-- S3 memory test " + fmt.Sprintf("%d", os.Getpid()) + " -->"
	testContent := original + testMarker
	if err := store.WriteLongTerm(testContent); err != nil {
		fmt.Fprintf(os.Stderr, "❌ WriteLongTerm failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ WriteLongTerm succeeded")

	// Re-read to verify
	content2 := store.ReadLongTerm()
	if content2 != testContent {
		fmt.Fprintf(os.Stderr, "❌ Round-trip failed: content mismatch\n")
		os.Exit(1)
	}
	fmt.Println("✓ Round-trip read/write verified")

	// Restore original
	if err := store.WriteLongTerm(original); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Restore failed (memory may have test marker): %v\n", err)
	}
	fmt.Println("✓ Original memory restored")

	fmt.Println()
	fmt.Println("--- Agent memory check (gateway must be running) ---")

	gatewayURL := os.Getenv("CLAW_CUBED_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:18790"
	}

	client := &http.Client{Timeout: 60 * time.Second}

	// Check gateway health
	resp, err := client.Get(gatewayURL + "/health")
	if err != nil || resp.StatusCode != 200 {
		fmt.Printf("⚠️  Gateway not reachable at %s (start with: ./pico-aws gateway)\n", gatewayURL)
		fmt.Println("   Skipping agent memory check. S3 store tests passed.")
		fmt.Println()
		fmt.Println("✅ S3 memory store tests passed.")
		return
	}
	resp.Body.Close()

	// Ask the agent about its memory
	chatBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "What is your memory? Do you have access to your S3 memory? Answer briefly."},
		},
	}
	bodyBytes, _ := json.Marshal(chatBody)
	req, _ := http.NewRequest("POST", gatewayURL+"/api/chat", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	chatResp, chatErr := client.Do(req)
	if chatErr != nil {
		fmt.Printf("❌ Chat request failed: %v\n", chatErr)
		os.Exit(1)
	}
	defer chatResp.Body.Close()

	var chatResult struct {
		Content string `json:"content"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(chatResp.Body).Decode(&chatResult); err != nil {
		fmt.Printf("❌ Failed to decode chat response: %v\n", err)
		os.Exit(1)
	}
	if chatResp.StatusCode != 200 {
		fmt.Printf("❌ Chat API error (%d): %s\n", chatResp.StatusCode, chatResult.Error)
		os.Exit(1)
	}

	agentContent := strings.ToLower(chatResult.Content)

	// Negative: agent explicitly says it does NOT have S3 (avoid matching "not local disk")
	negativePhrases := []string{"don't have s3", "do not have s3", "s3 access: no", "memory is local files", "local files only", "stored locally only"}
	for _, phrase := range negativePhrases {
		if strings.Contains(agentContent, phrase) {
			fmt.Println("❌ Agent says it does NOT have S3 memory access.")
			fmt.Printf("  Response: %s\n", truncate(chatResult.Content, 400))
			fmt.Println("\n  Ensure: 1) Gateway was restarted after selecting bucket, 2) memory.backend=s3 in config")
			os.Exit(1)
		}
	}

	// Positive: agent confirms S3
	positivePhrases := []string{"s3 bucket", "stored in s3", "memory in s3", "cloud-backed", "pico-aws", "aws bucket"}
	found := false
	for _, phrase := range positivePhrases {
		if strings.Contains(agentContent, phrase) {
			found = true
			break
		}
	}

	if found {
		fmt.Println("✓ Agent confirmed S3 memory access")
		fmt.Printf("  Response preview: %s\n", truncate(chatResult.Content, 150))
	} else {
		fmt.Println("❌ Agent response does not clearly confirm S3. It may be using filesystem fallback.")
		fmt.Printf("  Response: %s\n", truncate(chatResult.Content, 400))
		fmt.Println("\n  Rebuild (go build) and restart the gateway after selecting a bucket in the Memory tab.")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✅ All S3 memory tests passed. Agent can reach S3 and confirms it.")
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
