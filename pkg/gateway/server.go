package gateway

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bigneek/claw-cubed/pkg/agent"
	"github.com/bigneek/claw-cubed/pkg/config"
	"github.com/bigneek/claw-cubed/pkg/health"
)

// NewServer creates an HTTP server with health checks, chat API, and config API.
func NewServer(host string, port int, agentLoop *agent.AgentLoop, healthServer *health.Server, cfg *config.Config, configPath string) *http.Server {
	mux := http.NewServeMux()
	healthServer.RegisterRoutes(mux)
	mux.HandleFunc("/api/chat", ChatHandler(agentLoop))
	mux.HandleFunc("/api/chat/stream", ChatStreamHandler(agentLoop))
	mux.HandleFunc("/api/s3/buckets", S3BucketsHandler())

	if cfg != nil && configPath != "" {
		getH, patchH := ConfigHandler(cfg, configPath)
		mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				getH(w, r)
			case http.MethodPatch, http.MethodPut, http.MethodPost:
				patchH(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})
		mux.HandleFunc("/api/profile", ProfileHandler(cfg, configPath))
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	return &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
}
