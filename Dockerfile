# Build the controller binary
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY api/ api/
COPY cmd/ cmd/
COPY controllers/ controllers/
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -a -o tenant-controller ./cmd

# Minimal runtime image, runs as nonroot
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/tenant-controller .
USER 65532:65532
ENTRYPOINT ["/tenant-controller"]
