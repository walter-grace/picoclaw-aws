#!/bin/bash
# Test S3 memory connectivity for pico-aws.
# Run from picoclaw-aws directory: ./scripts/test_s3_memory.sh

set -e
cd "$(dirname "$0")/.."
go run ./scripts/test_s3_memory.go
