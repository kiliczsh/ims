# IMS - Development Guide

## Table of Contents
1. [Architecture](#architecture)
2. [Development Setup](#development-setup)
3. [Database Schema](#database-schema)
4. [Configuration](#configuration)
5. [API Development](#api-development)
6. [Testing](#testing)
7. [Build & Deployment](#build--deployment)
8. [Troubleshooting](#troubleshooting)

## Architecture

### System Overview
The IMS (Insider Message Sender) is a Go-based microservice that automatically processes and sends messages from a database queue at configurable intervals, with RESTful API endpoints for control and monitoring.

### Component Architecture
```
┌─────────────────────────────────────────────────────────────────────────┐
│                          API Gateway Layer                               │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────┐           ┌──────────────────────────┐        │
│  │   Control API       │           │    Monitoring API        │        │
│  │  POST /api/control  │           │  GET /api/messages/sent  │        │
│  │  - start/stop       │           │  - list sent messages    │        │
│  └──────────┬──────────┘           └──────────┬───────────────┘        │
├─────────────┴──────────────────────────────────┴────────────────────────┤
│                         Service Layer                                    │
│  ┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────┐ │
│  │  Control Service    │  │  Message Service    │  │ Audit Service   │ │
│  │  - Start/Stop       │  │  - Fetch Messages   │  │ - Log Events    │ │
│  │  - Status Check     │  │  - Send Messages    │  │ - Statistics    │ │
│  └──────────┬──────────┘  │  - Update Status    │  └────────┬────────┘ │
│             │              └──────────┬──────────┘           │          │
│  ┌──────────▼──────────────────────────▼──────────────────────▼────────┐ │
│  │                    Scheduler Component                               │ │
│  │  - Custom Timer (configurable intervals)                            │ │
│  │  - Goroutine Management & Graceful Shutdown                         │ │
│  └──────────────────────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────────────────┤
│                        Repository Layer                                  │
│  ┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────┐ │
│  │ Message Repository  │  │  Cache Repository   │  │ Audit           │ │
│  │ (PostgreSQL)        │  │  (Redis - Optional) │  │ Repository      │ │
│  └─────────────────────┘  └─────────────────────┘  └─────────────────┘ │
└──────────────────────────────────────────────────────────────────────────┘
```

### Project Structure
```
├── bin/                     # Built binaries (generated)
├── cmd/server/              # Application entry point
├── docs/                    # Generated API documentation
├── internal/                # Internal application code
│   ├── config/              # Configuration management
│   ├── domain/              # Domain models and types
│   ├── handlers/            # HTTP request handlers
│   ├── middleware/          # HTTP middleware
│   ├── repository/          # Data access layer
│   │   ├── postgres/        # PostgreSQL implementations
│   │   └── redis/           # Redis cache implementations
│   ├── scheduler/           # Message scheduling logic
│   ├── server/              # HTTP server setup
│   └── service/             # Business logic services
├── migrations/              # Database migration files
└── scripts/                 # Utility scripts
    ├── run.sh               # Application runner script
    ├── migrate.sh           # Database migration script
    ├── setup-dev.sh         # Development environment setup
    └── common.sh            # Shared script utilities
```

## Development Setup

### Prerequisites
- **Go 1.21+** (`go version`)
- **PostgreSQL 12+** (local or remote)
- **Redis 6+** (optional, for caching)
- **Git** for version control

### Quick Start
```bash
# 1. Clone and setup
git clone <repository-url>
cd ims

# 2. Setup development environment
make setup-dev

# 3. Edit configuration
cp .env.example .env
# Edit .env with your database and webhook URLs

# 4. Run database migrations
make migrate

# 5. Build with documentation
make build

# 6. Start the service
make run
```

### Development Tools
```bash
# Install development tools
go install github.com/cosmtrek/air@latest           # Hot reload
go install github.com/swaggo/swag/cmd/swag@latest  # API docs
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest  # Linting

# Development workflow
make dev          # Run with hot reload
make test         # Run tests
make lint         # Run linter
make swagger      # Generate API docs
```

## Database Schema

### Main Tables
```sql
-- Messages table
CREATE TYPE message_status AS ENUM ('pending', 'sending', 'sent', 'failed');

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    status message_status NOT NULL DEFAULT 'pending',
    message_id VARCHAR(255),
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Audit logs table
CREATE TYPE audit_event_type AS ENUM (
    'batch_started', 'batch_completed', 'message_sent', 
    'message_failed', 'scheduler_started', 'scheduler_stopped'
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type audit_event_type NOT NULL,
    event_name VARCHAR(100) NOT NULL,
    description TEXT,
    batch_id VARCHAR(255),
    message_id VARCHAR(255),
    request_id VARCHAR(255),
    http_method VARCHAR(10),
    endpoint VARCHAR(255),
    status_code INTEGER,
    duration_ms INTEGER,
    message_count INTEGER,
    success_count INTEGER,
    failure_count INTEGER,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### Indexes for Performance
```sql
-- Message indexes
CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_messages_created_at ON messages(created_at);
CREATE INDEX idx_messages_status_created ON messages(status, created_at);

-- Audit indexes
CREATE INDEX idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_batch_id ON audit_logs(batch_id);
```

### Migration System
```bash
# Add new migration
./scripts/migrate.sh

# Check migration status
psql $DATABASE_URL -c "SELECT * FROM migration_history ORDER BY applied_at;"

# Migration with sample data
./scripts/migrate.sh --with-sample-data
```

## Configuration

### Environment Variables
```bash
# Required Configuration
DATABASE_URL=postgresql://postgres:password@localhost:5432/insider_messages
WEBHOOK_URL=https://webhook.site/your-unique-url
WEBHOOK_AUTH_KEY=your-secret-api-key

# Server Configuration  
SERVER_PORT=8080
SERVER_READ_TIMEOUT=15s
SERVER_WRITE_TIMEOUT=15s

# Optional Configuration
REDIS_URL=redis://localhost:6379/0
SCHEDULER_INTERVAL=2m
SCHEDULER_BATCH_SIZE=2
MESSAGE_MAX_LENGTH=160
WEBHOOK_TIMEOUT=30s
WEBHOOK_MAX_RETRIES=3
LOG_LEVEL=info
LOG_FORMAT=json
```

### Configuration Structure
```go
type Config struct {
    Server    ServerConfig
    Database  DatabaseConfig
    Redis     RedisConfig
    Webhook   WebhookConfig
    Scheduler SchedulerConfig
    Message   MessageConfig
}
```

## API Development

### Adding New Endpoints
1. **Define handler** in `internal/handlers/`
2. **Add swagger annotations**:
```go
// @Summary      Endpoint summary
// @Description  Detailed description
// @Tags         tag-name
// @Accept       json
// @Produce      json
// @Param        param query string false "Parameter description"
// @Success      200 {object} ResponseType
// @Failure      400 {object} ErrorResponse
// @Security     ApiKeyAuth
// @Router       /endpoint [get]
func (h *Handler) NewEndpoint(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```
3. **Register route** in `internal/server/server.go`
4. **Regenerate docs**: `make swagger`

### Authentication
- All protected endpoints require `Authorization` header
- Use the API key from your `.env` file
- Public endpoint: `/api/health`

### Response Formats
```go
// Success Response
type SuccessResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Message string      `json:"message,omitempty"`
}

// Error Response
type ErrorResponse struct {
    Error   string `json:"error"`
    Code    int    `json:"code,omitempty"`
    Details string `json:"details,omitempty"`
}
```

## Testing

### Running Tests
```bash
# All tests
make test

# Tests with coverage
make test-coverage

# Specific package
go test -v ./internal/service/...

# Integration tests
go test -v -tags=integration ./...
```

### Test Structure
```
internal/
├── handlers/
│   ├── handler.go
│   └── handler_test.go
├── service/
│   ├── service.go
│   └── service_test.go
└── repository/
    ├── repository.go
    └── repository_test.go
```

### Test Patterns
```go
func TestMessageService_SendMessage(t *testing.T) {
    // Arrange
    mockRepo := &MockMessageRepository{}
    service := NewMessageService(mockRepo, nil, nil, 160)
    
    // Act
    result, err := service.SendMessage(context.Background(), message)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

## Build & Deployment

### Local Development
```bash
# Development with hot reload
make dev

# Manual build and run
make build
./bin/ims

# Using legacy script
make run-script
```

### Docker Development
```bash
# Start all services (PostgreSQL, Redis, IMS)
make docker-up

# View logs
make docker-logs

# Run migrations
make docker-migrate

# Stop services
make docker-down

# Clean resources
make docker-clean
```

### Production Build

#### Native Binary
```bash
# Build optimized binary
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/ims cmd/server/main.go

# Cross-platform builds
GOOS=darwin GOARCH=amd64 go build -o bin/ims-darwin cmd/server/main.go
GOOS=windows GOARCH=amd64 go build -o bin/ims.exe cmd/server/main.go
```

#### Docker Production
```bash
# Build Docker image
make docker-build

# Build with version tag
make docker-build-tag TAG=v1.0.0

# Production deployment
cp docker/.env.template .env
# Edit .env with production values
make docker-up-prod

# Monitor production
make docker-logs-prod
make docker-status
```

### Container Registry
```bash
# Tag for registry
docker tag ims:latest your-registry.com/ims:v1.0.0

# Push to registry
docker push your-registry.com/ims:v1.0.0

# Pull and deploy
docker pull your-registry.com/ims:v1.0.0
IMS_IMAGE=your-registry.com/ims:v1.0.0 make docker-up-prod
```

### Deployment Checklist
- [ ] Environment variables configured
- [ ] Database migrations applied
- [ ] SSL certificates in place (if not using reverse proxy)
- [ ] Monitoring and logging configured
- [ ] Health checks enabled
- [ ] Backup strategy implemented
- [ ] Docker images built and pushed
- [ ] Resource limits configured
- [ ] Network security configured

## Troubleshooting

### Common Issues

**Build Failures**:
```bash
# Clean and rebuild
make clean
go mod tidy
make build
```

**Database Connection**:
```bash
# Test connection
psql $DATABASE_URL -c "SELECT 1"

# Check migrations
psql $DATABASE_URL -c "SELECT * FROM migration_history"
```

**Swagger Generation**:
```bash
# Reinstall swag
go install github.com/swaggo/swag/cmd/swag@latest
make swagger-gen
```

**Performance Issues**:
- Check database indexes
- Monitor connection pools
- Review batch sizes
- Check webhook timeouts

### Logging
```bash
# View logs
tail -f /var/log/ims/app.log

# JSON log parsing
tail -f app.log | jq '.'

# Filter by level
tail -f app.log | jq 'select(.level=="error")'
```

### Monitoring Endpoints
- Health check: `GET /api/health`
- Metrics: Built-in Go metrics
- Audit logs: `GET /api/audit/stats`

### Development Workflow
1. **Feature Development**:
   - Create feature branch
   - Add tests first (TDD)
   - Implement feature
   - Add swagger docs
   - Update migration if needed

2. **Code Quality**:
   ```bash
   make fmt         # Format code
   make lint        # Run linter
   make test        # Run tests
   make swagger     # Generate docs
   ```

3. **Integration**:
   - Test with real database
   - Verify webhook integration
   - Test scheduler functionality
   - Review API documentation

This guide provides comprehensive technical information for developing, testing, and deploying the IMS application. 