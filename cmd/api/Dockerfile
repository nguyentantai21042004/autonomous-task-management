# syntax=docker/dockerfile:1

# =============================================================================
# STAGE 1: BUILDER
# Tip: --platform=$BUILDPLATFORM ensures Docker always uses the Go image that matches your machine's chip (e.g. Apple Silicon M4)
# so the compiler runs at full speed with no need for slow emulation (QEMU).
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /app

# 1. Declare Docker Buildx automatic ARG variables
# When you build with --platform linux/amd64, Docker will automatically set "linux" for TARGETOS and "amd64" for TARGETARCH
ARG TARGETOS
ARG TARGETARCH

# Install git and tzdata (will be copied to runtime image)
RUN apk add --no-cache git tzdata

# 2. Cache & Download Dependencies + Install Swag
# Using cache mount prevents downloading dependencies on every build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && \
    go install github.com/swaggo/swag/cmd/swag@v1.8.12

# 3. Copy Source Code
COPY . .

# 4. Generate Swagger Documentation
RUN swag init -g cmd/api/main.go

# 5. Build Binary (Explicitly specify GOOS and GOARCH from Docker ARGs)
# CGO_ENABLED=0: Required to run with Distroless Static (no C libraries needed)
# -ldflags="-s -w": Remove debug info to reduce binary size (~30-40% smaller)
# --mount=type=cache: Reuses build cache between builds → Much faster!
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -ldflags="-s -w" \
    -o api ./cmd/api

# =============================================================================
# STAGE 2: RUNTIME (Distroless Static - Ultra lightweight & Secure)
# - No shell, no package manager → Minimal attack surface
# - Image size ~2MB (vs Alpine ~20MB)
# - Contains only: binary + ca-certs + timezone + user
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

# Copy timezone data (because distroless static doesn't have it by default)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Ho_Chi_Minh

# Copy the compiled binary for the correct architecture
COPY --from=builder --chown=nonroot:nonroot /app/api .

# Run as non-root (UID 65532)
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["./api"]