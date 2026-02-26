#!/bin/bash
# Test S3 memory connectivity for Claw Cubed.
# Run from claw-cubed directory: ./scripts/test_s3_memory.sh

set -e
cd "$(dirname "$0")/.."
go run ./scripts/test_s3_memory.go
