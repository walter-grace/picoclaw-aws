// Package agent - Cloudflare R2 memory backend.
// R2 is S3-compatible; uses custom endpoint https://<account_id>.r2.cloudflarestorage.com
package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

const (
	r2MemoryKeyLongTerm = "memory/MEMORY.md"
	r2MemoryKeyFormat   = "memory/%s/%s.md" // memory/YYYYMM/YYYYMMDD.md
)

// R2MemoryStoreConfig configures the Cloudflare R2 memory backend.
type R2MemoryStoreConfig struct {
	AccountID      string // Cloudflare account ID
	Bucket         string // R2 bucket name (e.g. pico-flare)
	AccessKeyID    string // R2 API token access key
	SecretAccessKey string // R2 API token secret
	Prefix         string // optional, e.g. "prod/" for multi-env
}

// R2MemoryStore manages persistent memory in Cloudflare R2.
// Implements tools.MemoryStore with the same layout as S3MemoryStore.
type R2MemoryStore struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewR2MemoryStore creates a new R2MemoryStore from config.
func NewR2MemoryStore(cfg R2MemoryStoreConfig) (*R2MemoryStore, error) {
	if cfg.AccountID == "" || cfg.Bucket == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("R2 memory requires account_id, bucket, access_key_id, and secret_access_key")
	}
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	awsCfg := aws.Config{
		Region: "auto",
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		),
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	prefix := strings.Trim(cfg.Prefix, "/")
	if prefix != "" {
		prefix = prefix + "/"
	}

	return &R2MemoryStore{
		client: client,
		bucket: cfg.Bucket,
		prefix: prefix,
	}, nil
}

func (ms *R2MemoryStore) s3Key(path string) string {
	return ms.prefix + path
}

// BucketName returns the bucket name (for context builder).
func (ms *R2MemoryStore) BucketName() string {
	return ms.bucket
}

func (ms *R2MemoryStore) getObject(ctx context.Context, key string) (string, error) {
	out, err := ms.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(ms.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) && ae.ErrorCode() == "NoSuchKey" {
			return "", nil
		}
		return "", err
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (ms *R2MemoryStore) putObject(ctx context.Context, key string, content string) error {
	_, err := ms.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(ms.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader([]byte(content)),
		ContentType: aws.String("text/markdown"),
	})
	return err
}

// ReadLongTerm reads the long-term memory (MEMORY.md).
func (ms *R2MemoryStore) ReadLongTerm() string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	content, err := ms.getObject(ctx, ms.s3Key(r2MemoryKeyLongTerm))
	if err != nil {
		return ""
	}
	return content
}

// WriteLongTerm writes content to the long-term memory file (MEMORY.md).
func (ms *R2MemoryStore) WriteLongTerm(content string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return ms.putObject(ctx, ms.s3Key(r2MemoryKeyLongTerm), content)
}

// ReadToday reads today's daily note.
func (ms *R2MemoryStore) ReadToday() string {
	today := time.Now().Format("20060102")
	return ms.ReadDailyNote(today)
}

// AppendToday appends content to today's daily note.
func (ms *R2MemoryStore) AppendToday(content string) error {
	today := time.Now().Format("20060102")
	existing := ms.ReadDailyNote(today)
	var newContent string
	if existing == "" {
		header := fmt.Sprintf("# %s\n\n", time.Now().Format("2006-01-02"))
		newContent = header + content
	} else {
		newContent = existing + "\n" + content
	}
	return ms.WriteDailyNote(today, newContent)
}

// ReadDailyNote reads a daily note for the given date (YYYYMMDD).
func (ms *R2MemoryStore) ReadDailyNote(date string) string {
	if len(date) != 8 {
		return ""
	}
	monthDir := date[:6]
	key := ms.s3Key(fmt.Sprintf(r2MemoryKeyFormat, monthDir, date))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	content, err := ms.getObject(ctx, key)
	if err != nil {
		return ""
	}
	return content
}

// WriteDailyNote overwrites a daily note for the given date (YYYYMMDD).
func (ms *R2MemoryStore) WriteDailyNote(date, content string) error {
	if len(date) != 8 {
		return fmt.Errorf("invalid date format: %s (expected YYYYMMDD)", date)
	}
	monthDir := date[:6]
	key := ms.s3Key(fmt.Sprintf(r2MemoryKeyFormat, monthDir, date))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return ms.putObject(ctx, key, content)
}

// GetRecentDailyNotes returns daily notes from the last N days.
func (ms *R2MemoryStore) GetRecentDailyNotes(days int) string {
	var notes []string
	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("20060102")
		note := ms.ReadDailyNote(dateStr)
		if note != "" {
			notes = append(notes, note)
		}
	}
	if len(notes) == 0 {
		return ""
	}
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
func (ms *R2MemoryStore) GetMemoryContext() string {
	var parts []string
	longTerm := ms.ReadLongTerm()
	if longTerm != "" {
		parts = append(parts, "## Long-term Memory\n\n"+longTerm)
	}
	recentNotes := ms.GetRecentDailyNotes(3)
	if recentNotes != "" {
		parts = append(parts, "## Recent Daily Notes\n\n"+recentNotes)
	}
	if len(parts) == 0 {
		return ""
	}
	var result string
	for i, part := range parts {
		if i > 0 {
			result += "\n\n---\n\n"
		}
		result += part
	}
	return fmt.Sprintf("# Memory\n\n%s", result)
}
