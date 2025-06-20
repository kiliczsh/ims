.PHONY: build run run-script run-script-setup test test-all test-coverage test-integration test-integration-coverage test-all-coverage test-benchmark test-race test-package test-watch test-verbose test-clean test-setup test-ci test-quick clean deps install-tools lint fmt help migrate migrate-with-data swagger swagger-gen swagger-install swagger-serve docker-build docker-build-tag docker-up docker-up-prod docker-down docker-down-prod docker-logs docker-logs-prod docker-restart docker-migrate docker-migrate-prod docker-status docker-shell docker-shell-prod docker-clean release release-minor release-major release-version release-docker version rabbitmq-start rabbitmq-stop rabbitmq-logs rabbitmq-ui rabbitmq-status

# Build the application (legacy target - use build with bin dependency)

# Run the application
run:
	@if [ -f ".env" ]; then \
		echo "Loading environment from .env..."; \
		echo ""; \
		echo "🌟 Starting Insider Message Sender..."; \
		echo ""; \
		set -a && source .env && set +a && \
		echo "📍 Server will start on: http://localhost:$$SERVER_PORT"; \
		echo "🏥 Health check: http://localhost:$$SERVER_PORT/api/health"; \
		echo "📚 API docs: http://localhost:$$SERVER_PORT/api/docs"; \
		echo "🔑 API Key: $$WEBHOOK_AUTH_KEY"; \
		echo ""; \
		echo "💡 To test the API:"; \
		echo "   curl http://localhost:$$SERVER_PORT/api/health"; \
		echo ""; \
		echo "💡 To start the scheduler:"; \
		echo "   curl -X POST http://localhost:$$SERVER_PORT/api/control \\"; \
		echo "     -H 'Content-Type: application/json' \\"; \
		echo "     -H 'x-ins-auth-key: $$WEBHOOK_AUTH_KEY' \\"; \
		echo "     -d '{\"action\": \"start\"}'"; \
		echo ""; \
		echo "⏹️  Press Ctrl+C to stop the server"; \
		echo ""; \
		go run cmd/server/main.go; \
	else \
		echo "⚠️  .env file not found. Creating from template..."; \
		make env-template; \
		echo "📝 Please edit .env with your configuration and run 'make run' again"; \
		exit 1; \
	fi

# Run using the legacy script (alternative method)
run-script:
	@echo "🚀 Running with legacy script..."
	@./scripts/run.sh

# Run with database setup using the legacy script
run-script-setup:
	@echo "🚀 Running with legacy script and database setup..."
	@./scripts/run.sh --setup-db

# Download dependencies
deps:
	go mod download
	go mod tidy

# Install development tools
install-tools:
	@echo "🛠️  Installing development tools..."
	@echo "📥 Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "📥 Installing swag (Swagger CLI)..."
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "📥 Installing air (hot reload)..."
	go install github.com/cosmtrek/air@latest
	@echo "✅ All development tools installed successfully"

# Generate Swagger documentation
swagger-gen:
	@echo "📚 Generating Swagger documentation..."
	@if command -v swag >/dev/null 2>&1; then \
		swag init \
			--generalInfo cmd/server/main.go \
			--dir ./ \
			--output docs \
			--outputTypes go,json,yaml \
			--parseInternal \
			--quiet; \
		echo "✅ Swagger documentation generated in docs/"; \
		echo "📖 View docs at: http://localhost:8080/api/docs (when server is running)"; \
	else \
		echo "❌ swag is not installed. Installing..."; \
		$(MAKE) swagger-install; \
		$(MAKE) swagger-gen; \
	fi

# Install Swagger CLI
swagger-install:
	@echo "📥 Installing swag CLI..."
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "✅ swag CLI installed"

# Generate swagger docs (alias)
swagger: swagger-gen

