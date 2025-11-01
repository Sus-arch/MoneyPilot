# Multi-stage Dockerfile for MoneyPilot API (build with Go, run on Alpine)
# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates
WORKDIR /src

# Download dependencies first (cached unless go.mod/go.sum change)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source and build the api binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags='-s -w' -o /app ./cmd/api

# Final minimal image
FROM alpine:3.18
RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /app /app

# Certificates for HTTPS/TLS clients if needed
RUN apk add --no-cache ca-certificates && update-ca-certificates

USER app
EXPOSE 8080
ENTRYPOINT ["/app"]
