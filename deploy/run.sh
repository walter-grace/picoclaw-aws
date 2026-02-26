#!/usr/bin/env bash
# Run Claw Cubed in production. Invoke from the repo root (parent of deploy/).
# Usage: ./deploy/run.sh [agent|gateway]
#   agent   – CLI-only agent (default; minimal footprint)
#   gateway – agent + HTTP API (for channels / future UI)

set -e
cd "$(dirname "$0")/.."
ROOT="$PWD"

if ! command -v go &>/dev/null; then
  echo "error: go not found. Install Go 1.21+ and ensure it is on PATH."
  exit 1
fi

# Build once
if [[ ! -f "$ROOT/clawcubed" ]] || [[ "$ROOT/cmd/picoclaw/main.go" -nt "$ROOT/clawcubed" ]]; then
  echo "Building clawcubed..."
  go build -o "$ROOT/clawcubed" ./cmd/picoclaw
fi

MODE="${1:-agent}"
case "$MODE" in
  agent)   exec "$ROOT/clawcubed" agent ;;
  gateway) exec "$ROOT/clawcubed" gateway ;;
  *)
    echo "Usage: $0 [agent|gateway]"
    echo "  agent   – CLI-only (default)"
    echo "  gateway – agent + HTTP API"
    exit 1
    ;;
esac