# Serve swagger docs standalone (for development)
swagger-serve:
	@echo "🌐 Serving standalone Swagger UI..."
	@if [ ! -f "docs/swagger.yaml" ]; then \
		echo "❌ docs/swagger.yaml not found. Generating..."; \
		$(MAKE) swagger-gen; \
	fi
	@echo "📖 Open http://localhost:9090 in your browser"
	@echo "⏹️  Press Ctrl+C to stop"
	@if command -v python3 >/dev/null 2>&1; then \
		cd docs && python3 -m http.server 9090; \
	elif command -v python >/dev/null 2>&1; then \
		cd docs && python -m SimpleHTTPServer 9090; \
	else \
		echo "❌ Python not found. Cannot serve standalone docs."; \
		echo "💡 Install the server and run 'make run' to view docs at /api/docs"; \
	fi

# Create docs directory
docs:
	mkdir -p docs

# =============================================================================
# Testing Commands
# =============================================================================

# Run unit tests (quick)
test:
	@echo "🧪 Running unit tests..."
	@./scripts/test-runner.sh --type unit

# Run all tests
test-all:
	@echo "🧪 Running all tests..."
	@./scripts/test-runner.sh --type all

# Run unit tests with coverage
test-coverage:
	@echo "🧪 Running unit tests with coverage..."
	@./scripts/test-runner.sh --type unit --coverage

# Run integration tests
test-integration:
	@echo "🧪 Running integration tests..."
	@./scripts/test-runner.sh --type integration --setup

# Run integration tests with coverage
test-integration-coverage:
	@echo "🧪 Running integration tests with coverage..."
	@./scripts/test-runner.sh --type integration --coverage --setup

# Run all tests with coverage
test-all-coverage:
	@echo "🧪 Running all tests with coverage..."
	@./scripts/test-runner.sh --type all --coverage --setup

# Run benchmark tests
test-benchmark:
	@echo "🧪 Running benchmark tests..."
	@./scripts/test-runner.sh --benchmark

# Run tests with race detection
test-race:
	@echo "🧪 Running tests with race detection..."
	@./scripts/test-runner.sh --type unit --race

# Run tests for specific package
test-package:
	@if [ -z "$(PKG)" ]; then \
		echo "❌ Please provide a package. Usage: make test-package PKG=./internal/service"; \
		exit 1; \
	fi
	@echo "🧪 Running tests for package: $(PKG)"
	@./scripts/test-runner.sh --package $(PKG) --coverage

# Watch tests (re-run on file changes)
test-watch:
	@echo "🧪 Starting test watch mode..."
	@./scripts/test-runner.sh --watch --coverage

# Run tests with verbose output
test-verbose:
	@echo "🧪 Running tests with verbose output..."
	@./scripts/test-runner.sh --type unit --verbose --coverage

# Clean test artifacts
test-clean:
	@echo "🧪 Cleaning test artifacts..."
	@./scripts/test-runner.sh --clean

# Setup test environment
test-setup:
	@echo "🧪 Setting up test environment..."
	@./scripts/test-runner.sh --setup

# Full test suite for CI/CD
test-ci:
	@echo "🧪 Running full CI test suite..."
	@./scripts/test-runner.sh --type all --coverage --race --json --xml --setup --threshold 80

# Quick test for development
test-quick:
	@echo "🧪 Running quick tests..."
	@./scripts/test-runner.sh --type unit --failfast

# Format code
fmt:
	@echo "🎨 Formatting Go code..."
	go fmt ./...

# Lint code (auto-installs golangci-lint if missing)
lint:
	@echo "🔍 Running Go linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "📥 golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		echo "✅ golangci-lint installed successfully"; \
		golangci-lint run; \
	fi

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf docs/
	rm -rf dist/
	rm -rf test-results/
	rm -f coverage.out coverage.html coverage.xml
	rm -f test.json benchmark.txt profile.out
	rm -f *.tar.gz *.zip *.sha256
	go clean -testcache

# Run database migrations
migrate:
	./scripts/migrate.sh

