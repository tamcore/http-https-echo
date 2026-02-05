package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// EchoResponse represents the JSON response structure
type EchoResponse struct {
	Path     string              `json:"path"`
	Method   string              `json:"method"`
	Headers  map[string][]string `json:"headers"`
	Body     string              `json:"body"`
	Query    map[string][]string `json:"query"`
	Hostname string              `json:"hostname"`
	IP       string              `json:"ip"`
	Protocol string              `json:"protocol"`
	OS       OSInfo              `json:"os"`
	JWT      *JWTInfo            `json:"jwt,omitempty"`
}

// OSInfo contains OS-level information
type OSInfo struct {
	Hostname string `json:"hostname"`
}

// JWTInfo contains decoded JWT information
type JWTInfo struct {
	Header  any    `json:"header,omitempty"`
	Payload any    `json:"payload,omitempty"`
	Raw     string `json:"raw,omitempty"`
	Error   string `json:"error,omitempty"`
}

var (
	jwtHeader string
	logJWT    bool
)

func init() {
	jwtHeader = os.Getenv("JWT_HEADER")
	logJWT = strings.ToLower(os.Getenv("LOG_JWT")) == "true"
}

func main() {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", loggingMiddleware(echoHandler))

	log.Printf("Starting HTTP echo server on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next(wrapped, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		body = []byte{}
	}
	defer func() { _ = r.Body.Close() }()

	// Get OS hostname
	osHostname, _ := os.Hostname()

	// Determine client IP
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = strings.Split(forwarded, ",")[0]
	}

	// Determine protocol
	protocol := "http"
	if r.TLS != nil {
		protocol = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		protocol = proto
	}

	response := EchoResponse{
		Path:     r.URL.Path,
		Method:   r.Method,
		Headers:  r.Header,
		Body:     string(body),
		Query:    r.URL.Query(),
		Hostname: r.Host,
		IP:       ip,
		Protocol: protocol,
		OS: OSInfo{
			Hostname: osHostname,
		},
	}

	// Decode JWT if header is configured
	if jwtHeader != "" {
		token := r.Header.Get(jwtHeader)
		if token != "" {
			jwtInfo := decodeJWT(token)
			response.JWT = jwtInfo

			if logJWT && jwtInfo != nil {
				jwtJSON, _ := json.Marshal(jwtInfo)
				log.Printf("Decoded JWT: %s", string(jwtJSON))
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// decodeJWT decodes a JWT token without verifying the signature
func decodeJWT(token string) *JWTInfo {
	// Strip "Bearer " prefix if present
	token = strings.TrimSpace(token)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return &JWTInfo{
			Raw:   token,
			Error: "invalid JWT format: expected 3 parts",
		}
	}

	header, err := decodeBase64JSON(parts[0])
	if err != nil {
		return &JWTInfo{
			Raw:   token,
			Error: "failed to decode header: " + err.Error(),
		}
	}

	payload, err := decodeBase64JSON(parts[1])
	if err != nil {
		return &JWTInfo{
			Raw:   token,
			Error: "failed to decode payload: " + err.Error(),
		}
	}

	return &JWTInfo{
		Header:  header,
		Payload: payload,
	}
}

// decodeBase64JSON decodes a base64url-encoded JSON string
func decodeBase64JSON(s string) (any, error) {
	// JWT uses base64url encoding (RFC 4648)
	// Add padding if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		// Try standard base64 as fallback
		decoded, err = base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, err
		}
	}

	var result any
	if err := json.Unmarshal(decoded, &result); err != nil {
		return nil, err
	}

	return result, nil
}
