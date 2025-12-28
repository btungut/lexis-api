# Lexis API

Lightweight HTTP microservice for language detection. Built with Go, Fiber, and `lingua-go`.

**Features**: Detects 15+ languages • Kubernetes-ready • Docker image available • Health monitoring • CORS enabled

## Quick Start

```bash
# Run with Docker
docker run -p 3000:3000 -e PORT=3000 ghcr.io/btungut/lexis-api:0.0.1

# Test it
curl -X POST http://localhost:3000/detect \
  -H "Content-Type: application/json" \
  -d '{"text":"Hello, this is a test."}'
```

## Kubernetes Deployment

### Install with Helm

```bash
helm dependency update ./helm
helm upgrade --install lexis-api ./helm \
  --namespace lexis-api \
  --create-namespace
```

If you encounter authentication errors:

```bash
helm registry login ghcr.io
```

### Configuration

Key values in [helm/values.yaml](helm/values.yaml):

- `deployment.image.repository` / `deployment.image.tag` - Container image
- `deployment.env.PORT` - Application port (e.g., `4001`)
- `deployment.env.MAX_CHAR_PROCESS` - Max text length (default: `2000`)
- `service.type` / `service.port` - Service configuration
- `ingress.enabled` / `ingress.rule.*` - Ingress settings

Custom values example:

```yaml
# values.local.yaml
shared:
  App_Port: 4001  # Centralized port configuration

deployment:
  replicaCount: 2
  image:
    tag: latest
  env:
    PORT: "{{ .Values.shared.App_Port }}"  # References shared.App_Port
    MAX_CHAR_PROCESS: "2000"
```

Apply:

```bash
helm upgrade --install lexis-api ./helm -n lexis-api --create-namespace -f values.local.yaml
```

### Advanced Ingress Configuration

For path-based routing with URL rewriting:

```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    nginx.ingress.kubernetes.io/use-regex: "true"
    nginx.ingress.kubernetes.io/rewrite-target: /$2
  rule:
    host: nonprod.example.local
    path: /lexis(/|$)(.*)
```

This routes `nonprod.example.local/lexis/*` to the service root path.

## Development

### Local Development

```bash
# Build
go build -o lexis-api .

# Run (PORT will be normalized automatically)
export PORT=3000
export MAX_CHAR_PROCESS=2000
./lexis-api

# Run tests
go test ./...

# Run specific test
go test -run TestDetectEndpoint ./...
```

### Build Docker Image

```bash
docker build -t lexis-api:local .
docker run --rm -p 3000:3000 -e PORT=3000 -e MAX_CHAR_PROCESS=2000 lexis-api:local
```

## API

### `GET /health`

Health check endpoint for Kubernetes probes.

```bash
curl http://localhost:3000/health
# Returns: 200 OK
```

### `POST /detect`

Detects language and returns ISO 639-1 code with confidence score.

**Request**:

```json
{"text": "Hello, this is a test."}
```

**Response**:

```json
{"language": "en", "confidence": 0.97}
```

**Examples**:

```bash
# English
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{"text":"Hello, this is a test."}'

# Turkish
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{"text":"Merhaba, bu bir testtir."}'

# Spanish
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{"text":"Hola, esto es una prueba."}'
```

**Response Fields**:

- `language` - ISO 639-1 code (`en`, `tr`, `es`, etc.) or `unknown`
- `confidence` - Float between 0.0 and 1.0

**Error (400)**:

```json
{"error": "Invalid JSON format"}
{"error": "Description cannot be empty"}
```

**Note**: Text exceeding `MAX_CHAR_PROCESS` is automatically truncated (default: 2000 chars).

## Supported Languages

15 preloaded language models:

| Language    | Code | Language    | Code | Language   | Code |
| ----------- | ---- | ----------- | ---- | ---------- | ---- |
| English     | `en` | Spanish     | `es` | Arabic     | `ar` |
| Turkish     | `tr` | Italian     | `it` | Chinese    | `zh` |
| German      | `de` | Portuguese  | `pt` | Japanese   | `ja` |
| French      | `fr` | Russian     | `ru` | Korean     | `ko` |
| Dutch       | `nl` | Azerbaijani | `az` | Persian    | `fa` |

For broader coverage, modify [main.go](main.go:57-67) to use `.FromAllLanguages()`.

## Environment Variables

| Variable           | Default | Description                  |
| ------------------ | ------- | ---------------------------- |
| `PORT`             | `3000`  | HTTP server port             |
| `MAX_CHAR_PROCESS` | `1000`  | Max characters per request   |

## Docker Image

**Image**: `ghcr.io/btungut/lexis-api:0.0.1`

**Features**:

- Multi-stage build (golang:1.25-alpine → alpine:3.19)
- Static binary with stripped debug symbols
- Non-root user execution
- Minimal footprint

## Contributing

Contributions welcome!

**Guidelines**:

- Add/update tests in [main_test.go](main_test.go) for behavior changes
- Follow Go best practices (`gofmt`)
- Keep changes focused and documented
- Run `go test ./...` before submitting

**Security**: Report vulnerabilities privately to maintainers, not via public issues.

## License

This project is released under the **Lexis Non-Commercial Source License (NCSL) v1.0**.

### You are allowed to:
- Pull and run the official Docker image
- Use the software for personal, educational, or internal evaluation purposes
- Clone, modify, and build your own Docker images
- Self-host the software for non-commercial use only

### You are NOT allowed to:
- Use this software in any commercial or revenue-generating product or service
- Offer the software as part of a paid platform or subscription
- Provide the software as a hosted or managed service (SaaS)
- Monetize the software directly or indirectly
- Sell access to the software or its functionality

Commercial use requires a separate commercial license.

📩 For commercial licensing inquiries, contact: **burak.tungut@tungops.com.tr**
