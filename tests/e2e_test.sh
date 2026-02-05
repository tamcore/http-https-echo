#!/bin/bash
# E2E tests for http-https-echo with JWT decoding
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
PORT="${TEST_PORT:-18081}"

echo "=== E2E Tests ==="

# Build the binary
echo "Building..."
cd "$PROJECT_DIR"
go build -o http-echo .

# Sample JWT token (not validated, just decoded)
# Header: {"alg":"HS256","typ":"JWT"}
# Payload: {"sub":"1234567890","name":"John Doe","iat":1516239022}
JWT_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

cleanup() {
    kill $PID 2>/dev/null || true
    rm -f http-echo
}
trap cleanup EXIT

# =====================
# Test Suite 1: No JWT_HEADER configured
# =====================
echo ""
echo "--- Test Suite 1: No JWT_HEADER configured ---"

HTTP_PORT=$PORT ./http-echo &
PID=$!
sleep 1

echo "Test 1.1: JWT not decoded when JWT_HEADER not set..."
RESPONSE=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" "http://localhost:$PORT/")
# jwt should not be present in response
if echo "$RESPONSE" | jq -e '.jwt' > /dev/null 2>&1; then
    echo "  FAIL: jwt should not be present"
    exit 1
fi
echo "  PASS"

kill $PID 2>/dev/null || true
sleep 1

# =====================
# Test Suite 2: JWT_HEADER configured
# =====================
echo ""
echo "--- Test Suite 2: JWT_HEADER configured ---"

JWT_HEADER=Authorization HTTP_PORT=$PORT ./http-echo &
PID=$!
sleep 1

echo "Test 2.1: JWT decoded with Bearer prefix..."
RESPONSE=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" "http://localhost:$PORT/")
echo "$RESPONSE" | jq -e '.jwt.header.alg == "HS256"' > /dev/null
echo "$RESPONSE" | jq -e '.jwt.header.typ == "JWT"' > /dev/null
echo "$RESPONSE" | jq -e '.jwt.payload.sub == "1234567890"' > /dev/null
echo "$RESPONSE" | jq -e '.jwt.payload.name == "John Doe"' > /dev/null
echo "  PASS"

echo "Test 2.2: JWT decoded without Bearer prefix..."
RESPONSE=$(curl -s -H "Authorization: $JWT_TOKEN" "http://localhost:$PORT/")
echo "$RESPONSE" | jq -e '.jwt.payload.name == "John Doe"' > /dev/null
echo "  PASS"

echo "Test 2.3: Invalid JWT returns error..."
RESPONSE=$(curl -s -H "Authorization: not-a-jwt" "http://localhost:$PORT/")
echo "$RESPONSE" | jq -e '.jwt.error != null' > /dev/null
echo "  PASS"

echo "Test 2.4: Missing JWT header returns no jwt field..."
RESPONSE=$(curl -s "http://localhost:$PORT/")
if echo "$RESPONSE" | jq -e '.jwt' > /dev/null 2>&1; then
    echo "  FAIL: jwt should not be present when header missing"
    exit 1
fi
echo "  PASS"

kill $PID 2>/dev/null || true
sleep 1

# =====================
# Test Suite 3: Custom JWT header name (Teleport-Jwt-Assertion)
# =====================
echo ""
echo "--- Test Suite 3: Custom JWT header (Teleport-Jwt-Assertion) ---"

JWT_HEADER=Teleport-Jwt-Assertion HTTP_PORT=$PORT ./http-echo &
PID=$!
sleep 1

echo "Test 3.1: JWT decoded from custom header..."
RESPONSE=$(curl -s -H "Teleport-Jwt-Assertion: $JWT_TOKEN" "http://localhost:$PORT/")
echo "$RESPONSE" | jq -e '.jwt.payload.name == "John Doe"' > /dev/null
echo "  PASS"

echo "Test 3.2: Authorization header ignored when custom header configured..."
RESPONSE=$(curl -s -H "Authorization: Bearer $JWT_TOKEN" "http://localhost:$PORT/")
if echo "$RESPONSE" | jq -e '.jwt' > /dev/null 2>&1; then
    echo "  FAIL: jwt should not be decoded from Authorization header"
    exit 1
fi
echo "  PASS"

kill $PID 2>/dev/null || true
sleep 1

# =====================
# Test Suite 4: Various HTTP methods
# =====================
echo ""
echo "--- Test Suite 4: HTTP Methods ---"

HTTP_PORT=$PORT ./http-echo &
PID=$!
sleep 1

for METHOD in GET POST PUT DELETE PATCH OPTIONS HEAD; do
    if [ "$METHOD" = "HEAD" ]; then
        # HEAD returns no body
        echo "Test 4.x: $METHOD request (skip body check)..."
        curl -s -X $METHOD "http://localhost:$PORT/" > /dev/null
    else
        echo "Test 4.x: $METHOD request..."
        RESPONSE=$(curl -s -X $METHOD "http://localhost:$PORT/")
        echo "$RESPONSE" | jq -e ".method == \"$METHOD\"" > /dev/null
    fi
    echo "  PASS"
done

echo ""
echo "=== All E2E tests passed ==="
