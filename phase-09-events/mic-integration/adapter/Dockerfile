FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o adapter .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /build/adapter /app/adapter
EXPOSE 8080
CMD ["/app/adapter"]
