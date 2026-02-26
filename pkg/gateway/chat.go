package gateway

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/bigneek/claw-cubed/pkg/agent"
)

// ChatHandler provides an HTTP API for chat compatible with Vercel AI SDK useChat.
// POST /api/chat with JSON body: { "messages": [{ "role": "user", "content": "..." }] }
// Returns JSON: { "content": "..." }
func ChatHandler(agentLoop *agent.AgentLoop) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		var userContent string
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == "user" {
				userContent = req.Messages[i].Content
				break
			}
		}
		if userContent == "" {
			http.Error(w, "No user message in messages", http.StatusBadRequest)
			return
		}

		sessionKey := "web:default"
		if id := r.Header.Get("X-Session-ID"); id != "" {
			sessionKey = "web:" + id
		}

		response, usage, duration, iterations, err := agentLoop.ProcessDirectWithMetrics(r.Context(), userContent, sessionKey, "web", "default")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{"content": response}
		if usage != nil {
			resp["usage"] = map[string]interface{}{
				"prompt_tokens":     usage.PromptTokens,
				"completion_tokens": usage.CompletionTokens,
				"total_tokens":      usage.TotalTokens,
				"prompt_tokens_details": map[string]int{
					"cached_tokens": usage.PromptTokensDetails.CachedTokens,
				},
				"completion_tokens_details": map[string]int{
					"reasoning_tokens": usage.CompletionTokensDetails.ReasoningTokens,
				},
			}
		}
		resp["duration_ms"] = duration.Milliseconds()
		resp["iterations"] = iterations
		json.NewEncoder(w).Encode(resp)
	}
}

// ChatStreamHandler returns SSE stream for useChat compatibility (optional).
// For now we use the simple JSON response; streaming can be added later.
func ChatStreamHandler(agentLoop *agent.AgentLoop) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		var userContent string
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == "user" {
				userContent = req.Messages[i].Content
				break
			}
		}
		if userContent == "" {
			http.Error(w, "No user message in messages", http.StatusBadRequest)
			return
		}

		sessionKey := "web:default"
		if id := r.Header.Get("X-Session-ID"); id != "" {
			sessionKey = "web:" + id
		}

		response, err := agentLoop.ProcessDirect(r.Context(), userContent, sessionKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// SSE format for AI SDK compatibility
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Vercel-AI-Data-Stream", "v1")

		// Send text delta events
		for _, chunk := range splitIntoChunks(response, 50) {
			writeSSE(w, "0", chunk)
		}
		writeSSE(w, "d", `{"finishReason":"stop"}`)
	}
}

func splitIntoChunks(s string, size int) []string {
	var chunks []string
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	if len(chunks) == 0 {
		chunks = []string{""}
	}
	return chunks
}

func writeSSE(w http.ResponseWriter, id, data string) {
	escaped := strings.ReplaceAll(data, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	w.Write([]byte("id: " + id + "\n"))
	w.Write([]byte("data: " + escaped + "\n\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
