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

# Final stage - minimal scratch container
FROM scratch

# Copy the binary from builder stage
COPY --from=builder /app/github-stars-notify /github-stars-notify

# Add labels for better maintainability
LABEL org.opencontainers.image.title="GitHub Stars Notify"
LABEL org.opencontainers.image.description="A service to notify about GitHub repository stars"
LABEL org.opencontainers.image.source="https://github.com/mydoomfr/github-stars-notify"
LABEL org.opencontainers.image.vendor="mydoomfr"

# Default command
ENTRYPOINT ["/github-stars-notify"] 