# Lexis API

Lexis is a tiny HTTP “nanoservice” that detects the language of a given text. It’s built with Go + Fiber and uses `lingua-go` language models.

## Deploy directly into your kubernetes cluster with the helm chart we provide

### Prerequisites

- A Kubernetes cluster + `kubectl`
- Helm v3
- Network access to pull the container image (default: `ghcr.io/btungut/lexis-api:latest`)
- OCI support for Helm dependencies (Helm supports OCI by default in modern versions)

### Install / upgrade

From the repo root:

```bash
helm dependency update ./helm
helm upgrade --install lexis-api ./helm \
  --namespace lexis-api \
  --create-namespace
```

If you see authentication errors while Helm pulls the `helm-serve` dependency from GHCR, login first:

```bash
helm registry login ghcr.io
```

### Configuration

The chart values live in [helm/values.yaml](helm/values.yaml). The most common knobs:

- `deployment.image.repository` / `deployment.image.tag`
- `deployment.env.PORT` (can be `4001` or `:4001`; the app will add the `:` if missing)
- `deployment.env.MAX_CHAR_PROCESS` (defaults to `2000`)
- `service.type` / `service.port`
- `ingress.enabled` and `ingress.rule.*`

Example override file:

```yaml
# values.local.yaml
shared:
  App_Port: 4001

deployment:
  replicaCount: 2
  image:
    repository: ghcr.io/btungut/lexis-api
    tag: latest
  env:
    PORT: ":4001"
    MAX_CHAR_PROCESS: "2000"

service:
  type: ClusterIP
  port: 80

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

Apply it:

```bash
helm upgrade --install lexis-api ./helm \
  --namespace lexis-api \
  --create-namespace \
  -f values.local.yaml
```

### Verify

Health endpoint:

```bash
# Find the Service name and port
kubectl -n lexis-api get svc

# (Optional) port-forward to your machine
kubectl -n lexis-api port-forward svc/$(kubectl -n lexis-api get svc -o jsonpath='{.items[0].metadata.name}') 8080:80

curl -i http://127.0.0.1:8080/health
```

## Building and testing this Go code

### Requirements

- Go (see [go.mod](go.mod); currently `go 1.25.5`)

### Run tests

```bash
go test ./...
```

Run a single test:

```bash
go test -run TestDetectEndpoint ./...
```

### Build

```bash
go build -o lexis-api .
```

### Run locally

`PORT` can be provided as `3000` or `:3000` (the app will normalize it).

```bash
export PORT="3000"
export MAX_CHAR_PROCESS="2000"
./lexis-api
```

### Build and run with Docker

```bash
docker build -t lexis-api:local .
docker run --rm -p 3000:3000 -e PORT="3000" -e MAX_CHAR_PROCESS="2000" lexis-api:local
```

## API

### `GET /health`

Returns `200` when the service is up.

### `POST /detect`

Request body:

```json
{"text":"Hello, this is a test."}
```

Response body:

```json
{"language":"en","confidence":0.97}
```

Notes:

- `language` is ISO 639-1 lowercase (e.g. `en`, `tr`, `es`) or `unknown`.
- Very large text is truncated to `MAX_CHAR_PROCESS` runes to avoid CPU spikes.

## Supported languages

At startup, Lexis preloads language models for a curated set of common languages (English, Turkish, German, French, Spanish, Italian, Portuguese, Russian, Arabic, Chinese, Japanese, Korean, Dutch, Azerbaijani, Persian). If you want broader coverage, you can change the detector builder in [main.go](main.go).

## Contributing

PRs are welcome.

- Keep changes focused and documented.
- Prefer adding/adjusting tests in [main_test.go](main_test.go) when behavior changes.

## Security

If you discover a security issue, please avoid filing a public issue. Prefer a private report to the maintainers.
