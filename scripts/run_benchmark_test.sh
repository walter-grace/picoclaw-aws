#!/usr/bin/env bash
# Run Code Mode benchmark: start gateway twice (code_mode off, then on), compare tokens and latency.
# Uses port 18790. Requires: ~/.picoclaw/config.json (or config.example.json), .env with AWS + LLM keys, jq, go.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT"

GATEWAY_URL="${CLAW_CUBED_URL:-http://localhost:18790}"
CONFIG_SOURCE="${HOME}/.picoclaw/config.json"
if [[ ! -f "$CONFIG_SOURCE" ]]; then
  CONFIG_SOURCE="$ROOT/config/config.example.json"
fi
if [[ ! -f "$CONFIG_SOURCE" ]]; then
  echo "No config found at $CONFIG_SOURCE or ~/.picoclaw/config.json"
  exit 1
fi
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

CONFIG_OFF="$TMP_DIR/config_code_off.json"
CONFIG_ON="$TMP_DIR/config_code_on.json"

jq '.tools.aws_mcp.enabled = true | .tools.code_mode.enabled = false' "$CONFIG_SOURCE" > "$CONFIG_OFF"
jq '.tools.aws_mcp.enabled = true | .tools.code_mode.enabled = true'  "$CONFIG_SOURCE" > "$CONFIG_ON"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  source "$ROOT/.env"
  set +a
fi

wait_health() {
  local url="$1"
  local max=30
  while [[ $max -gt 0 ]]; do
    if curl -s -o /dev/null -w "%{http_code}" "$url/health" 2>/dev/null | grep -q 200; then
      return 0
    fi
    sleep 1
    ((max--)) || true
  done
  return 1
}

run_benchmark() {
  local label="$1"
  go run ./scripts/benchmark_code_mode.go --url "$GATEWAY_URL" --label "$label" --json
}

echo "=== Code Mode benchmark (port 18790) ==="
echo "Config source: $CONFIG_SOURCE"
echo ""

# Kill any existing process on port 18790 so we own it cleanly
echo "Clearing port 18790..."
lsof -ti:18790 | xargs kill -9 2>/dev/null || true
sleep 2

echo "--- Run 1: code_mode OFF ---"
PICOCLAW_CONFIG_PATH="$CONFIG_OFF" go run ./cmd/picoclaw gateway &
PID_OFF=$!
if ! wait_health "$GATEWAY_URL"; then
  kill $PID_OFF 2>/dev/null || true
  echo "Gateway (code_mode off) did not become ready."
  exit 1
fi
RESULT_OFF=$(run_benchmark "tool_calls")
kill $PID_OFF 2>/dev/null || true
wait $PID_OFF 2>/dev/null || true
# go run spawns a child binary; kill the actual port listener too
lsof -ti:18790 | xargs kill -9 2>/dev/null || true
for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
  ! curl -s -o /dev/null "$GATEWAY_URL/health" 2>/dev/null && break
  sleep 1
done
sleep 2

echo "--- Run 2: code_mode ON ---"
PICOCLAW_CONFIG_PATH="$CONFIG_ON" go run ./cmd/picoclaw gateway &
PID_ON=$!
if ! wait_health "$GATEWAY_URL"; then
  kill $PID_ON 2>/dev/null || true
  echo "Gateway (code_mode on) did not become ready."
  exit 1
fi
RESULT_ON=$(run_benchmark "code_mode")
kill $PID_ON 2>/dev/null || true
wait $PID_ON 2>/dev/null || true
lsof -ti:18790 | xargs kill -9 2>/dev/null || true

echo ""
echo "=== Comparison ==="
echo "code_mode OFF (tool_calls):"
echo "$RESULT_OFF" | jq .
echo ""
echo "code_mode ON:"
echo "$RESULT_ON" | jq .
echo ""
echo "Summary:"
echo "$RESULT_OFF" | jq -r '"\(.label): duration_ms=\(.duration_ms) total_tokens=\(.total_tokens) iterations=\(.iterations)"'
echo "$RESULT_ON"  | jq -r '"\(.label): duration_ms=\(.duration_ms) total_tokens=\(.total_tokens) iterations=\(.iterations)"'
