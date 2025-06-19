#!/bin/bash

# Insider Message Sender - Development Setup Script
# This script sets up the development environment

set -e

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common utilities
source "$SCRIPT_DIR/common.sh"

print_info "🚀 Setting up Insider Message Sender development environment..."
echo

# Check required tools
print_info "Checking required tools..."

# Check Go
if ! command -v go &> /dev/null; then
    print_error "Go is not installed"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
print_success "Go $GO_VERSION installed"

# Check psql
if ! command -v psql &> /dev/null; then
    print_error "PostgreSQL client (psql) is not installed"
    echo "Install with:"
    echo "  macOS: brew install postgresql"
    echo "  Ubuntu: sudo apt-get install postgresql-client"
    exit 1
fi

print_success "PostgreSQL client installed"

# Check if we're in the right directory
if [[ ! -f "go.mod" ]]; then
    print_error "go.mod not found. Please run this script from the project root directory."
    exit 1
fi

# Download dependencies
print_info "Downloading Go dependencies..."
go mod download
go mod tidy
print_success "Dependencies installed"

# Create directories
print_info "Creating necessary directories..."
mkdir -p bin
mkdir -p logs
print_success "Directories created"

# Create .env file if it doesn't exist
ensure_env_file

# Create .env.example for reference
print_info "Creating .env.example for reference..."
cat > .env.example << EOF
# Insider Message Sender Configuration Example
# Copy this to .env and update with your actual values
# Usage: cp .env.example .env

# ===========================================
# REQUIRED CONFIGURATION
# ===========================================

# Database Configuration (REQUIRED)
# PostgreSQL connection string
DATABASE_URL=postgresql://postgres:password@localhost:5432/insider_messages

# Webhook Configuration (REQUIRED)
# Get a webhook URL from https://webhook.site
WEBHOOK_URL=https://webhook.site/your-unique-url
WEBHOOK_AUTH_KEY=INS.me1x9uMcyYGlhKKQVPoc.bO3j9aZwRTOcA2Ywo

# ===========================================
# OPTIONAL CONFIGURATION
# ===========================================

# Server Configuration
SERVER_PORT=8080
SERVER_READ_TIMEOUT=15s
SERVER_WRITE_TIMEOUT=15s

# Database Connection Pool
DATABASE_MAX_CONNECTIONS=25
DATABASE_MAX_IDLE_CONNECTIONS=5

# Redis Configuration (Optional - for caching)
# Leave empty to disable Redis caching
REDIS_URL=redis://localhost:6379/0
REDIS_CACHE_TTL=168h

# Webhook Settings
WEBHOOK_TIMEOUT=30s
WEBHOOK_MAX_RETRIES=3

# Scheduler Configuration
# How often to check for pending messages
SCHEDULER_INTERVAL=2m
# How many messages to process per batch
SCHEDULER_BATCH_SIZE=2

# Message Configuration
MESSAGE_MAX_LENGTH=160

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json
EOF

# Build the application
print_info "Building the application..."
    go build -o bin/ims cmd/server/main.go
print_success "Application built successfully"

# Make scripts executable
print_info "Making scripts executable..."
chmod +x scripts/*.sh
print_success "Scripts made executable"

print_success "🎉 Development environment setup complete!"
echo
print_info "Next steps:"
echo "1. Edit .env with your database and webhook configuration"
echo "2. Set up your database and run migrations:"
echo "   make migrate-with-data"
echo "3. Start the development server:"
echo "   make run"
echo
print_info "Useful commands:"
echo "  make help               - Show all available commands"
echo "  make migrate            - Run database migrations"
echo "  make migrate-with-data  - Run migrations with sample data"
echo "  make test               - Run tests"
echo "  make dev                - Run with hot reload (if air is installed)"
echo
print_info "For hot reload during development, install air:"
echo "  go install github.com/cosmtrek/air@latest"
echo 