# Run database migrations with sample data
migrate-with-data:
	./scripts/migrate.sh --with-sample-data

# Setup database (legacy - use migrate instead)
db-setup:
	@echo "⚠️  db-setup is deprecated. Use 'make migrate' instead."
	./scripts/migrate.sh

# Create bin directory
bin:
	mkdir -p bin

# Build with proper directory creation and swagger generation
build: bin docs swagger-gen
	@version=$$(if [ -f VERSION ]; then cat VERSION; else echo "dev"; fi); \
	buildTime=$$(date -u +%Y-%m-%dT%H:%M:%SZ); \
	gitCommit=$$(if git rev-parse --git-dir > /dev/null 2>&1; then git rev-parse --short HEAD; else echo "unknown"; fi); \
	echo "Building IMS v$$version (commit: $$gitCommit, built: $$buildTime)"; \
	go build \
		-ldflags "-X main.version=$$version -X main.buildTime=$$buildTime -X main.gitCommit=$$gitCommit" \
		-o bin/ims \
		cmd/server/main.go

# Development server with hot reload (if air is installed)
dev:
	@if [ ! -f ".env" ]; then \
		echo "⚠️  .env file not found. Creating from template..."; \
		make env-template; \
		echo "📝 Please edit .env with your configuration and run 'make dev' again"; \
		exit 1; \
	fi
	@if command -v air > /dev/null; then \
		echo "Loading environment from .env and starting with hot reload..."; \
		set -a && source .env && set +a && air; \
	else \
		echo "Air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to regular run..."; \
		$(MAKE) run; \
	fi

# Check if all required tools are installed
check-tools:
	@echo "Checking required tools..."
	@command -v go >/dev/null 2>&1 || { echo "❌ Go is not installed"; exit 1; }
	@command -v psql >/dev/null 2>&1 || { echo "❌ PostgreSQL client (psql) is not installed"; exit 1; }
	@echo "✅ All required tools are installed"

# Setup development environment
setup-dev: check-tools deps setup-env
	@echo "✅ Development environment setup complete"

# Setup .env file
setup-env:
	@echo "Setting up environment configuration..."
	@./scripts/setup-dev.sh

# Create .env from template
env-template:
	@if [ ! -f ".env" ]; then \
		if [ -f ".env.example" ]; then \
			cp .env.example .env; \
			echo "✅ Created .env from .env.example template"; \
			echo "📝 Please edit .env with your actual configuration"; \
		elif [ -f "scripts/.env.example" ]; then \
			cp scripts/.env.example .env; \
			echo "✅ Created .env from scripts/.env.example template"; \
			echo "📝 Please edit .env with your actual configuration"; \
		else \
			echo "Creating basic .env template..."; \
			echo "# Insider Message Sender Configuration" > .env; \
			echo "# Edit with your actual values" >> .env; \
			echo "" >> .env; \
			echo "# Required Configuration" >> .env; \
			echo "DATABASE_URL=postgresql://postgres:password@localhost:5432/insider_messages" >> .env; \
			echo "WEBHOOK_URL=https://webhook.site/your-unique-url" >> .env; \
			echo "WEBHOOK_AUTH_KEY=INS.me1x9uMcyYGlhKKQVPoc.bO3j9aZwRTOcA2Ywo" >> .env; \
			echo "" >> .env; \
			echo "# Optional Configuration" >> .env; \
			echo "SERVER_PORT=8080" >> .env; \
			echo "REDIS_URL=redis://localhost:6379/0" >> .env; \
			echo "SCHEDULER_INTERVAL=2m" >> .env; \
			echo "SCHEDULER_BATCH_SIZE=2" >> .env; \
			echo "MESSAGE_MAX_LENGTH=160" >> .env; \
			echo "✅ Created basic .env template"; \
			echo "📝 Please edit .env with your actual configuration"; \
		fi \
	else \
		echo "ℹ️  .env file already exists"; \
	fi

