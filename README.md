# IMS

A Go-based message scheduling and sending service with comprehensive audit logging and dynamic API documentation.

## What is IMS?

IMS (Insider Message Sender) is a reliable service that automatically sends messages via webhooks. It queues messages in a database and processes them in batches at configurable intervals.

## Key Features

- 📨 **Automatic Message Scheduling** - Set it and forget it message processing
- 🔄 **Webhook Integration** - Send messages to any webhook endpoint
- 📊 **Complete Audit Trail** - Track every message and system event
- 🏥 **Health Monitoring** - Monitor database, cache, and scheduler status
- 📚 **Interactive API Documentation** - Built-in Swagger UI for easy testing
- 🚀 **Simple Deployment** - Single binary with Docker support

## Quick Start

### Prerequisites
- **Go 1.21+** or **Docker**
- **PostgreSQL database** (local or remote)
- **Webhook endpoint** (get a free one at [webhook.site](https://webhook.site))

### 1. Get the Code
```bash
git clone <repository-url>
cd ims
```

### 2. Configure
```bash
# Copy configuration template
cp .env.example .env

# Edit with your settings
nano .env
```

Required settings:
- `DATABASE_URL` - Your PostgreSQL connection string
- `WEBHOOK_URL` - Where to send messages (webhook.site URL works great for testing)
- `WEBHOOK_AUTH_KEY` - API key for accessing the service

### 3. Run

#### Option A: Docker (Recommended)
```bash
# Start all services (PostgreSQL + Redis + IMS)
make docker-up

# The service will start automatically with database setup
```

#### Option B: Local Development
```bash
# Setup database and start
make setup-dev
make migrate
make run
```

The service will start on `http://localhost:8080`

## Using the Service

### View API Documentation
Open `http://localhost:8080/api/docs` in your browser for interactive API documentation.

### Check Health
```bash
curl http://localhost:8080/api/health
```

### Start Message Processing
```bash
curl -X POST http://localhost:8080/api/control \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{"action": "start"}'
```

### View Sent Messages
```bash
curl "http://localhost:8080/api/messages/sent" \
  -H "Authorization: your-api-key"
```

## How It Works

1. **Add Messages** - Insert messages into the database with status 'pending'
2. **Start Scheduler** - Use the control API to start automatic processing
3. **Batch Processing** - The service processes messages in configurable batches
4. **Webhook Delivery** - Messages are sent to your webhook endpoint
5. **Status Tracking** - Monitor progress through audit logs and API endpoints

## Configuration Options

| Setting | Default | Description |
|---------|---------|-------------|
| `SERVER_PORT` | 8080 | HTTP server port |
| `SCHEDULER_INTERVAL` | 2m | How often to process messages |
| `SCHEDULER_BATCH_SIZE` | 2 | Messages per batch |
| `MESSAGE_MAX_LENGTH` | 160 | Maximum message content length |

## API Endpoints

- **Health Check**: `GET /api/health` (public)
- **Control Scheduler**: `POST /api/control` (requires auth)
- **View Messages**: `GET /api/messages/sent` (requires auth)
- **Audit Logs**: `GET /api/audit` (requires auth)
- **API Documentation**: `GET /api/docs` (public)

## Testing

IMS includes a comprehensive testing framework with unit tests, integration tests, and benchmarks.

### Quick Testing
```bash
# Run basic unit tests
make test

# Run all tests with coverage
make test-all-coverage

# Watch tests during development
make test-watch

# Run tests for CI/CD
make test-ci
```

### Testing Options
- **Unit Tests**: Fast, isolated tests (>80% coverage required)
- **Integration Tests**: Real database testing
- **Benchmark Tests**: Performance measurements
- **Race Detection**: Concurrent safety testing
- **Coverage Reports**: HTML and XML formats

See [docs/TESTING.md](docs/TESTING.md) for comprehensive testing documentation.

## Development

For technical details, architecture information, and development setup, see [DEVELOPMENT.md](DEVELOPMENT.md).

Quick development commands:
```bash
make help          # Show all available commands
make dev           # Run with hot reload
make test          # Run tests
make swagger       # Generate API docs
```

## Docker Support

### Quick Docker Start
```bash
# Start all services (includes PostgreSQL and Redis)
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

### Docker Commands
```bash
# Build image
make docker-build

# Production deployment
cp docker/.env.template .env
# Edit .env with production values
make docker-up-prod

# Manual Docker run
docker build -t ims .
docker run -p 8080:8080 --env-file .env ims
```

See [docker/README.md](docker/README.md) for detailed Docker documentation.

## Support

- **API Documentation**: `http://localhost:8080/api/docs`
- **Health Monitoring**: `http://localhost:8080/api/health`
- **Technical Details**: [DEVELOPMENT.md](DEVELOPMENT.md)

## License

This project is licensed under the MIT License.
