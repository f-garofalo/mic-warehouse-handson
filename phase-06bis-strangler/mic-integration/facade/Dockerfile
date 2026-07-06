# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN --mount=type=secret,id=netskope_ca,target=/usr/local/share/ca-certificates/netskope.crt \
    update-ca-certificates 2>/dev/null; \
    CGO_ENABLED=0 go build -o /facade .

FROM alpine:3.20
COPY --from=builder /facade /facade
EXPOSE 80
ENTRYPOINT ["/facade"]
