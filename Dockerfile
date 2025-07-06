# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application for the target platform
# Docker will automatically set TARGETARCH and TARGETOS
ARG TARGETARCH
ARG TARGETOS
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -installsuffix cgo -ldflags="-w -s" -o github-stars-notify .

# Final stage - minimal container with CA certificates
FROM alpine:3.21.1

# Install CA certificates
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Copy the binary from builder stage
COPY --from=builder /app/github-stars-notify /app/github-stars-notify

# Set ownership and make executable
RUN chown appuser:appuser /app/github-stars-notify && \
    chmod +x /app/github-stars-notify && \
    mkdir -p /app/data && \
    chown -R appuser:appuser /app/data

# Switch to non-root user
USER appuser

# Set working directory
WORKDIR /app

# Add labels for better maintainability
LABEL org.opencontainers.image.title="GitHub Stars Notify"
LABEL org.opencontainers.image.description="A service to notify about GitHub repository stars"
LABEL org.opencontainers.image.source="https://github.com/mydoomfr/github-stars-notify"
LABEL org.opencontainers.image.vendor="mydoomfr"

# Default command
ENTRYPOINT ["/app/github-stars-notify"] 