#!/bin/bash
# Run pico-aws with .env loaded
cd "$(dirname "$0")"
set -a
source .env 2>/dev/null || true
set +a
export PATH="$HOME/.local/bin:$PATH"
exec ./pico-aws "$@"
