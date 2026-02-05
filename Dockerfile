# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.26rc3-alpine3.22 AS builder

WORKDIR /btp-manager-workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . ./

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOFIPS140=v1.0.0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

ENV GODEBUG=fips140=only,tlsmlkem=0

WORKDIR /
COPY --chown=65532:65532 --from=builder /btp-manager-workspace/manager .
COPY --chown=65532:65532 --from=builder /btp-manager-workspace/module-chart ./module-chart
COPY --chown=65532:65532 --from=builder /btp-manager-workspace/module-resources ./module-resources
COPY --chown=65532:65532 --from=builder /btp-manager-workspace/manager-resources ./manager-resources
USER 65532:65532

ENTRYPOINT ["/manager"]
