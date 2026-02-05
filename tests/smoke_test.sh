#!/bin/bash
# Smoke tests for http-https-echo
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
PORT="${TEST_PORT:-18080}"

echo "=== Smoke Tests ==="

# Build the binary
echo "Building..."
cd "$PROJECT_DIR"
go build -o http-echo .

# Start server in background
HTTP_PORT=$PORT ./http-echo &
PID=$!
trap "kill $PID 2>/dev/null || true; rm -f http-echo" EXIT

# Wait for server to start
sleep 1

# Test 1: Basic GET request
echo "Test 1: Basic GET request..."
RESPONSE=$(curl -s "http://localhost:$PORT/test/path")
echo "$RESPONSE" | jq -e '.path == "/test/path"' > /dev/null
echo "$RESPONSE" | jq -e '.method == "GET"' > /dev/null
echo "  PASS"

# Test 2: POST request with body
echo "Test 2: POST request with body..."
RESPONSE=$(curl -s -X POST -d "hello world" "http://localhost:$PORT/")
echo "$RESPONSE" | jq -e '.method == "POST"' > /dev/null
echo "$RESPONSE" | jq -e '.body == "hello world"' > /dev/null
echo "  PASS"

# Test 3: Query parameters
echo "Test 3: Query parameters..."
RESPONSE=$(curl -s "http://localhost:$PORT/?foo=bar&baz=qux")
echo "$RESPONSE" | jq -e '.query.foo[0] == "bar"' > /dev/null
echo "$RESPONSE" | jq -e '.query.baz[0] == "qux"' > /dev/null
echo "  PASS"

# Test 4: Custom headers
echo "Test 4: Custom headers..."
RESPONSE=$(curl -s -H "X-Custom: test-value" "http://localhost:$PORT/")
echo "$RESPONSE" | jq -e '.headers["X-Custom"][0] == "test-value"' > /dev/null
echo "  PASS"

# Test 5: Response contains OS hostname
echo "Test 5: OS hostname present..."
RESPONSE=$(curl -s "http://localhost:$PORT/")
echo "$RESPONSE" | jq -e '.os.hostname != null' > /dev/null
echo "  PASS"

echo ""
echo "=== All smoke tests passed ==="
