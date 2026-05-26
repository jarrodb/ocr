set shell := ["bash", "-uc"]

tests_image  := "ocr-tests:dev"
mod_volume   := "ocr-gomod"
build_volume := "ocr-gocache"

default:
	@echo "Usage:"
	@echo "  just docker - Build & run the mock service container"
	@echo "  just run    - Run the mock locally (no docker; for quick iteration)"
	@echo "  just test   - Run Go unit tests (in Docker)"
	@echo "  just e2e    - Run end-to-end tests against the official Python SDK (in Docker)"
	@echo "  just ci     - Format + vet + test + e2e"
	@echo "  just tilt   - Start Tilt on the local kind cluster"
	@echo "  just down   - Stop the docker mock"

docker:
	docker build -t ocr-mock:dev .
	docker run --rm -d \
		--name ocr-mock \
		-p 8080:8080 \
		-e API_TOKEN=mocktoken \
		ocr-mock:dev
	@echo "POST http://localhost:8080/v1/plate-reader/"

run:
	go run ./cmd

down:
	docker stop ocr-mock 2>/dev/null || true

tilt:
	tilt up

test:
	@docker image inspect {{tests_image}} >/dev/null 2>&1 || docker build -f Dockerfile.tests -t {{tests_image}} .
	@docker run --rm \
		-v "$PWD:/work" \
		-v {{mod_volume}}:/go/pkg/mod \
		-v {{build_volume}}:/root/.cache/go-build \
		-w /work \
		{{tests_image}} go test -race -skip TestPythonSDK ./pkg/...

e2e:
	@docker image inspect {{tests_image}} >/dev/null 2>&1 || docker build -f Dockerfile.tests -t {{tests_image}} .
	@docker run --rm \
		-v "$PWD:/work" \
		-v {{mod_volume}}:/go/pkg/mod \
		-v {{build_volume}}:/root/.cache/go-build \
		-w /work \
		{{tests_image}} go test -v -run TestPythonSDK ./pkg/server/

ci:
	@docker image inspect {{tests_image}} >/dev/null 2>&1 || docker build -f Dockerfile.tests -t {{tests_image}} .
	@docker run --rm \
		-v "$PWD:/work" \
		-v {{mod_volume}}:/go/pkg/mod \
		-v {{build_volume}}:/root/.cache/go-build \
		-w /work \
		{{tests_image}} bash -c '\
			set -e; \
			echo "==> gofmt"; test -z "$(gofmt -l .)" || { gofmt -l .; exit 1; }; \
			echo "==> vet"; go vet ./...; \
			echo "==> lint"; golangci-lint run ./...; \
			echo "==> test"; go test -race -skip TestPythonSDK ./pkg/...; \
			echo "==> e2e"; go test -v -run TestPythonSDK ./pkg/server/; \
		'
