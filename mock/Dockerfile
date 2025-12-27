# Stage 1: Build
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o router ./cmd/router

# Stage 2: Final Image
FROM alpine:latest
RUN adduser -D -u 1000 appuser
USER appuser
COPY --from=builder /app/router /router
EXPOSE 3000
ENTRYPOINT ["/router"]