# Validate .env configuration
validate-env:
	@echo "Validating environment configuration..."
	@if [ -f "scripts/common.sh" ]; then \
		bash -c "source scripts/common.sh && load_env && validate_config && echo '✅ Environment configuration is valid'"; \
	else \
		echo "❌ scripts/common.sh not found"; \
		exit 1; \
	fi

# Show current environment summary
show-env:
	@echo "Current environment configuration:"
	@if [ -f "scripts/common.sh" ] && [ -f ".env" ]; then \
		bash -c "source scripts/common.sh && load_env && show_env_summary"; \
	else \
		echo "❌ Missing scripts/common.sh or .env file"; \
	fi

# =============================================================================
# Docker Commands
# =============================================================================

# Build Docker image
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t ims:latest .
	@echo "✅ Docker image built successfully"

# Build Docker image with custom tag
docker-build-tag:
	@if [ -z "$(TAG)" ]; then \
		echo "❌ Please provide a TAG. Usage: make docker-build-tag TAG=v1.0.0"; \
		exit 1; \
	fi
	@echo "🐳 Building Docker image with tag: $(TAG)"
	docker build -t ims:$(TAG) -t ims:latest .
	@echo "✅ Docker image built with tag: $(TAG)"

# Run with Docker Compose (development)
docker-up:
	@echo "🐳 Starting IMS with Docker Compose..."
	docker-compose up -d
	@echo "✅ IMS services started"
	@echo "📖 API: http://localhost:8080"
	@echo "📖 Docs: http://localhost:8080/api/docs"
	@echo "📊 PostgreSQL: localhost:5432"
	@echo "📊 Redis: localhost:6379"

# Run with Docker Compose (production)
docker-up-prod:
	@if [ ! -f ".env" ]; then \
		echo "❌ .env file required for production. Copy docker/.env.template to .env and configure"; \
		exit 1; \
	fi
	@echo "🐳 Starting IMS Production with Docker Compose..."
	docker-compose -f docker-compose.prod.yml up -d
	@echo "✅ IMS production services started"

# Stop Docker Compose services
docker-down:
	@echo "🐳 Stopping IMS services..."
	docker-compose down
	@echo "✅ IMS services stopped"

# Stop production Docker Compose services
docker-down-prod:
	@echo "🐳 Stopping IMS production services..."
	docker-compose -f docker-compose.prod.yml down
	@echo "✅ IMS production services stopped"

# View Docker Compose logs
docker-logs:
	@echo "🐳 Viewing IMS logs..."
	docker-compose logs -f

# View production Docker Compose logs
docker-logs-prod:
	@echo "🐳 Viewing IMS production logs..."
	docker-compose -f docker-compose.prod.yml logs -f

# Restart Docker Compose services
docker-restart:
	@echo "🐳 Restarting IMS services..."
	docker-compose restart
	@echo "✅ IMS services restarted"

# Clean Docker resources
docker-clean:
	@echo "🐳 Cleaning Docker resources..."
	docker-compose down --volumes --remove-orphans
	docker-compose -f docker-compose.prod.yml down --volumes --remove-orphans 2>/dev/null || true
	docker system prune -f
	@echo "✅ Docker resources cleaned"

# Run database migrations in Docker
docker-migrate:
	@echo "🐳 Running database migrations in Docker..."
	docker-compose run --rm migration
	@echo "✅ Database migrations completed"

# Run database migrations in production Docker
docker-migrate-prod:
	@echo "🐳 Running database migrations in production Docker..."
	docker-compose -f docker-compose.prod.yml run --rm migration
	@echo "✅ Database migrations completed"

# Check Docker services status
docker-status:
	@echo "🐳 Docker services status:"
	@echo ""
	@echo "Development services:"
	@docker-compose ps 2>/dev/null || echo "  No development services running"
	@echo ""
	@echo "Production services:"
	@docker-compose -f docker-compose.prod.yml ps 2>/dev/null || echo "  No production services running"

