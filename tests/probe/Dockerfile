FROM --platform=$BUILDPLATFORM golang:1.26.4-alpine3.22 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOFIPS140=v1.0.0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /probe ./tests/probe/

FROM gcr.io/distroless/static:nonroot
ENV GODEBUG=fips140=only,tlsmlkem=0
COPY --chown=65532:65532 --from=builder /probe /probe
USER 65532:65532
ENTRYPOINT ["/probe"]
