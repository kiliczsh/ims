# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Install swag for documentation generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Generate swagger documentation
RUN swag init \
    --generalInfo cmd/server/main.go \
    --dir ./ \
    --output docs \
    --outputTypes go,json,yaml \
    --parseInternal \
    --quiet

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags '-extldflags "-static"' \
    -o bin/ims \
    cmd/server/main.go

# Production stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata postgresql-client

# Create non-root user
RUN addgroup -g 1000 -S ims && \
    adduser -u 1000 -S ims -G ims

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/ims /app/ims

# Copy documentation
COPY --from=builder /app/docs /app/docs

# Copy migration files
COPY --from=builder /app/migrations /app/migrations

# Copy scripts
COPY --from=builder /app/scripts /app/scripts

# Make scripts executable
RUN chmod +x /app/scripts/*.sh

# Change ownership to non-root user
RUN chown -R ims:ims /app

# Switch to non-root user
USER ims

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Run the application
CMD ["/app/ims"] 