# Shell into running container
docker-shell:
	@echo "🐳 Opening shell in IMS container..."
	docker-compose exec ims sh

# Shell into production container
docker-shell-prod:
	@echo "🐳 Opening shell in IMS production container..."
	docker-compose -f docker-compose.prod.yml exec ims sh

# Run with Docker Compose + RabbitMQ (high-performance)
docker-up-rabbitmq:
	@echo "🐳 Starting IMS with RabbitMQ for high-performance messaging..."
	docker-compose -f docker-compose.yml -f docker-compose.rabbitmq.yml up -d
	@echo "✅ IMS services started with RabbitMQ"
	@echo "📖 API: http://localhost:8080"
	@echo "📖 Docs: http://localhost:8080/api/docs"
	@echo "📊 PostgreSQL: localhost:5432"
	@echo "📊 Redis: localhost:6379"
	@echo "🐰 RabbitMQ Management: http://localhost:15672 (guest/guest)"
	@echo "🐰 RabbitMQ AMQP: localhost:5672"

# Start only RabbitMQ service
rabbitmq-start:
	@echo "🐰 Starting RabbitMQ service..."
	docker-compose --profile rabbitmq up rabbitmq -d
	@echo "✅ RabbitMQ started"
	@echo "🐰 Management UI: http://localhost:15672 (guest/guest)"
	@echo "🐰 AMQP Port: localhost:5672"

# Stop RabbitMQ service
rabbitmq-stop:
	@echo "🐰 Stopping RabbitMQ service..."
	docker-compose stop rabbitmq
	@echo "✅ RabbitMQ stopped"

# View RabbitMQ logs
rabbitmq-logs:
	@echo "🐰 Viewing RabbitMQ logs..."
	docker-compose logs -f rabbitmq

# Open RabbitMQ Management UI
rabbitmq-ui:
	@echo "🐰 Opening RabbitMQ Management UI..."
	@echo "🌐 http://localhost:15672"
	@echo "👤 Username: guest"
	@echo "🔑 Password: guest"
	@if command -v open >/dev/null 2>&1; then \
		open http://localhost:15672; \
	elif command -v xdg-open >/dev/null 2>&1; then \
		xdg-open http://localhost:15672; \
	else \
		echo "💡 Please open http://localhost:15672 in your browser"; \
	fi

# Show RabbitMQ status and queue information
rabbitmq-status:
	@echo "🐰 RabbitMQ Status:"
	@echo ""
	@if docker-compose ps rabbitmq | grep -q "Up"; then \
		echo "✅ RabbitMQ is running"; \
		echo ""; \
		echo "📊 Queue Overview:"; \
		curl -s -u guest:guest http://localhost:15672/api/overview 2>/dev/null | \
		grep -o '"queue_totals":{[^}]*}' | \
		sed 's/[{}"]//g' | \
		sed 's/queue_totals://g' | \
		sed 's/,/\n/g' | \
		sed 's/^/  /' || echo "  Unable to fetch queue statistics"; \
		echo ""; \
		echo "🔗 Management UI: http://localhost:15672"; \
	else \
		echo "❌ RabbitMQ is not running"; \
		echo "💡 Start with: make rabbitmq-start"; \
	fi

# =============================================================================
# Release Management
# =============================================================================

# Create a new release (patch version)
release:
	@./scripts/release.sh patch

# Create a new minor release
release-minor:
	@./scripts/release.sh minor

# Create a new major release
release-major:
	@./scripts/release.sh major

# Create a release with specific version
release-version:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Please provide a VERSION. Usage: make release-version VERSION=1.2.3"; \
		exit 1; \
	fi
	@./scripts/release.sh $(VERSION)

# Build Docker image for release (current version)
release-docker:
	@if [ ! -f "VERSION" ]; then \
		echo "❌ VERSION file not found. Run 'make release' first."; \
		exit 1; \
	fi
	@version=$$(cat VERSION); \
	echo "🐳 Building Docker image for version $$version..."; \
	docker build -t ims:v$$version -t ims:latest .

