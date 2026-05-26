# AGENTS.md

## Project

Mock of the Platerecognizer Snapshot Cloud API plus a Go SDK that consumes it. Drop-in replacement for `https://api.platerecognizer.com/v1/plate-reader/` against an in-cluster mock pod, for local development and integration tests.

Pattern follows `~/.local/dev/twilio` and `~/.local/dev/auth0`: container ships the mock service, an SDK in `pkg/client/` is the canonical Go consumer, and tests drive the mock via the SDK.

## Architecture

```
                       POST /v1/plate-reader/
client (pkg/client)  ─────────────────────────►  pkg/server
                                                     │
                                                     ▼
                                                 pkg/ocr (Engine ladder)
                                                     │
                                          ┌──────────┼──────────┐
                                          ▼          ▼          ▼
                                     fixture     tesseract   generator
                                     (config)   (subprocess) (MCK-prefix)
```

Engine ladder:

1. **Fixture** — substring match against `upload_url` + filename. Deterministic, fixture-defined plate and MMC.
2. **Tesseract** — subprocess call to bundled `tesseract` binary. PSM 8 (single word) + TSV output → highest-confidence word. Below `min_confidence` falls through.
3. **Generator** — synthetic plate prefixed `MCK` (configurable). MMC picked deterministically from the vehicle seed table by hashing the plate. This tier cannot fail.

The MMC fields (make / model / color / year / type) are always synthesized — Tesseract reads text, not cars. Real PR Cloud uses trained CNNs that have no comparable OSS drop-in.

## Setup

```bash
go mod download
# For real OCR locally:
brew install tesseract       # macOS
apk add tesseract-ocr tesseract-ocr-data-eng   # alpine
```

## Run

```bash
just run         # local Go process, reads ./config.yaml
just docker      # build and run the alpine image (tesseract bundled)
just tilt        # deploy to local kind via Helm
```

Mock endpoints (token: `mocktoken`):

```
POST   /v1/plate-reader/        # multipart: upload | upload_url, regions, mmc, camera_id
GET    /v1/statistics/          # usage counters
GET    /info/                   # version / license info (mirrors on-prem SDK)
GET    /status                  # debug: counters + tesseract availability (no auth)
```

## Test

All tests run inside the `Dockerfile.tests` image with the repo bind-mounted; Go module and build caches live in named docker volumes (`ocr-gomod`, `ocr-gocache`) so re-runs are fast.

```bash
just test    # Go unit tests
just e2e     # exercises the official Python SDK (parkpow/deep-license-plate-recognition) against the mock
just ci      # gofmt + vet + test + e2e
```

The Python SDK tests use the script's `--sdk-url` mode, which does not send `Authorization`. The mock therefore runs with `auth_required: false` for those tests — the on-prem SDK parity mode. The Go SDK tests cover the cloud/Token contract separately.

`TestPlateReader_TesseractPath` is skipped unless a test fixture image is added to `testdata/` (see TODO at bottom).

## Principles

- Library-first: prefer existing packages over custom implementations. Review for duplication.
- No code without a plan. Investigation and implementation are separate steps.
- Own the problem. Never dismiss environment issues.
- Research before action: check current best practices and latest stable versions before making changes.
- Run linters and tests before completing work.

## Code Conventions

### Git

- Feature branches from main: `feat/`, `fix/`, `chore/`, `hotfix/`
- Never commit to main. Never push without explicit ask.
- Conventional commit prefixes: `feat:`, `fix:`, `docs:`, `test:`, `chore:`, `refactor:`
- Single line, concise, imperative subjects.
- No Co-Authored-By trailers.
- Squash merge preferred.

### Testing

- Tests drive the mock via the Go SDK in `pkg/client/`, not raw HTTP.
- Real OCR path is exercised only when tesseract is present; otherwise the test is `t.Skip`ed.
- No `time.Sleep` for ordering — the engine ladder is synchronous; tests assert on response.

### Kubernetes

- Gateway API HTTPRoute only. No raw Ingress.
- Chart is environment-agnostic; values overrides drive local vs stage vs prod.
- Local Tilt uses `kind-jarrodb` context (allowlisted in Tiltfile).

## API Surface

Snapshot Cloud subset implemented:

| Endpoint | Status |
|---|---|
| `POST /v1/plate-reader/` | upload (multipart + base64), upload_url, regions, mmc, camera_id, timestamp |
| `GET /v1/statistics/` | calls / month / total_calls / resets_on |
| `GET /info/` | on-prem-style version + usage |
| Webhooks | not yet implemented |
| `mode=redaction`, `mode=fast`, advanced config | accepted, no behavior change |

## OCR Engine Notes

- Tesseract is bundled into the image (~85 MB). Subprocess invocation keeps the Go binary CGO-free.
- `tesseract.enabled: false` short-circuits to fixture→generator only.
- For higher accuracy, swap `pkg/ocr/tesseract.go` for an ONNX Runtime plate-OCR pipeline — the `Engine` interface is the swap point.
- Make/Model/Color/Year are seeded, not detected. Replacing them with a real classifier requires shipping CNN weights (~100–200 MB) and is outside the mock's scope.
