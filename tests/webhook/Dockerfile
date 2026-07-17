FROM golang:1.26.5-alpine3.24 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOFIPS140=v1.0.0 go build -o /webhook ./tests/webhook/

FROM gcr.io/distroless/static:nonroot
ENV GODEBUG=fips140=only,tlsmlkem=0
COPY --from=builder /webhook /webhook
ENTRYPOINT ["/webhook"]
