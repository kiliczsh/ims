# IMS - Development Guide

## Table of Contents
1. [Architecture](#architecture)
2. [Development Setup](#development-setup)
3. [Database Schema](#database-schema)
4. [Queue Systems](#queue-systems)
5. [Configuration](#configuration)
6. [API Development](#api-development)
7. [Testing](#testing)
8. [Build & Deployment](#build--deployment)
9. [Docker Development](#docker-development)
10. [Troubleshooting](#troubleshooting)

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
│   ├── queue/               # Queue abstraction layer
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
- **RabbitMQ 3.13+** (optional, for high-performance queue)
- **Docker & Docker Compose** (recommended)
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
CREATE TYPE message_status AS ENUM ('pending', 'sending', 'sent', 'failed', 'dead_letter');

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    status message_status NOT NULL DEFAULT 'pending',
    message_id VARCHAR(255),
    retry_count INTEGER DEFAULT 0,
    last_retry_at TIMESTAMP WITH TIME ZONE,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    failure_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    sent_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Dead letter queue table
CREATE TABLE dead_letter_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_message_id UUID NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    failure_reason TEXT NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_retry_at TIMESTAMP WITH TIME ZONE,
    original_created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    moved_to_dlq_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
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
CREATE INDEX idx_messages_next_retry_at ON messages(next_retry_at);

-- Dead letter queue indexes
CREATE INDEX idx_dlq_original_message_id ON dead_letter_messages(original_message_id);
CREATE INDEX idx_dlq_moved_at ON dead_letter_messages(moved_to_dlq_at);

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

## Queue Systems

IMS supports two queue implementations: Database Queue (default) and RabbitMQ Queue (optional).

### Queue Abstraction

The system uses interfaces for queue operations:

```go
type MessageQueue interface {
    Publish(ctx context.Context, message domain.Message) error
    StartConsumer(ctx context.Context, handler MessageHandler) error
    Health() error
    Close() error
}

type QueueManager interface {
    GetQueue() MessageQueue
    Close() error
}
```

### Database Queue (Default)

Uses PostgreSQL with polling for message processing.

**Characteristics:**
- ✅ Simple setup, no additional dependencies
- ✅ ACID compliance and data persistence
- ⚠️ Less efficient at high message volumes
- ⚠️ Polling creates database load

**Message Flow:**
```
API Request → Database → Scheduler (polling) → Webhook → Status Update
```

**Configuration:**
```bash
# Default configuration (database queue)
RABBITMQ_ENABLED=false  # or omit this variable
```

### RabbitMQ Queue (Optional)

Uses RabbitMQ message broker for high-performance scenarios.

**Characteristics:**
- ✅ High throughput and low latency
- ✅ Push-based message delivery (no polling)
- ✅ Built-in retry logic and dead letter queues
- ✅ Horizontal scaling support
- ⚠️ Additional service dependency

**Message Flow:**
```
API Request → RabbitMQ Queue → Consumer (push) → Webhook → Ack/Nack
```

**Configuration:**
```bash
# Enable RabbitMQ
RABBITMQ_ENABLED=true
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Queue names (optional, with defaults)
RABBITMQ_MESSAGES_QUEUE=messages.pending
RABBITMQ_RETRY_QUEUE=messages.retry
RABBITMQ_DLQ=messages.dead_letter

# Retry configuration (optional)
RABBITMQ_MAX_RETRIES=5
RABBITMQ_RETRY_DELAY_MULTIPLIER=60  # seconds
```

### Queue Performance Comparison

| Feature | Database Queue | RabbitMQ Queue |
|---------|----------------|----------------|
| **Best For** | <100 messages/min | >100 messages/min |
| **Setup Complexity** | Simple | Moderate |
| **Latency** | Higher (polling) | Lower (push-based) |
| **Scaling** | Single instance | Multiple workers |
| **Dependencies** | PostgreSQL only | PostgreSQL + RabbitMQ |
| **Reliability** | Database ACID | Native retry/DLQ |

### Retry and Dead Letter Queue

Both implementations support:

- **Exponential Backoff**: Failed messages are retried with increasing delays
- **Dead Letter Queue**: Messages that exceed max retries are moved to DLQ
- **Failure Tracking**: All failures are logged with reasons

#### Database Implementation
- Retries stored in `messages` table with `next_retry_at` timestamp
- DLQ entries stored in `dead_letter_messages` table
- Scheduler processes both new and retry-ready messages

#### RabbitMQ Implementation
- Uses separate queues: `messages.pending`, `messages.retry`, `messages.dead_letter`
- Headers track retry count and failure reasons
- Native RabbitMQ features for reliability

## Configuration

### Environment Variables

#### Required Configuration
```bash
DATABASE_URL=postgresql://postgres:password@localhost:5432/insider_messages
WEBHOOK_URL=https://webhook.site/your-unique-url
WEBHOOK_AUTH_KEY=your-secret-api-key
```

#### Server Configuration  
```bash
SERVER_PORT=8080
SERVER_READ_TIMEOUT=15s
SERVER_WRITE_TIMEOUT=15s
SERVER_IDLE_TIMEOUT=60s
SERVER_SHUTDOWN_TIMEOUT=30s
```

#### Scheduler Configuration
```bash
SCHEDULER_INTERVAL=2m
SCHEDULER_BATCH_SIZE=2
SCHEDULER_WEBHOOK_TIMEOUT=10s
SCHEDULER_MAX_RETRIES=3
SCHEDULER_RETRY_DELAY=1s
```

#### Database Configuration
```bash
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=10
DATABASE_CONN_MAX_LIFETIME=1h
DATABASE_CONN_MAX_IDLE_TIME=30m
```

#### Redis Configuration (Optional)
```bash
REDIS_URL=redis://localhost:6379/0
REDIS_TTL=1h
REDIS_MAX_RETRIES=3
REDIS_MIN_RETRY_BACKOFF=8ms
REDIS_MAX_RETRY_BACKOFF=512ms
```

#### RabbitMQ Configuration (Optional)
```bash
RABBITMQ_ENABLED=false
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_MESSAGES_QUEUE=messages.pending
RABBITMQ_RETRY_QUEUE=messages.retry
RABBITMQ_DLQ=messages.dead_letter
RABBITMQ_MAX_RETRIES=5
RABBITMQ_RETRY_DELAY_MULTIPLIER=60
```

#### Message Configuration
```bash
MESSAGE_MAX_LENGTH=160
MESSAGE_PHONE_VALIDATION_ENABLED=true
```

#### Audit Configuration
```bash
AUDIT_ENABLED=true
AUDIT_BATCH_SIZE=100
AUDIT_RETENTION_DAYS=30
```

## API Development

### Request/Response Types

#### Create Message
```go
type CreateMessageRequest struct {
    PhoneNumber string `json:"phone_number" validate:"required,phone"`
    Content     string `json:"content" validate:"required,max=160"`
}

type CreateMessageResponse struct {
    ID          string    `json:"id"`
    PhoneNumber string    `json:"phone_number"`
    Content     string    `json:"content"`
    Status      string    `json:"status"`
    CreatedAt   time.Time `json:"created_at"`
}
```

#### Control API
```go
type ControlRequest struct {
    Action string `json:"action" validate:"required,oneof=start stop"`
}

type ControlResponse struct {
    Message string `json:"message"`
    Status  string `json:"status"`
}
```

### Authentication

The API uses header-based authentication:

```go
func AuthMiddleware(apiKey string) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader != apiKey {
            c.JSON(401, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### Validation

Uses go-playground/validator for request validation:

```go
type CreateMessageRequest struct {
    PhoneNumber string `json:"phone_number" validate:"required,phone"`
    Content     string `json:"content" validate:"required,max=160"`
}
```

### API Documentation

Swagger documentation is auto-generated:

```bash
# Generate docs
make swagger

# View docs
open http://localhost:8080/api/docs
```

## Testing

### Test Types

#### Unit Tests
Fast, isolated tests that don't require external dependencies.

```bash
# Run unit tests
make test

# With coverage
make test-coverage

# With race detection
make test-race
```

**Characteristics:**
- Fast execution (< 5 minutes)
- No external dependencies
- Mock external services
- High code coverage expected (>80%)

#### Integration Tests
Tests that verify component interactions with real external dependencies.

```bash
# Run integration tests
make test-integration

# With coverage
make test-integration-coverage
```

**Requirements:**
- PostgreSQL database
- Redis (optional)
- Test environment setup

#### Benchmark Tests
Performance tests to measure execution time and memory usage.

```bash
# Run benchmarks
make test-benchmark

# Or directly
./scripts/test-runner.sh --benchmark
```

### Test Runner

The project includes a comprehensive test runner at `scripts/test-runner.sh`:

```bash
# Basic usage
./scripts/test-runner.sh

# Run all test types
./scripts/test-runner.sh --type all

# Generate coverage reports
./scripts/test-runner.sh --coverage

# Watch mode for development
./scripts/test-runner.sh --watch
```

### Command Line Options

| Option | Description | Example |
|--------|-------------|---------|
| `-t, --type TYPE` | Test type: unit, integration, benchmark, all | `--type integration` |
| `-c, --coverage` | Generate coverage reports | `--coverage` |
| `-r, --race` | Enable race condition detection | `--race` |
| `-v, --verbose` | Verbose output | `--verbose` |
| `-w, --watch` | Watch mode (re-run on file changes) | `--watch` |
| `-p, --package PKG` | Run tests for specific package | `--package ./internal/service` |
| `-f, --filter FILTER` | Run tests matching filter pattern | `--filter TestMessage` |
| `--threshold N` | Coverage threshold percentage | `--threshold 85` |
| `--timeout DURATION` | Test timeout | `--timeout 10m` |

### Make Targets

```bash
# Basic testing
make test                    # Quick unit tests
make test-all               # All test types
make test-coverage          # Unit tests with coverage
make test-integration       # Integration tests
make test-benchmark         # Benchmark tests

# Advanced testing
make test-race              # Race condition detection
make test-watch             # Watch mode
make test-verbose           # Verbose output
make test-ci                # Full CI test suite

# Package-specific testing
make test-package PKG=./internal/service

# Maintenance
make test-clean             # Clean test artifacts
make test-setup             # Setup test environment
```

### Coverage Reports

The test runner generates multiple coverage formats:

```bash
# Generate coverage reports
make test-coverage

# Coverage files generated:
# - coverage.out (go format)
# - coverage.html (HTML report)
# - coverage.xml (XML for CI/CD)
```

### Test Environment Setup

```bash
# Setup everything automatically
make test-setup

# Manual setup
./scripts/test-runner.sh --setup
```

### Writing Tests

#### Unit Test Example
```go
func TestMessageService_CreateMessage(t *testing.T) {
    // Setup
    mockRepo := &repository.MockMessageRepository{}
    service := NewMessageService(mockRepo, nil)
    
    // Test data
    req := CreateMessageRequest{
        PhoneNumber: "+1234567890",
        Content:     "Test message",
    }
    
    // Mock expectations
    mockRepo.On("Create", mock.AnythingOfType("*domain.Message")).
        Return(nil)
    
    // Execute
    resp, err := service.CreateMessage(context.Background(), req)
    
    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, resp.ID)
    assert.Equal(t, req.PhoneNumber, resp.PhoneNumber)
    mockRepo.AssertExpectations(t)
}
```

#### Integration Test Example
```go
func TestMessageRepository_Create_Integration(t *testing.T) {
    // Skip if not integration test
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    repo := postgres.NewMessageRepository(db)
    
    // Test data
    message := &domain.Message{
        PhoneNumber: "+1234567890",
        Content:     "Test message",
        Status:      domain.StatusPending,
    }
    
    // Execute
    err := repo.Create(message)
    
    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, message.ID)
}
```

## Build & Deployment

### Build Commands

```bash
# Build binary
make build

# Build with all platforms
make build-all

# Build Docker image
make docker-build

# Clean build artifacts
make clean
```

### Binary Output

```bash
# Local build
bin/ims-server

# Cross-platform builds
bin/ims-server-linux-amd64
bin/ims-server-darwin-amd64
bin/ims-server-windows-amd64.exe
```

### Release Process

```bash
# Create release
make release VERSION=v1.0.0

# This will:
# 1. Tag the version
# 2. Build all platforms
# 3. Create release notes
# 4. Build Docker images
```

## Docker Development

### Docker Compose Profiles

#### Default Profile (Database Queue)
```bash
# Start with database queue
make docker-up

# Services: postgres, redis, ims-app
```

#### RabbitMQ Profile
```bash
# Start with RabbitMQ queue
make docker-up-rabbitmq

# Services: postgres, redis, rabbitmq, ims-app
```

#### Production Profile
```bash
# Production deployment
make docker-up-prod

# Uses optimized settings and multi-stage builds
```

### Docker Commands

```bash
# Development
make docker-build          # Build image
make docker-up             # Start services
make docker-down           # Stop services
make docker-logs           # View logs
make docker-restart        # Restart services

# RabbitMQ specific
make rabbitmq-start        # Start RabbitMQ only
make rabbitmq-stop         # Stop RabbitMQ only
make rabbitmq-ui           # Open management UI
make rabbitmq-logs         # View RabbitMQ logs
make rabbitmq-status       # Check RabbitMQ status

# Maintenance
make docker-clean          # Clean images and volumes
make docker-prune          # Prune unused resources
```

### Docker Configuration

Environment variables can be set in:
- `.env` file (for docker-compose)
- `docker/.env.template` (template for production)

### Multi-stage Dockerfile

The project uses a multi-stage Dockerfile:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/ims-server .
EXPOSE 8080
CMD ["./ims-server"]
```

### Health Checks

Docker services include health checks:

```yaml
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/api/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

## Troubleshooting

### Common Issues

#### Database Connection Issues
```bash
# Check database connectivity
psql $DATABASE_URL -c "SELECT 1;"

# Check database logs
docker logs postgres

# Check migrations
./scripts/migrate.sh --status
```

#### RabbitMQ Connection Issues
```bash
# Check RabbitMQ status
docker logs rabbitmq

# Test connection
curl -u guest:guest http://localhost:15672/api/overview

# Check queues
curl -u guest:guest http://localhost:15672/api/queues
```

#### Queue Processing Issues
```bash
# Check application logs
docker logs ims-app

# Check scheduler status
curl -H "Authorization: your-api-key" http://localhost:8080/api/health

# Monitor queue sizes (RabbitMQ)
make rabbitmq-ui
```

#### Performance Issues
```bash
# Run benchmarks
make test-benchmark

# Check database indexes
psql $DATABASE_URL -c "\d+ messages"

# Monitor resource usage
docker stats
```

### Debugging

#### Enable Debug Mode
```bash
# Set log level
LOG_LEVEL=debug

# Enable SQL query logging
DATABASE_LOG_QUERIES=true

# Enable detailed HTTP logging
HTTP_LOG_REQUESTS=true
```

#### Profiling
```bash
# Enable pprof endpoint
PPROF_ENABLED=true

# Access profiling
go tool pprof http://localhost:8080/debug/pprof/profile
```

### Monitoring

#### Health Check Endpoint
```bash
curl http://localhost:8080/api/health
```

Response includes:
- Database connectivity
- Cache connectivity (if enabled)
- RabbitMQ connectivity (if enabled)
- Scheduler status

#### Metrics

The application exposes metrics for monitoring:

```bash
# Basic metrics
curl http://localhost:8080/api/metrics

# Detailed audit logs
curl -H "Authorization: your-api-key" \
     "http://localhost:8080/api/audit?limit=100"
```

#### Log Analysis

```bash
# View application logs
docker logs ims-app

# Follow logs in real-time
docker logs -f ims-app

# Search for specific events
docker logs ims-app 2>&1 | grep "ERROR"
```

### Migration Issues

#### Check Migration Status
```bash
# List applied migrations
psql $DATABASE_URL -c "SELECT * FROM migration_history ORDER BY applied_at;"

# Check for pending migrations
./scripts/migrate.sh --dry-run
```

#### Reset Database (Development Only)
```bash
# Drop and recreate database
dropdb insider_messages
createdb insider_messages
./scripts/migrate.sh
```