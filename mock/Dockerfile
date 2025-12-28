FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
# Explicitly build the mock main.go
RUN go build -o mock-server ./mock/main.go

FROM alpine:latest
COPY --from=builder /app/mock-server /mock-server
EXPOSE 8080
CMD ["/mock-server"]