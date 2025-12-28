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

**Shared Values** (centralized configuration):

- `shared.App_Port` - Application port (default: `4001`)
- `shared.App_MaxCharProcess` - Max characters to process (default: `2000`)

**Deployment**:

- `deployment.image.repository` / `deployment.image.tag` - Container image
- `deployment.replicaCount` - Number of pod replicas
- `deployment.env.PORT` - References `{{ .Values.shared.App_Port }}`
- `deployment.env.MAX_CHAR_PROCESS` - References `{{ .Values.shared.App_MaxCharProcess }}`

**Service & Ingress**:

- `service.type` / `service.port` - Service configuration
- `ingress.enabled` / `ingress.rule.*` - Ingress settings

Custom values example:

```yaml
# values.local.yaml
shared:
  App_Port: 4001           # Centralized port configuration
  App_MaxCharProcess: 3000 # Centralized max char process

deployment:
  replicaCount: 2
  image:
    tag: latest
```

The `deployment.env.PORT` and `deployment.env.MAX_CHAR_PROCESS` automatically reference the shared values via Helm templating.

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

Detects language and returns ISO 639-1 code, language name, and confidence score.

**Request**:

```json
{"text": "Hello, this is a test."}
```

**Response (200)**:

```json
{"iso_code": "en", "language": "english", "confidence": 0.97}
```

**Examples**:

```bash
# English
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{"text":"Hello, this is a test."}'
# Response: {"iso_code":"en","language":"english","confidence":0.97}

# Turkish
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{"text":"Merhaba, bu bir testtir."}'
# Response: {"iso_code":"tr","language":"turkish","confidence":0.98}

# Spanish
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{"text":"Hola, esto es una prueba."}'
# Response: {"iso_code":"es","language":"spanish","confidence":0.95}
```

**Response Fields**:

- `iso_code` - ISO 639-1 code (`en`, `tr`, `es`, etc.)
- `language` - Full language name (`english`, `turkish`, `spanish`, etc.)
- `confidence` - Float between 0.0 and 1.0

**Note**: Text exceeding `MAX_CHAR_PROCESS` is automatically truncated (default: 2000 chars).

### Error Responses

All error responses return HTTP 400 Bad Request with a JSON body containing `error` (human-readable message) and `code` (machine-readable identifier) fields.

#### `INVALID_JSON`

Returned when the request body is not valid JSON or cannot be parsed.

```json
{"error": "Invalid JSON format", "code": "INVALID_JSON"}
```

**Common causes**:

- Malformed JSON syntax (missing brackets, quotes, commas)
- Empty request body
- Incorrect Content-Type header
- Binary or non-UTF-8 encoded data

**Example of invalid requests**:

```bash
# Missing closing brace
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{"text": "Hello"'

# Missing quotes around value
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{text: Hello}'

# Empty body
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json"
```

#### `EMPTY_TEXT`

Returned when the `text` field is present but empty or contains only whitespace.

```json
{"error": "Description cannot be empty", "code": "EMPTY_TEXT"}
```

**Common causes**:

- Empty string value: `{"text": ""}`
- Missing text field: `{}`
- Null value: `{"text": null}`

**Example of invalid requests**:

```bash
# Empty text field
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{"text": ""}'

# Missing text field entirely
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{}'
```

#### `DETECTION_FAILED`

Returned when the language detection algorithm cannot identify the language with sufficient confidence. This typically happens when:

```json
{"error": "Could not detect language with sufficient confidence", "code": "DETECTION_FAILED"}
```

**Common causes**:

- Text is in a language not included in the supported languages list
- Text is too short or ambiguous to determine language
- Text contains mixed languages
- Text consists mostly of numbers, symbols, or non-linguistic content
- Text contains transliterated content (e.g., romanized Japanese)

**Example scenarios**:

```bash
# Unsupported language (Hindi - not in supported list)
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{"text": "यह एक परीक्षण है।"}'

# Too short/ambiguous text
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{"text": "OK"}'

# Non-linguistic content
curl -X POST http://localhost:3000/detect -H "Content-Type: application/json" \
  -d '{"text": "12345 !@#$% ..."}'
```

**Resolution**: Ensure the text is in one of the [Supported Languages](#supported-languages) and contains enough linguistic content for accurate detection. Longer, more natural text produces better results.

### Error Handling Best Practices

When integrating with Lexis API, handle errors by checking the `code` field:

```javascript
const response = await fetch('/detect', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ text: userInput })
});

if (!response.ok) {
  const error = await response.json();
  switch (error.code) {
    case 'INVALID_JSON':
      console.error('Invalid request format');
      break;
    case 'EMPTY_TEXT':
      console.error('Text input is required');
      break;
    case 'DETECTION_FAILED':
      console.error('Could not detect language - try longer text');
      break;
  }
}
```

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

### You are allowed to

- Pull and run the official Docker image
- Use the software for personal, educational, or internal evaluation purposes
- Clone, modify, and build your own Docker images
- Self-host the software for non-commercial use only

### You are NOT allowed to

- Use this software in any commercial or revenue-generating product or service
- Offer the software as part of a paid platform or subscription
- Provide the software as a hosted or managed service (SaaS)
- Monetize the software directly or indirectly
- Sell access to the software or its functionality

Commercial use requires a separate commercial license.

For commercial licensing inquiries, contact: **[burak.tungut@tungops.com.tr](mailto:burak.tungut@tungops.com.tr)**
