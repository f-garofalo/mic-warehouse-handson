# syntax=docker/dockerfile:1.6
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=secret,id=netskope_ca \
    if [ -f /run/secrets/netskope_ca ]; then \
      cat /run/secrets/netskope_ca >> /etc/ssl/certs/ca-certificates.crt; \
    fi \
 && go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest
RUN --mount=type=secret,id=netskope_ca \
    if [ -f /run/secrets/netskope_ca ]; then \
      cat /run/secrets/netskope_ca >> /etc/ssl/certs/ca-certificates.crt; \
    fi \
 && apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /build/app .
# The Hermes publisher validates every event against these Data Product
# schemas at runtime (events/data_product.go loads ./schemas at startup).
COPY --from=builder /build/schemas ./schemas
EXPOSE 8081
CMD ["./app"]
