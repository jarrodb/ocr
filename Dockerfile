# Multi-arch image. The builder stage pins to the host's native arch
# ($BUILDPLATFORM) and uses Go's own cross-compiler to emit the target binary.
# That avoids QEMU emulating the entire Go toolchain — orders of magnitude
# faster than cross-builds that run the toolchain under emulation.
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o ocr-mock cmd/main.go

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/ocr-mock /ocr-mock
EXPOSE 8080
CMD ["/ocr-mock"]