# Show current version
version:
	@if [ -f "VERSION" ]; then \
		echo "Current version: $$(cat VERSION)"; \
	else \
		echo "No VERSION file found. Run 'make release' to create first release."; \
	fi

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "🏗️  Build & Run:"
	@echo "  build            - Build the application with swagger docs"
	@echo "  run              - Run the application"
	@echo "  run-script       - Run using legacy script (alternative method)"
	@echo "  run-script-setup - Run with database setup using legacy script"
	@echo "  dev              - Run with hot reload (requires air)"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-build-tag - Build Docker image with custom tag (TAG=version)"
	@echo "  docker-up        - Start services with Docker Compose (development)"
	@echo "  docker-up-rabbitmq - Start services with RabbitMQ (high-performance)"
	@echo "  docker-up-prod   - Start services with Docker Compose (production)"
	@echo "  docker-down      - Stop Docker Compose services"
	@echo "  docker-down-prod - Stop production Docker Compose services"
	@echo "  docker-logs      - View Docker Compose logs"
	@echo "  docker-logs-prod - View production Docker Compose logs"
	@echo "  docker-restart   - Restart Docker Compose services"
	@echo "  docker-migrate   - Run database migrations in Docker"
	@echo "  docker-migrate-prod - Run migrations in production Docker"
	@echo "  docker-status    - Check Docker services status"
	@echo "  docker-shell     - Shell into running container"
	@echo "  docker-shell-prod- Shell into production container"
	@echo "  docker-clean     - Clean Docker resources"
	@echo ""
	@echo "🐰 RabbitMQ:"
	@echo "  rabbitmq-start   - Start only RabbitMQ service"
	@echo "  rabbitmq-stop    - Stop RabbitMQ service"
	@echo "  rabbitmq-logs    - View RabbitMQ logs"
	@echo "  rabbitmq-ui      - Open RabbitMQ Management UI"
	@echo "  rabbitmq-status  - Show RabbitMQ status and queue info"
	@echo ""
	@echo "📦 Release Management:"
	@echo "  release          - Create new patch release (1.0.0 -> 1.0.1)"
	@echo "  release-minor    - Create new minor release (1.0.1 -> 1.1.0)"
	@echo "  release-major    - Create new major release (1.1.0 -> 2.0.0)"
	@echo "  release-version  - Create release with specific version (VERSION=1.2.3)"
	@echo "  release-docker   - Build Docker image for current version"
	@echo "  version          - Show current version"
	@echo ""
	@echo "📚 Documentation:"
	@echo "  swagger          - Generate Swagger/OpenAPI documentation"
	@echo "  swagger-gen      - Generate Swagger documentation"
	@echo "  swagger-install  - Install swag CLI tool"
	@echo "  swagger-serve    - Serve docs standalone on :9090"
	@echo ""
	@echo "🗄️  Database:"
	@echo "  migrate          - Run database migrations"
	@echo "  migrate-with-data- Run migrations with sample data"
	@echo ""
	@echo "⚙️  Environment:"
	@echo "  setup-dev        - Setup complete development environment"
	@echo "  setup-env        - Setup .env configuration file"
	@echo "  env-template     - Create .env from .env.example"
	@echo "  validate-env     - Validate .env configuration"
	@echo "  show-env         - Show current environment summary"
	@echo ""
	@echo "🧪 Testing & Quality:"
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  fmt              - Format code"
	@echo "  install-tools    - Install development tools (golangci-lint, swag, air)"
	@echo "  lint             - Lint code (auto-installs golangci-lint if missing)"
	@echo ""
	@echo "🔧 Utilities:"
	@echo "  deps             - Download and tidy dependencies"
	@echo "  clean            - Clean build artifacts and docs"
	@echo "  check-tools      - Check if required tools are installed"
	@echo "  help             - Show this help" 