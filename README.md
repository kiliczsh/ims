# IMS - Insider Message Sender

A reliable Go-based service that automatically sends messages via webhooks with comprehensive audit logging and monitoring.

## What is IMS?

IMS queues messages in a database and processes them in batches at configurable intervals, sending them to webhook endpoints. Perfect for automated messaging, notifications, and integrations.

## Key Features

- 📨 **Automatic Message Scheduling** - Process messages at configurable intervals
- 🔄 **Webhook Integration** - Send messages to any HTTP endpoint
- 📊 **Complete Audit Trail** - Track every message and system event
- 🏥 **Health Monitoring** - Monitor system components and status
- 📚 **Interactive API Documentation** - Built-in Swagger UI for testing
- 🚀 **Simple Deployment** - Single binary with Docker support
- 🐰 **High-Performance Queue** - Optional RabbitMQ for scale (>100 msg/min)
- 🔄 **Automatic Retries** - Smart retry logic with exponential backoff
- 💀 **Dead Letter Queue** - Handle failed messages gracefully

## Quick Start

### Prerequisites
- **Docker** (recommended) or **Go 1.21+**
- **PostgreSQL database** (included in Docker setup)
- **Webhook endpoint** (get a free one at [webhook.site](https://webhook.site))

### 1. Get the Code
```bash
git clone <repository-url>
cd ims
```

### 2. Quick Setup with Docker
```bash
# Start all services (PostgreSQL + Redis + IMS)
make docker-up

# The service starts automatically on http://localhost:8080
```

### 3. Configure Your Webhook
```bash
# Edit configuration
nano .env

# Set your webhook URL (required)
WEBHOOK_URL=https://webhook.site/your-unique-url
WEBHOOK_AUTH_KEY=your-secret-api-key
```

### 4. Start Processing Messages
Visit `http://localhost:8080/api/docs` for interactive API documentation.

```bash
# Start the message scheduler
curl -X POST http://localhost:8080/api/control \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{"action": "start"}'
```

## High-Performance Option: RabbitMQ

For high-volume scenarios (>100 messages/minute), enable RabbitMQ:

```bash
# Start with RabbitMQ queue
make docker-up-rabbitmq

# Monitor RabbitMQ
make rabbitmq-ui  # Opens http://localhost:15672
```

## How It Works

1. **Add Messages** - Insert messages into the database (status: 'pending')
2. **Start Scheduler** - Use the control API to begin processing
3. **Batch Processing** - Messages are processed in configurable batches
4. **Webhook Delivery** - Each message is sent to your webhook endpoint
5. **Status Tracking** - Monitor progress through APIs and logs
6. **Retry Logic** - Failed messages are retried with exponential backoff
7. **Dead Letter Queue** - Permanently failed messages are stored for review

## Using the Service

### Check Service Health
```bash
curl http://localhost:8080/api/health
```

### Create a Message
```bash
curl -X POST http://localhost:8080/api/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "phone_number": "+1234567890",
    "content": "Hello from IMS!"
  }'
```

### View Sent Messages
```bash
curl "http://localhost:8080/api/messages/sent" \
  -H "Authorization: your-api-key"
```

### View Dead Letter Queue
```bash
curl "http://localhost:8080/api/messages/dead-letter" \
  -H "Authorization: your-api-key"
```

### Control the Scheduler
```bash
# Start processing
curl -X POST http://localhost:8080/api/control \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{"action": "start"}'

# Stop processing
curl -X POST http://localhost:8080/api/control \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{"action": "stop"}'
```

## API Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/health` | GET | No | Service health check |
| `/api/docs` | GET | No | Interactive API documentation |
| `/api/messages` | POST | Yes | Create a new message |
| `/api/messages/sent` | GET | Yes | List sent messages |
| `/api/messages/dead-letter` | GET | Yes | List failed messages |
| `/api/control` | POST | Yes | Start/stop scheduler |
| `/api/audit` | GET | Yes | View audit logs |

## Configuration

Key environment variables:

| Setting | Default | Description |
|---------|---------|-------------|
| `DATABASE_URL` | - | PostgreSQL connection string (required) |
| `WEBHOOK_URL` | - | Target webhook endpoint (required) |
| `WEBHOOK_AUTH_KEY` | - | API key for authentication (required) |
| `SERVER_PORT` | 8080 | HTTP server port |
| `SCHEDULER_INTERVAL` | 2m | Message processing interval |
| `SCHEDULER_BATCH_SIZE` | 2 | Messages per batch |
| `RABBITMQ_ENABLED` | false | Enable RabbitMQ queue |

## Queue Options Comparison

| Feature | Database Queue (Default) | RabbitMQ Queue |
|---------|-------------------------|----------------|
| **Best For** | <100 messages/min | >100 messages/min |
| **Setup** | Simple, no extra services | Requires RabbitMQ |
| **Latency** | Higher (polling-based) | Lower (push-based) |
| **Scaling** | Single instance | Multiple workers |
| **Dependencies** | PostgreSQL only | PostgreSQL + RabbitMQ |

## Development

For technical details, architecture, setup instructions, and development workflows, see [DEVELOPMENT.md](DEVELOPMENT.md).

## License

[MIT License](LICENSE)