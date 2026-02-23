FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata curl wget

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Copy config directory
COPY --from=builder /app/config ./config

EXPOSE 8080

CMD ["./main"]