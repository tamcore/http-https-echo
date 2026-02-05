package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEchoHandler_BasicRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test/path?foo=bar", nil)
	req.Header.Set("X-Custom-Header", "test-value")

	rr := httptest.NewRecorder()
	echoHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var response EchoResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Path != "/test/path" {
		t.Errorf("expected path /test/path, got %s", response.Path)
	}

	if response.Method != http.MethodGet {
		t.Errorf("expected method GET, got %s", response.Method)
	}

	if len(response.Query["foo"]) == 0 || response.Query["foo"][0] != "bar" {
		t.Errorf("expected query param foo=bar, got %v", response.Query)
	}

	if len(response.Headers["X-Custom-Header"]) == 0 || response.Headers["X-Custom-Header"][0] != "test-value" {
		t.Errorf("expected X-Custom-Header=test-value, got %v", response.Headers)
	}
}

func TestEchoHandler_WithBody(t *testing.T) {
	body := "test request body"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	rr := httptest.NewRecorder()
	echoHandler(rr, req)

	var response EchoResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Body != body {
		t.Errorf("expected body %q, got %q", body, response.Body)
	}

	if response.Method != http.MethodPost {
		t.Errorf("expected method POST, got %s", response.Method)
	}
}

func TestEchoHandler_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")

	rr := httptest.NewRecorder()
	echoHandler(rr, req)

	var response EchoResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.IP != "192.168.1.1" {
		t.Errorf("expected IP 192.168.1.1, got %s", response.IP)
	}
}

func TestEchoHandler_XForwardedProto(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")

	rr := httptest.NewRecorder()
	echoHandler(rr, req)

	var response EchoResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Protocol != "https" {
		t.Errorf("expected protocol https, got %s", response.Protocol)
	}
}

func TestDecodeJWT_ValidToken(t *testing.T) {
	// Sample JWT: {"alg":"HS256","typ":"JWT"}.{"sub":"1234567890","name":"John Doe","iat":1516239022}
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	result := decodeJWT(token)

	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}

	header, ok := result.Header.(map[string]any)
	if !ok {
		t.Fatalf("expected header to be map, got %T", result.Header)
	}

	if header["alg"] != "HS256" {
		t.Errorf("expected alg=HS256, got %v", header["alg"])
	}

	if header["typ"] != "JWT" {
		t.Errorf("expected typ=JWT, got %v", header["typ"])
	}

	payload, ok := result.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected payload to be map, got %T", result.Payload)
	}

	if payload["sub"] != "1234567890" {
		t.Errorf("expected sub=1234567890, got %v", payload["sub"])
	}

	if payload["name"] != "John Doe" {
		t.Errorf("expected name=John Doe, got %v", payload["name"])
	}
}

func TestDecodeJWT_BearerPrefix(t *testing.T) {
	token := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	result := decodeJWT(token)

	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}

	payload, ok := result.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected payload to be map, got %T", result.Payload)
	}

	if payload["name"] != "John Doe" {
		t.Errorf("expected name=John Doe, got %v", payload["name"])
	}
}

func TestDecodeJWT_InvalidFormat(t *testing.T) {
	result := decodeJWT("not-a-jwt")

	if result.Error == "" {
		t.Error("expected error for invalid JWT format")
	}

	if !strings.Contains(result.Error, "invalid JWT format") {
		t.Errorf("expected 'invalid JWT format' error, got: %s", result.Error)
	}
}

func TestDecodeJWT_InvalidBase64(t *testing.T) {
	result := decodeJWT("!!!.!!!.!!!")

	if result.Error == "" {
		t.Error("expected error for invalid base64")
	}
}

func TestEchoHandler_WithJWT(t *testing.T) {
	// Save and restore original jwtHeader
	originalJwtHeader := jwtHeader
	defer func() { jwtHeader = originalJwtHeader }()

	jwtHeader = "Authorization"

	token := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", token)

	rr := httptest.NewRecorder()
	echoHandler(rr, req)

	body, _ := io.ReadAll(rr.Body)
	var response EchoResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.JWT == nil {
		t.Fatal("expected JWT in response")
	}

	if response.JWT.Error != "" {
		t.Errorf("unexpected JWT error: %s", response.JWT.Error)
	}

	payload, ok := response.JWT.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected JWT payload to be map, got %T", response.JWT.Payload)
	}

	if payload["name"] != "John Doe" {
		t.Errorf("expected name=John Doe in JWT payload, got %v", payload["name"])
	}
}
