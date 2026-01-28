FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS backend
WORKDIR /backend-build

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files and download dependencies (cached layer)
COPY go.mod go.sum ./
# Use Go proxy for China (fallback to direct)
RUN --mount=type=cache,target=/go/pkg/mod \
    GOPROXY=https://goproxy.cn,https://goproxy.io,direct \
    go mod download

# Copy source code (use .dockerignore to exclude unnecessary files)
COPY . .

# Please build frontend first, so that the static files are available.
# Refer to `pnpm release` in package.json for the build command.
ARG TARGETOS TARGETARCH VERSION COMMIT
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
      -trimpath \
      -ldflags="-s -w -extldflags '-static'" \
      -tags netgo,osusergo \
      -o divinesense \
      ./cmd/divinesense

# Use minimal Alpine with security updates
FROM alpine:3.21 AS monolithic
WORKDIR /usr/local/divinesense

# Install runtime dependencies and create non-root user in single layer
RUN apk add --no-cache tzdata ca-certificates && \
    addgroup -g 10001 -S nonroot && \
    adduser -u 10001 -S -G nonroot -h /var/opt/divinesense nonroot && \
    mkdir -p /var/opt/divinesense && \
    chown -R nonroot:nonroot /var/opt/divinesense

# Copy binary and entrypoint
COPY --from=backend /backend-build/divinesense /usr/local/divinesense/divinesense
COPY --from=backend --chmod=755 /backend-build/scripts/entrypoint.sh /usr/local/divinesense/entrypoint.sh

# Switch to non-root user
USER nonroot:nonroot

# Data directory
VOLUME /var/opt/divinesense

ENV TZ="UTC" \
    MEMOS_MODE="prod" \
    MEMOS_PORT="5230"

EXPOSE 5230

ENTRYPOINT ["/usr/local/divinesense/entrypoint.sh", "/usr/local/divinesense/divinesense"]
