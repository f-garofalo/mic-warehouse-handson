FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod .
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o mic-compat-adapter .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /src/mic-compat-adapter .
EXPOSE 8080
CMD ["./mic-compat-adapter"]
