# Docker Setup for IMS

This directory contains Docker configuration files for running IMS (Insider Message Sender) in containerized environments.

## Files Overview

- `docker-compose.yml` - Development environment setup
- `docker-compose.prod.yml` - Production environment setup  
- `.env.template` - Environment variables template

## Quick Start

### Development Environment

1. **Start all services:**
   ```bash
   make docker-up
   ```

2. **Access the application:**
   - API: http://localhost:8080
   - Swagger docs: http://localhost:8080/api/docs
   - PostgreSQL: localhost:5432
   - Redis: localhost:6379

3. **View logs:**
   ```bash
   make docker-logs
   ```

4. **Stop services:**
   ```bash
   make docker-down
   ```

### Production Environment

1. **Configure environment:**
   ```bash
   cp docker/.env.template .env
   # Edit .env with your production values
   ```

2. **Start production services:**
   ```bash
   make docker-up-prod
   ```

3. **Monitor logs:**
   ```bash
   make docker-logs-prod
   ```

## Services

### PostgreSQL Database
- **Image:** postgres:15-alpine
- **Port:** 5432
- **Database:** insider_messages
- **Default credentials:** postgres/password
- **Volume:** Persistent data storage

### Redis Cache
- **Image:** redis:7-alpine  
- **Port:** 6379
- **Configuration:** Persistent storage with LRU eviction
- **Volume:** Persistent data storage

### IMS Application
- **Build:** Multi-stage Dockerfile
- **Port:** 8080
- **Features:** 
  - Swagger documentation generation
  - Health checks
  - Non-root user execution
  - Security hardening

### Migration Service
- **Purpose:** One-time database migration execution
- **Behavior:** Runs migrations and exits
- **Dependencies:** Waits for database to be healthy

## Environment Configuration

### Required Variables (Production)
```env
# Database
POSTGRES_PASSWORD=your_secure_password

# Webhook
WEBHOOK_URL=https://your-webhook-endpoint.com/webhook
WEBHOOK_AUTH_KEY=your_webhook_auth_key
```

### Optional Variables
```env
# Application
IMS_IMAGE=ims:latest
SERVER_PORT=8080

# Database Pool
DATABASE_MAX_CONNECTIONS=25
DATABASE_MAX_IDLE_CONNECTIONS=5

# Redis
REDIS_CACHE_TTL=168h

# Scheduler
SCHEDULER_INTERVAL=2m
SCHEDULER_BATCH_SIZE=2

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

## Docker Commands

### Building
```bash
# Build image
make docker-build

# Build with custom tag  
make docker-build-tag TAG=v1.0.0
```

### Running
```bash
# Development
make docker-up
make docker-down

# Production
make docker-up-prod
make docker-down-prod
```

### Maintenance
```bash
# View logs
make docker-logs
make docker-logs-prod

# Run migrations
make docker-migrate
make docker-migrate-prod

# Shell access
make docker-shell
make docker-shell-prod

# Service status
make docker-status

# Clean resources
make docker-clean
```

## Health Checks

All services include health checks:

- **PostgreSQL:** `pg_isready` command
- **Redis:** `redis-cli ping` command  
- **IMS App:** HTTP health endpoint `/api/health`

Health checks ensure proper startup order and service availability.

## Security Features

### Application Container
- Non-root user execution (uid: 1000)
- Minimal Alpine Linux base image
- No unnecessary packages or tools
- Read-only filesystem compatibility

### Network Isolation
- Dedicated Docker network
- Services communicate via service names
- No external network access unless required

### Resource Limits (Production)
- CPU and memory limits
- Restart policies
- Health check timeouts

## Troubleshooting

### Common Issues

1. **Port conflicts:**
   ```bash
   # Check if ports are in use
   lsof -i :8080 -i :5432 -i :6379
   ```

2. **Permission issues:**
   ```bash
   # Ensure proper permissions
   chmod +x scripts/*.sh
   ```

3. **Database connection issues:**
   ```bash
   # Check database logs
   docker-compose logs postgres
   ```

4. **Migration failures:**
   ```bash
   # Run migrations manually
   make docker-migrate
   ```

### Logs and Debugging

```bash
# View all service logs
make docker-logs

# View specific service logs
docker-compose logs postgres
docker-compose logs redis
docker-compose logs ims

# Follow logs in real-time
docker-compose logs -f ims
```

### Performance Monitoring

```bash
# View resource usage
docker stats

# Check service health
docker-compose ps
make docker-status
```

## Production Deployment

### Prerequisites
- Docker Engine 20.10+
- Docker Compose 2.0+
- Minimum 2GB RAM
- Minimum 10GB disk space

### Deployment Steps

1. **Prepare environment:**
   ```bash
   # Create production directory
   mkdir -p /opt/ims
   cd /opt/ims
   
   # Copy configuration
   cp docker/.env.template .env
   # Edit .env with production values
   ```

2. **Deploy services:**
   ```bash
   # Start production services
   make docker-up-prod
   
   # Verify deployment
   make docker-status
   ```

3. **Configure monitoring:**
   ```bash
   # Setup log rotation
   # Configure monitoring tools
   # Setup backup procedures
   ```

### Backup and Recovery

```bash
# Database backup
docker-compose exec postgres pg_dump -U postgres insider_messages > backup.sql

# Database restore  
docker-compose exec -T postgres psql -U postgres insider_messages < backup.sql

# Volume backup
docker run --rm -v ims-postgres-data:/data -v $(pwd):/backup alpine tar czf /backup/postgres-backup.tar.gz /data
```

## Development Tips

### Local Development
- Use development compose file for easier debugging
- Mount source code for hot reloading (if needed)
- Use localhost URLs for external services

### Testing
- Use separate database for tests
- Run tests in isolated containers
- Validate migrations before deployment

### Debugging
- Use `make docker-shell` for container access
- Check environment variables with `env` command
- Monitor resource usage with `docker stats` 