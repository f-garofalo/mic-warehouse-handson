# syntax=docker/dockerfile:1.6
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod .
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o mic-compat-adapter .

FROM alpine:3.20
RUN --mount=type=secret,id=netskope_ca \
    if [ -f /run/secrets/netskope_ca ]; then \
      cat /run/secrets/netskope_ca >> /etc/ssl/certs/ca-certificates.crt; \
    fi \
 && apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /src/mic-compat-adapter .
EXPOSE 8080
CMD ["./mic-compat-adapter"]
