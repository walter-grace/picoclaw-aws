package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/walter-grace/picoclaw-aws/pkg/config"
)

// ConfigHandler returns handlers for GET/PATCH /api/config (memory, tools.aws_mcp, aws_credentials).
func ConfigHandler(cfg *config.Config, configPath string) (get, patch http.HandlerFunc) {
	picoClawDir := filepath.Dir(configPath)

	get = func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		cfg.RLock()
		mem := cfg.Memory
		awsMCP := cfg.Tools.AWSMCP
		cfg.RUnlock()

		s3Map := map[string]interface{}{
			"bucket":  mem.S3.Bucket,
			"buckets": mem.S3.Buckets,
			"prefix":  mem.S3.Prefix,
			"region":  mem.S3.Region,
		}
		if s3Map["buckets"] == nil {
			s3Map["buckets"] = []string{}
		}
		resp := map[string]interface{}{
			"memory": map[string]interface{}{
				"backend": mem.Backend,
				"s3":     s3Map,
			},
			"tools": map[string]interface{}{
				"aws_mcp": map[string]interface{}{
					"enabled": awsMCP.Enabled,
					"region":  awsMCP.Region,
				},
			},
			"aws_credentials_configured": os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" || hasAWSEnvFile(picoClawDir),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	patch = func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch && r.Method != http.MethodPut && r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			Memory *struct {
				Backend string `json:"backend"`
				S3      *struct {
					Bucket  string   `json:"bucket"`
					Buckets []string `json:"buckets"`
					Prefix  string   `json:"prefix"`
					Region  string   `json:"region"`
				} `json:"s3"`
			} `json:"memory"`
			Tools *struct {
				AWSMCP *struct {
					Enabled *bool  `json:"enabled"`
					Region  string `json:"region"`
				} `json:"aws_mcp"`
			} `json:"tools"`
			AwsCredentials *struct {
				AccessKeyID     string `json:"access_key_id"`
				SecretAccessKey string `json:"secret_access_key"`
				Region          string `json:"region"`
			} `json:"aws_credentials"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		cfg.Lock()

		if body.Memory != nil {
			if body.Memory.Backend != "" {
				cfg.Memory.Backend = body.Memory.Backend
			}
			if body.Memory.S3 != nil {
				cfg.Memory.S3.Bucket = body.Memory.S3.Bucket
				cfg.Memory.S3.Buckets = body.Memory.S3.Buckets
				if len(cfg.Memory.S3.Buckets) > 0 && cfg.Memory.S3.Bucket == "" {
					cfg.Memory.S3.Bucket = cfg.Memory.S3.Buckets[0]
				}
				cfg.Memory.S3.Prefix = body.Memory.S3.Prefix
				cfg.Memory.S3.Region = body.Memory.S3.Region
			}
		}

		if body.Tools != nil && body.Tools.AWSMCP != nil {
			if body.Tools.AWSMCP.Enabled != nil {
				cfg.Tools.AWSMCP.Enabled = *body.Tools.AWSMCP.Enabled
			}
			region := body.Tools.AWSMCP.Region
			if region == "" {
				region = "us-east-1"
			}
			cfg.Tools.AWSMCP.Region = region
			cfg.Tools.AWSMCP.ProxyArgs = []string{
				"mcp-proxy-for-aws@latest",
				"https://aws-mcp.us-east-1.api.aws/mcp",
				"--metadata", "AWS_REGION=" + region,
			}
		}

		cfg.Unlock()

		if err := config.SaveConfig(configPath, cfg); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if body.AwsCredentials != nil {
			if err := saveAWSEnvFile(picoClawDir, body.AwsCredentials.AccessKeyID, body.AwsCredentials.SecretAccessKey, body.AwsCredentials.Region); err != nil {
				http.Error(w, "Failed to save AWS credentials: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Restart the gateway to apply changes"})
	}

	return get, patch
}

func hasAWSEnvFile(dir string) bool {
	path := filepath.Join(dir, ".env")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	s := string(data)
	return strings.Contains(s, "AWS_ACCESS_KEY_ID=") && strings.Contains(s, "AWS_SECRET_ACCESS_KEY=")
}

func saveAWSEnvFile(dir, accessKey, secretKey, region string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, ".env")

	// Parse existing .env: preserve non-AWS vars, overlay AWS vars we're updating
	existing := make(map[string]string)
	var otherLines []string
	if data, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if idx := strings.Index(line, "="); idx > 0 {
				k, v := line[:idx], line[idx+1:]
				if strings.HasPrefix(k, "AWS_") {
					existing[k] = v
				} else {
					otherLines = append(otherLines, line)
				}
			}
		}
	}

	if accessKey != "" {
		existing["AWS_ACCESS_KEY_ID"] = accessKey
	}
	if secretKey != "" {
		existing["AWS_SECRET_ACCESS_KEY"] = secretKey
	}
	if region != "" {
		existing["AWS_REGION"] = region
	}

	var lines []string
	for _, k := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION"} {
		if v, ok := existing[k]; ok && v != "" {
			lines = append(lines, fmt.Sprintf("%s=%s", k, v))
		}
	}
	lines = append(lines, otherLines...)
	if len(lines) == 0 {
		return nil
	}

	data := []byte(strings.Join(lines, "\n") + "\n")
	return os.WriteFile(path, data, 0600)
}
