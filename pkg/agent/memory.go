// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bigneek/claw-cubed/pkg/config"
	"github.com/bigneek/claw-cubed/pkg/tools"
)

// FilesystemMemoryStore manages persistent memory on the local filesystem.
// - Long-term memory: memory/MEMORY.md
// - Daily notes: memory/YYYYMM/YYYYMMDD.md
type FilesystemMemoryStore struct {
	workspace  string
	memoryDir  string
	memoryFile string
}

// NewFilesystemMemoryStore creates a new FilesystemMemoryStore with the given workspace path.
// It ensures the memory directory exists.
func NewFilesystemMemoryStore(workspace string) *FilesystemMemoryStore {
	memoryDir := filepath.Join(workspace, "memory")
	memoryFile := filepath.Join(memoryDir, "MEMORY.md")

	// Ensure memory directory exists
	os.MkdirAll(memoryDir, 0755)

	return &FilesystemMemoryStore{
		workspace:  workspace,
		memoryDir:  memoryDir,
		memoryFile: memoryFile,
	}
}

// getTodayFile returns the path to today's daily note file (memory/YYYYMM/YYYYMMDD.md).
func (ms *FilesystemMemoryStore) getTodayFile() string {
	today := time.Now().Format("20060102") // YYYYMMDD
	monthDir := today[:6]                  // YYYYMM
	filePath := filepath.Join(ms.memoryDir, monthDir, today+".md")
	return filePath
}

// ReadLongTerm reads the long-term memory (MEMORY.md).
// Returns empty string if the file doesn't exist.
func (ms *FilesystemMemoryStore) ReadLongTerm() string {
	if data, err := os.ReadFile(ms.memoryFile); err == nil {
		return string(data)
	}
	return ""
}

// WriteLongTerm writes content to the long-term memory file (MEMORY.md).
func (ms *FilesystemMemoryStore) WriteLongTerm(content string) error {
	return os.WriteFile(ms.memoryFile, []byte(content), 0644)
}

// ReadToday reads today's daily note.
// Returns empty string if the file doesn't exist.
func (ms *FilesystemMemoryStore) ReadToday() string {
	todayFile := ms.getTodayFile()
	if data, err := os.ReadFile(todayFile); err == nil {
		return string(data)
	}
	return ""
}

// AppendToday appends content to today's daily note.
// If the file doesn't exist, it creates a new file with a date header.
func (ms *FilesystemMemoryStore) AppendToday(content string) error {
	todayFile := ms.getTodayFile()

	// Ensure month directory exists
	monthDir := filepath.Dir(todayFile)
	os.MkdirAll(monthDir, 0755)

	var existingContent string
	if data, err := os.ReadFile(todayFile); err == nil {
		existingContent = string(data)
	}

	var newContent string
	if existingContent == "" {
		// Add header for new day
		header := fmt.Sprintf("# %s\n\n", time.Now().Format("2006-01-02"))
		newContent = header + content
	} else {
		// Append to existing content
		newContent = existingContent + "\n" + content
	}

	return os.WriteFile(todayFile, []byte(newContent), 0644)
}

// ReadDailyNote reads a daily note for the given date (YYYYMMDD).
func (ms *FilesystemMemoryStore) ReadDailyNote(date string) string {
	if len(date) != 8 {
		return ""
	}
	monthDir := date[:6]
	filePath := filepath.Join(ms.memoryDir, monthDir, date+".md")
	if data, err := os.ReadFile(filePath); err == nil {
		return string(data)
	}
	return ""
}

// WriteDailyNote overwrites a daily note for the given date (YYYYMMDD).
func (ms *FilesystemMemoryStore) WriteDailyNote(date, content string) error {
	if len(date) != 8 {
		return fmt.Errorf("invalid date format: %s (expected YYYYMMDD)", date)
	}
	monthDir := date[:6]
	filePath := filepath.Join(ms.memoryDir, monthDir, date+".md")
	dir := filepath.Dir(filePath)
	os.MkdirAll(dir, 0755)
	return os.WriteFile(filePath, []byte(content), 0644)
}

