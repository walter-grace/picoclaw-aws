package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/walter-grace/picoclaw-aws/pkg/agent"
	"github.com/walter-grace/picoclaw-aws/pkg/config"
)

const memoryPreviewMaxLen = 500

// ProfileHandler returns a handler for GET /api/profile.
func ProfileHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	picoClawDir := filepath.Dir(configPath)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		cfg.RLock()
		mem := cfg.Memory
		awsMCP := cfg.Tools.AWSMCP
		cfg.RUnlock()

		// Credentials can come from ~/.picoclaw/.env, project .env, or system env
		credsConfigured := os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != ""
		if !credsConfigured {
			credsConfigured = hasAWSEnvFile(picoClawDir)
		}
		awsRegion := os.Getenv("AWS_REGION")
		if awsRegion == "" {
			awsRegion = getAWSEnvRegion(picoClawDir)
		}
		if awsRegion == "" {
			awsRegion = awsMCP.Region
		}
		if awsRegion == "" {
			awsRegion = "us-east-1"
		}

		resp := map[string]interface{}{
			"aws_credentials": map[string]interface{}{
				"configured": credsConfigured,
				"region":     awsRegion,
			},
			"aws_mcp": map[string]interface{}{
				"enabled": awsMCP.Enabled,
				"region":  awsMCP.Region,
			},
			"memory": map[string]interface{}{
				"backend": mem.Backend,
				"s3": map[string]interface{}{
					"bucket":  mem.S3.Bucket,
					"buckets": mem.S3.Buckets,
					"prefix":  mem.S3.Prefix,
					"region":  mem.S3.Region,
				},
			},
		}

		if strings.ToLower(mem.Backend) == "s3" && mem.S3.Bucket != "" {
			s3Cfg := agent.S3MemoryStoreConfig{
				Bucket: mem.S3.Bucket,
				Prefix: mem.S3.Prefix,
				Region: mem.S3.Region,
			}
			store, err := agent.NewS3MemoryStore(s3Cfg)
			if err == nil {
				ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
				defer cancel()

				objects, err := store.ListMemoryObjects(ctx)
				if err == nil {
					// Convert to JSON-serializable format
					var files []map[string]interface{}
					for _, o := range objects {
						files = append(files, map[string]interface{}{
							"key":           o.Key,
							"size":          o.Size,
							"last_modified": o.LastModified,
						})
					}
					resp["s3_files"] = files
				}

				preview := store.ReadLongTerm()
				if len(preview) > memoryPreviewMaxLen {
					preview = preview[:memoryPreviewMaxLen] + "..."
				}
				resp["memory_preview"] = preview
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func getAWSEnvRegion(dir string) string {
	path := filepath.Join(dir, ".env")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "AWS_REGION=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "AWS_REGION="))
		}
	}
	return ""
}
