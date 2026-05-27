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
                                          ┌──────────┴──────────┐
                                          ▼                     ▼
                                     fixture                generator
                                     (config)               (MCK-prefix)
```

Engine ladder:

1. **Fixture** — substring match against `upload_url` + filename. Deterministic, fixture-defined plate and MMC.
2. **Generator** — synthetic plate prefixed `MCK` (configurable). MMC picked deterministically from the vehicle seed table by hashing the plate. This tier cannot fail.

The mock does not perform real OCR — generic OCR without plate localization can't read photos of cars. MMC (make / model / color / year / type) comes from the vehicle seed table, picked deterministically from the plate hash. Real PR Cloud uses trained CNNs that have no comparable OSS drop-in.

## Setup

```bash
go mod download
```

## Run

```bash
just run         # local Go process, reads ./config.yaml
just docker      # build and run the alpine image
just tilt        # deploy to local kind via Helm
```

Mock endpoints (token: `mocktoken`):

```
POST   /v1/plate-reader/        # multipart: upload | upload_url, regions, mmc, camera_id
GET    /v1/statistics/          # usage counters
GET    /info/                   # version / license info (mirrors on-prem SDK)
GET    /status                  # debug: counters + mode (no auth)
```

## Test

All tests run inside the `Dockerfile.tests` image with the repo bind-mounted; Go module and build caches live in named docker volumes (`ocr-gomod`, `ocr-gocache`) so re-runs are fast.

```bash
just test    # Go unit tests
just e2e     # exercises the official Python SDK (parkpow/deep-license-plate-recognition) against the mock
just ci      # gofmt + vet + lint + test + e2e
```

The Python SDK tests use the script's `--sdk-url` mode, which does not send `Authorization`. The mock therefore runs with `auth_required: false` for those tests — the on-prem SDK parity mode. The Go SDK tests cover the cloud/Token contract separately.

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