// GetRecentDailyNotes returns daily notes from the last N days.
// Contents are joined with "---" separator.
func (ms *FilesystemMemoryStore) GetRecentDailyNotes(days int) string {
	var notes []string

	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("20060102") // YYYYMMDD
		monthDir := dateStr[:6]            // YYYYMM
		filePath := filepath.Join(ms.memoryDir, monthDir, dateStr+".md")

		if data, err := os.ReadFile(filePath); err == nil {
			notes = append(notes, string(data))
		}
	}

	if len(notes) == 0 {
		return ""
	}

	// Join with separator
	var result string
	for i, note := range notes {
		if i > 0 {
			result += "\n\n---\n\n"
		}
		result += note
	}
	return result
}

// GetMemoryContext returns formatted memory context for the agent prompt.
// Includes long-term memory and recent daily notes.
func (ms *FilesystemMemoryStore) GetMemoryContext() string {
	var parts []string

	// Long-term memory
	longTerm := ms.ReadLongTerm()
	if longTerm != "" {
		parts = append(parts, "## Long-term Memory\n\n"+longTerm)
	}

	// Recent daily notes (last 3 days)
	recentNotes := ms.GetRecentDailyNotes(3)
	if recentNotes != "" {
		parts = append(parts, "## Recent Daily Notes\n\n"+recentNotes)
	}

	if len(parts) == 0 {
		return ""
	}

	// Join parts with separator
	var result string
	for i, part := range parts {
		if i > 0 {
			result += "\n\n---\n\n"
		}
		result += part
	}
	return fmt.Sprintf("# Memory\n\n%s", result)
}

// NewMemoryStoreFromConfig creates a MemoryStore based on config.
// Returns FilesystemMemoryStore when backend is "filesystem" or empty.
// Returns S3MemoryStore when backend is "s3".
// Returns R2MemoryStore when backend is "r2" (Cloudflare R2).
func NewMemoryStoreFromConfig(workspace string, memCfg *config.MemoryConfig) (tools.MemoryStore, error) {
	if memCfg == nil {
		return NewFilesystemMemoryStore(workspace), nil
	}
	backend := strings.ToLower(memCfg.Backend)

	if backend == "r2" {
		accountID := memCfg.R2.AccountID
		bucket := memCfg.R2.Bucket
		accessKey := memCfg.R2.AccessKeyID
		secretKey := memCfg.R2.SecretAccessKey
		if accountID == "" {
			accountID = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
		}
		if accessKey == "" {
			accessKey = os.Getenv("R2_ACCESS_KEY_ID")
		}
		if secretKey == "" {
			secretKey = os.Getenv("R2_SECRET_ACCESS_KEY")
		}
		if accountID == "" || bucket == "" || accessKey == "" || secretKey == "" {
			return nil, fmt.Errorf("memory backend is r2 but r2.account_id, r2.bucket, R2_ACCESS_KEY_ID, and R2_SECRET_ACCESS_KEY are required")
		}
		r2Cfg := R2MemoryStoreConfig{
			AccountID:        accountID,
			Bucket:           bucket,
			AccessKeyID:      accessKey,
			SecretAccessKey:  secretKey,
			Prefix:           memCfg.R2.Prefix,
		}
		return NewR2MemoryStore(r2Cfg)
	}

	if backend == "s3" {
		bucket := memCfg.S3.Bucket
		if bucket == "" && len(memCfg.S3.Buckets) > 0 {
			bucket = memCfg.S3.Buckets[0]
		}
		if bucket == "" {
			return nil, fmt.Errorf("memory backend is s3 but memory.s3.bucket (or buckets[0]) is not set")
		}
		s3Cfg := S3MemoryStoreConfig{
			Bucket: bucket,
			Prefix: memCfg.S3.Prefix,
			Region: memCfg.S3.Region,
		}
		return NewS3MemoryStore(s3Cfg)
	}

	return NewFilesystemMemoryStore(workspace), nil
}
