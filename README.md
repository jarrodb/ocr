# ocr

A mock of the [Platerecognizer Snapshot Cloud API](https://guides.platerecognizer.com/docs/snapshot/api-reference/) for local development and integration testing. Drop-in replacement for `https://api.platerecognizer.com/v1/plate-reader/`.

Returns deterministic plate readings — fixture matches first, then a hashed generator that prefixes synthesized plates with `MCK` so tests can identify them by string.

[![CI](https://github.com/jarrodb/ocr/actions/workflows/ci.yml/badge.svg)](https://github.com/jarrodb/ocr/actions/workflows/ci.yml)
[![Release](https://github.com/jarrodb/ocr/actions/workflows/release.yml/badge.svg)](https://github.com/jarrodb/ocr/actions/workflows/release.yml)

## What it returns

`POST /v1/plate-reader/` resolves a plate through a two-tier ladder:

1. **Fixture** — substring match against `upload_url` + filename. Define your own in `config.yaml`.
2. **Generator** — synthetic plate prefixed `MCK` (e.g. `MCK7421`). Same input always produces the same plate.

Make/model/color/year are always seeded from a vehicle table, picked deterministically from the plate hash. The response shape matches the real Snapshot API field-for-field.

## Pull the image

```bash
docker pull ghcr.io/jarrodb/ocr:latest          # tracks main
docker pull ghcr.io/jarrodb/ocr:v0.1.1          # pin a version
```

Multi-arch: `linux/amd64`, `linux/arm64`.

## Run

```bash
docker run --rm -p 8080:8080 \
    -e API_TOKEN=mocktoken \
    ghcr.io/jarrodb/ocr:latest

curl -X POST http://localhost:8080/v1/plate-reader/ \
    -H 'Authorization: Token mocktoken' \
    -F 'upload_url=https://example.com/photos/tesla-model3.jpg' \
    -F 'mmc=true'
```

To bring your own fixtures, mount a `config.yaml`:

```bash
docker run --rm -p 8080:8080 \
    -v $PWD/config.yaml:/config/config.yaml \
    ghcr.io/jarrodb/ocr:latest
```

## Helm

```bash
helm install ocr oci://ghcr.io/jarrodb/charts/ocr --version 0.1.0
```

Chart values of note:

| Key | Default | Notes |
|---|---|---|
| `image.tag` | `latest` | Pin to a semver tag for reproducibility |
| `config.apiToken` | `mocktoken` | Sent as `Authorization: Token <value>` |
| `config.authRequired` | `true` | Cloud parity. Set false for on-prem SDK parity (no auth) |
| `config.generatedPlatePrefix` | `MCK` | First chars of any synthesized plate |
| `config.defaultRegion` | `us-ca` | Used when caller didn't specify regions |
| `config.fixtures` | `[]` | Substring-match rules |
| `config.vehicles` | seeded defaults | Make/model pool for synthesis |
| `gateway.enabled` | `true` | HTTPRoute on Gateway API; disable for raw Service-only deploys |

## Use it from Go

```go
import "github.com/jarrodb/ocr/pkg/client"

c, _ := client.New(client.Config{
    BaseURL: "http://ocr.svc.cluster.local:8080",   // or api.platerecognizer.com
    Token:   "mocktoken",
})

resp, err := c.Read(ctx, client.ReadParams{
    UploadURL: "https://example.com/car.jpg",
    Regions:   []string{"us-ca"},
    MMC:       true,
})
```

## Use it from the official Python client

```bash
pip install requests Pillow
curl -O https://raw.githubusercontent.com/parkpow/deep-license-plate-recognition/master/plate_recognition.py

python plate_recognition.py -a mocktoken -s http://localhost:8080 car.jpg
```

The script's `--sdk-url` mode does not send an `Authorization` header — that mirrors the real Platerecognizer on-prem SDK. Set `config.authRequired: false` on the mock when calling it this way, or run the mock with `auth_required: true` (default) and route the script through cloud mode instead.

## Auth contracts

The mock honors both contracts the real Platerecognizer offers:

| `auth_required` | Behavior | Use for |
|---|---|---|
| `true` (default) | Requires `Authorization: Token <api_token>`; 403 otherwise | Cloud-API parity — local dev that mirrors prod |
| `false` | No auth | On-prem SDK parity — for the Python client's `--sdk-url` mode |

## Endpoints

| Endpoint | Notes |
|---|---|
| `POST /v1/plate-reader/` | `upload` (multipart + base64), `upload_url`, `regions`, `mmc`, `camera_id`, `timestamp` |
| `GET /v1/statistics/` | `calls` / `month` / `total_calls` / `resets_on` |
| `GET /info/` | on-prem-style version + usage |
| `GET /status` | Debug: counters and mode (no auth, mock-only) |

## Development

```bash
just docker   # build and run the mock locally
just run      # `go run ./cmd` against config.yaml
just test     # Go unit tests (in Docker)
just e2e      # exercises the official Python SDK against the mock (in Docker)
just ci       # everything above + gofmt + vet
just tilt     # deploy to local kind via Helm
```

All test execution happens inside `Dockerfile.tests` — Python 3 and the vendored `plate_recognition.py` are baked in. The repo is bind-mounted; module/build caches live in named docker volumes so iterative runs stay fast.

## Releasing

Tag a version on `main`:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The `release.yml` workflow builds a multi-arch image (`amd64`, `arm64`) and publishes it plus the Helm chart to `ghcr.io/jarrodb`. `latest` always tracks `main`.

## Not implemented

Webhooks, multi-plate per image, on-prem-specific endpoints beyond `/info/`, and the advanced `config` engine flags (`mode=fast`, `detection_rule`, etc. — accepted but no behavior change).
