# HTTP Echo Server

A minimal HTTP echo server written in Go that echoes HTTP request properties back to the client. Inspired by [mendhak/docker-http-https-echo](https://github.com/mendhak/docker-http-https-echo).

**Key Feature:** JWT header decoding - configure via `JWT_HEADER` env var to decode and display JWT claims (e.g., for Teleport JWT assertions).

## Features

- Echo all HTTP request properties (path, method, headers, body, query params)
- JWT token decoding (without signature verification)
- Request logging to stdout
- Minimal dependencies (Go stdlib only)
- Multi-arch container images via ko

## Quick Start

### Using Docker

```bash
docker run -p 8080:8080 ghcr.io/tamcore/http-https-echo:latest
```

### Using Go

```bash
go install github.com/tamcore/http-https-echo@latest
http-echo
```

### Build from Source

```bash
go build -o http-echo .
./http-echo
```

## Usage

```bash
# Basic request
curl http://localhost:8080/hello-world

# POST with body
curl -X POST -d '{"key":"value"}' http://localhost:8080/

# With custom headers
curl -H "X-Custom-Header: test" http://localhost:8080/
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_PORT` | Port to listen on | `8080` |
| `JWT_HEADER` | Header name containing JWT to decode | (none) |
| `LOG_JWT` | Log decoded JWT to stdout (`true`/`false`) | `false` |

## JWT Decoding

To decode JWT tokens from a specific header:

```bash
# Start with JWT decoding enabled
JWT_HEADER=Authorization ./http-echo

# Or with Teleport JWT assertions
JWT_HEADER=Teleport-Jwt-Assertion ./http-echo
```

Then make a request with a JWT:

```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c" \
  http://localhost:8080/
```

Response will include decoded JWT:

```json
{
  "path": "/",
  "method": "GET",
  "headers": { ... },
  "jwt": {
    "header": {
      "alg": "HS256",
      "typ": "JWT"
    },
    "payload": {
      "sub": "1234567890",
      "name": "John Doe",
      "iat": 1516239022
    }
  },
  ...
}
```

To also log decoded JWTs to stdout:

```bash
JWT_HEADER=Authorization LOG_JWT=true ./http-echo
```

## Response Format

```json
{
  "path": "/hello",
  "method": "GET",
  "headers": {
    "Accept": ["*/*"],
    "User-Agent": ["curl/8.0"]
  },
  "body": "",
  "query": {
    "foo": ["bar"]
  },
  "hostname": "localhost:8080",
  "ip": "127.0.0.1:54321",
  "protocol": "http",
  "os": {
    "hostname": "my-machine"
  },
  "jwt": { ... }
}
```

## Development

### Prerequisites

- Go 1.21+
- golangci-lint
- goreleaser

### Run Tests

```bash
# Unit tests
go test -v ./...

# Smoke tests
./tests/smoke_test.sh

# E2E tests
./tests/e2e_test.sh
```

### Build

```bash
# Local build
go build -o http-echo .

# Release build (dry run)
goreleaser release --snapshot --clean
```

## License

MIT
