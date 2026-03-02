package agent

import (
	"strings"
	"sync"

	"github.com/walter-grace/picoclaw-aws/pkg/config"
	"github.com/walter-grace/picoclaw-aws/pkg/tools"
)

// ConfigurableMemoryStore delegates to the appropriate MemoryStore (S3 or filesystem)
// based on the current config. When the user changes the bucket in the UI and saves,
// the agent uses the new bucket on the next memory access—no restart required.
type ConfigurableMemoryStore struct {
	workspace string
	cfg       *config.Config
	store     tools.MemoryStore
	cacheKey  string // "s3:bucket" or "fs" to detect config changes
	mu        sync.RWMutex
}

// NewConfigurableMemoryStore creates a memory store that reads from config at runtime.
func NewConfigurableMemoryStore(workspace string, cfg *config.Config) *ConfigurableMemoryStore {
	return &ConfigurableMemoryStore{
		workspace: workspace,
		cfg:       cfg,
	}
}

func (c *ConfigurableMemoryStore) getStore() tools.MemoryStore {
	c.mu.Lock()
	defer c.mu.Unlock()

	cfg := c.cfg
	cfg.RLock()
	backend := strings.ToLower(cfg.Memory.Backend)
	bucket := cfg.Memory.S3.Bucket
	if bucket == "" && len(cfg.Memory.S3.Buckets) > 0 {
		bucket = cfg.Memory.S3.Buckets[0]
	}
	r2Bucket := cfg.Memory.R2.Bucket

	newKey := "fs"
	if backend == "s3" && bucket != "" {
		newKey = "s3:" + bucket
	} else if backend == "r2" && r2Bucket != "" {
		newKey = "r2:" + r2Bucket
	}

	if c.store != nil && c.cacheKey == newKey {
		cfg.RUnlock()
		return c.store
	}

	store, err := NewMemoryStoreFromConfig(c.workspace, &cfg.Memory)
	cfg.RUnlock()
	if err != nil {
		store = NewFilesystemMemoryStore(c.workspace)
		newKey = "fs"
	}
	c.store = store
	c.cacheKey = newKey
	return c.store
}

func (c *ConfigurableMemoryStore) ReadLongTerm() string {
	return c.getStore().ReadLongTerm()
}

func (c *ConfigurableMemoryStore) WriteLongTerm(content string) error {
	return c.getStore().WriteLongTerm(content)
}

func (c *ConfigurableMemoryStore) ReadToday() string {
	return c.getStore().ReadToday()
}

func (c *ConfigurableMemoryStore) AppendToday(content string) error {
	return c.getStore().AppendToday(content)
}

func (c *ConfigurableMemoryStore) ReadDailyNote(date string) string {
	return c.getStore().ReadDailyNote(date)
}

func (c *ConfigurableMemoryStore) WriteDailyNote(date, content string) error {
	return c.getStore().WriteDailyNote(date, content)
}

func (c *ConfigurableMemoryStore) GetRecentDailyNotes(days int) string {
	return c.getStore().GetRecentDailyNotes(days)
}

func (c *ConfigurableMemoryStore) GetMemoryContext() string {
	return c.getStore().GetMemoryContext()
}

// BucketName returns the current memory bucket name when using S3 or R2, or empty string.
func (c *ConfigurableMemoryStore) BucketName() string {
	c.cfg.RLock()
	defer c.cfg.RUnlock()
	backend := strings.ToLower(c.cfg.Memory.Backend)
	if backend == "s3" {
		bucket := c.cfg.Memory.S3.Bucket
		if bucket == "" && len(c.cfg.Memory.S3.Buckets) > 0 {
			bucket = c.cfg.Memory.S3.Buckets[0]
		}
		return bucket
	}
	if backend == "r2" {
		return c.cfg.Memory.R2.Bucket
	}
	return ""
}

// UsesS3 returns true if the current config uses S3 (for context builder identity).
func (c *ConfigurableMemoryStore) UsesS3() bool {
	c.cfg.RLock()
	defer c.cfg.RUnlock()
	return strings.ToLower(c.cfg.Memory.Backend) == "s3" && (c.cfg.Memory.S3.Bucket != "" || len(c.cfg.Memory.S3.Buckets) > 0)
}

// UsesR2 returns true if the current config uses Cloudflare R2.
func (c *ConfigurableMemoryStore) UsesR2() bool {
	c.cfg.RLock()
	defer c.cfg.RUnlock()
	return strings.ToLower(c.cfg.Memory.Backend) == "r2" && c.cfg.Memory.R2.Bucket != ""
}
