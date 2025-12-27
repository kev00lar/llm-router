# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder
WORKDIR /app

# Required for fetching dependencies
RUN apk add --no-cache git

# Copy dependency files first
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build a statically linked binary targeting the correct path
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o llm-router ./cmd/router/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser
WORKDIR /home/appuser/

# 1. Copy the binary
COPY --from=builder /app/llm-router .

# 2. IMPORTANT: Copy the admin folder so StaticFile can find it
COPY --from=builder /app/admin ./admin

EXPOSE 3000
CMD ["./llm-router"]