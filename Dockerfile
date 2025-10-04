# Builder
FROM golang:1.24-alpine AS builder
WORKDIR /app

# CA sertifikalarını kopyalamak için gerekli
RUN apk --no-cache add ca-certificates

ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o main ./cmd/api

# Final
FROM alpine:latest

# SSL sertifikalarını ve timezone data'yı yükle
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Binary ve .env dosyasını kopyala
COPY --from=builder /app/main .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY .env .

EXPOSE 8080
CMD ["./main"]