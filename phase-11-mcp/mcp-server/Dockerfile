# syntax=docker/dockerfile:1.6
FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=secret,id=netskope_ca \
    if [ -f /run/secrets/netskope_ca ]; then \
      cat /run/secrets/netskope_ca >> /etc/ssl/certs/ca-certificates.crt; \
    fi \
 && go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o mcp-server .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /build/mcp-server .
# No EXPOSE: this server speaks JSON-RPC on stdin/stdout, not on a port.
ENTRYPOINT ["./mcp-server"]
