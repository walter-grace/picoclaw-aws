// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

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
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

const (
	s3MemoryKeyLongTerm = "memory/MEMORY.md"
	s3MemoryKeyFormat   = "memory/%s/%s.md" // memory/YYYYMM/YYYYMMDD.md
)

// S3MemoryStoreConfig configures the S3 memory backend.
type S3MemoryStoreConfig struct {
	Bucket string
	Prefix string // optional, e.g. "prod/" for multi-env
	Region string // optional, uses AWS_REGION env if empty
}

// S3MemoryStore manages persistent memory in an S3 bucket.
// - Long-term memory: memory/MEMORY.md (or prefix/memory/MEMORY.md)
// - Daily notes: memory/YYYYMM/YYYYMMDD.md
type S3MemoryStore struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewS3MemoryStore creates a new S3MemoryStore from config.
// Uses default AWS credential chain (env, shared config, etc.).
func NewS3MemoryStore(cfg S3MemoryStoreConfig) (*S3MemoryStore, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("S3 memory bucket is required")
	}

	ctx := context.Background()
	opts := []func(*config.LoadOptions) error{}
	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	prefix := strings.Trim(cfg.Prefix, "/")
	if prefix != "" {
		prefix = prefix + "/"
	}

	return &S3MemoryStore{
		client: client,
		bucket: cfg.Bucket,
		prefix: prefix,
	}, nil
}

func (ms *S3MemoryStore) s3Key(path string) string {
	return ms.prefix + path
}

// BucketName returns the bucket name (for context builder to show which bucket is memory).
func (ms *S3MemoryStore) BucketName() string {
	return ms.bucket
}

func (ms *S3MemoryStore) getObject(ctx context.Context, key string) (string, error) {
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

func (ms *S3MemoryStore) putObject(ctx context.Context, key string, content string) error {
	_, err := ms.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(ms.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader([]byte(content)),
		ContentType: aws.String("text/markdown"),
	})
	return err
}

// ReadLongTerm reads the long-term memory (MEMORY.md).
func (ms *S3MemoryStore) ReadLongTerm() string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	content, err := ms.getObject(ctx, ms.s3Key(s3MemoryKeyLongTerm))
	if err != nil {
		return ""
	}
	return content
}

// WriteLongTerm writes content to the long-term memory file (MEMORY.md).
func (ms *S3MemoryStore) WriteLongTerm(content string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return ms.putObject(ctx, ms.s3Key(s3MemoryKeyLongTerm), content)
}

// ReadToday reads today's daily note.
func (ms *S3MemoryStore) ReadToday() string {
	today := time.Now().Format("20060102")
	return ms.ReadDailyNote(today)
}

// AppendToday appends content to today's daily note.
func (ms *S3MemoryStore) AppendToday(content string) error {
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
func (ms *S3MemoryStore) ReadDailyNote(date string) string {
	if len(date) != 8 {
		return ""
	}
	monthDir := date[:6]
	key := ms.s3Key(fmt.Sprintf(s3MemoryKeyFormat, monthDir, date))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	content, err := ms.getObject(ctx, key)
	if err != nil {
		return ""
	}
	return content
}

// WriteDailyNote overwrites a daily note for the given date (YYYYMMDD).
func (ms *S3MemoryStore) WriteDailyNote(date, content string) error {
	if len(date) != 8 {
		return fmt.Errorf("invalid date format: %s (expected YYYYMMDD)", date)
	}
	monthDir := date[:6]
	key := ms.s3Key(fmt.Sprintf(s3MemoryKeyFormat, monthDir, date))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return ms.putObject(ctx, key, content)
}

// GetRecentDailyNotes returns daily notes from the last N days.
func (ms *S3MemoryStore) GetRecentDailyNotes(days int) string {
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

// MemoryObject represents a file in the S3 memory bucket.
type MemoryObject struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified"`
}

// ListMemoryObjects lists memory files (MEMORY.md and daily notes) in the bucket.
func (ms *S3MemoryStore) ListMemoryObjects(ctx context.Context) ([]MemoryObject, error) {
	prefix := ms.s3Key("memory/")
	paginator := s3.NewListObjectsV2Paginator(ms.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(ms.bucket),
		Prefix: aws.String(prefix),
	}, func(o *s3.ListObjectsV2PaginatorOptions) {
		o.Limit = 100
	})

	var objects []MemoryObject
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}
			key := strings.TrimPrefix(*obj.Key, ms.prefix)
			lastMod := ""
			if obj.LastModified != nil {
				lastMod = obj.LastModified.Format(time.RFC3339)
			}
			size := int64(0)
			if obj.Size != nil {
				size = *obj.Size
			}
			objects = append(objects, MemoryObject{
				Key:          key,
				Size:         size,
				LastModified: lastMod,
			})
		}
	}
	return objects, nil
}

// ListTopLevelPrefixes returns top-level "folder" names (common prefixes) in the bucket.
// Useful for answering "how many projects/folders are in the bucket".
func (ms *S3MemoryStore) ListTopLevelPrefixes(ctx context.Context) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(ms.bucket),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int32(1000),
	}
	if ms.prefix != "" {
		input.Prefix = aws.String(ms.prefix)
	}
	out, err := ms.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, p := range out.CommonPrefixes {
		if p.Prefix != nil {
			name := strings.TrimPrefix(*p.Prefix, ms.prefix)
			name = strings.Trim(name, "/")
			if name != "" {
				names = append(names, name)
			}
		}
	}
	return names, nil
}

// GetMemoryContext returns formatted memory context for the agent prompt.
func (ms *S3MemoryStore) GetMemoryContext() string {
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
