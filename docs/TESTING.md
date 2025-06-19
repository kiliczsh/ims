# IMS Testing Guide

This document provides comprehensive information about testing in the IMS (Insider Message Sender) project, including the test runner, testing strategies, and best practices.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Test Runner](#test-runner)
3. [Test Types](#test-types)
4. [Test Environment Setup](#test-environment-setup)
5. [Coverage Reports](#coverage-reports)
6. [CI/CD Integration](#cicd-integration)
7. [Writing Tests](#writing-tests)
8. [Best Practices](#best-practices)
9. [Troubleshooting](#troubleshooting)

## Quick Start

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

## Test Runner

The IMS project includes a comprehensive test runner script at `scripts/test-runner.sh` that provides advanced testing capabilities.

### Basic Usage

```bash
# Basic unit tests
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
| `-b, --benchmark` | Run benchmark tests | `--benchmark` |
| `-j, --json` | Output results in JSON format | `--json` |
| `-x, --xml` | Generate XML coverage report | `--xml` |
| `--threshold N` | Coverage threshold percentage | `--threshold 85` |
| `--timeout DURATION` | Test timeout | `--timeout 10m` |
| `--count N` | Run each test N times | `--count 3` |
| `--parallel N` | Number of parallel processes | `--parallel 4` |
| `--clean` | Clean test artifacts before running | `--clean` |
| `--setup` | Setup test environment | `--setup` |
| `--no-cache` | Disable test cache | `--no-cache` |
| `--failfast` | Stop on first test failure | `--failfast` |

### Make Targets

The project provides convenient Make targets for common testing scenarios:

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

## Test Types

### Unit Tests

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

### Integration Tests

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

**Characteristics:**
- Slower execution (< 10 minutes)
- Real database connections
- End-to-end workflows
- Moderate coverage expected (>70%)

### Benchmark Tests

Performance tests to measure execution time and memory usage.

```bash
# Run benchmarks
make test-benchmark

# Or directly
./scripts/test-runner.sh --benchmark
```

**Output includes:**
- Execution time per operation
- Memory allocations
- Performance comparisons

### End-to-End Tests

Full system tests (future implementation).

```bash
# When implemented
./scripts/test-runner.sh --type e2e
```

## Test Environment Setup

### Prerequisites

1. **Go 1.21+**
2. **PostgreSQL 12+** for integration tests
3. **Redis 6+** (optional)
4. **Test tools** (automatically installed)

### Automatic Setup

```bash
# Setup everything automatically
make test-setup

# Or manually
./scripts/test-runner.sh --setup
```

### Manual Setup

1. **Install test tools:**
   ```bash
   go install github.com/axw/gocov/gocov@latest
   go install github.com/AlekSi/gocov-xml@latest
   go install github.com/cespare/reflex@latest
   ```

2. **Database setup:**
   ```bash
   # Create test database
   createdb ims_test
   
   # Set environment variable
   export DATABASE_URL="postgresql://postgres:password@localhost:5432/ims_test"
   ```

3. **Configuration:**
   ```bash
   # Copy test environment
   cp .env.example .env.test
   
   # Edit with test values
   DATABASE_URL=postgresql://postgres:test@localhost:5432/ims_test
   REDIS_URL=redis://localhost:6379/1
   WEBHOOK_URL=http://localhost:9999/webhook
   ```

## Coverage Reports

### Generate Reports

```bash
# HTML report
make test-coverage

# XML report for CI
./scripts/test-runner.sh --coverage --xml

# Both formats
./scripts/test-runner.sh --coverage --xml --json
```

### Coverage Files

| File | Description |
|------|-------------|
| `coverage.out` | Raw coverage data |
| `coverage.html` | HTML coverage report |
| `coverage.xml` | XML coverage report (for CI) |
| `test-results/` | Detailed test results |

### Coverage Thresholds

| Package | Threshold | Rationale |
|---------|-----------|-----------|
| `./internal/domain` | 90% | Core business logic |
| `./internal/service` | 85% | Business services |
| `./internal/handlers` | 80% | HTTP handlers |
| `./internal/repository` | 75% | Data access |
| **Overall** | **80%** | Project minimum |

### Viewing Reports

```bash
# Open HTML report
open coverage.html

# Terminal summary
go tool cover -func=coverage.out

# Package-specific coverage
go tool cover -func=coverage.out | grep "internal/service"
```

## CI/CD Integration

### GitHub Actions

The project includes a comprehensive GitHub Actions workflow (`.github/workflows/test.yml`) that runs:

- **Unit tests** on multiple Go versions
- **Integration tests** with real databases
- **Security scans** with gosec
- **Race condition detection**
- **Benchmark tests**
- **Coverage reporting** to Codecov

### Local CI Simulation

```bash
# Run the same tests as CI
make test-ci

# Equivalent to:
./scripts/test-runner.sh --type all --coverage --race --json --xml --setup --threshold 80
```

### Coverage Integration

Configure Codecov or similar service:

```yaml
# codecov.yml
coverage:
  status:
    project:
      default:
        target: 80%
    patch:
      default:
        target: 80%
```

## Writing Tests

### Test Structure

Follow Go testing conventions with table-driven tests:

```go
func TestMessageService_CreateMessage(t *testing.T) {
    tests := []struct {
        name        string
        phoneNumber string
        content     string
        wantErr     bool
        errType     error
    }{
        {
            name:        "valid message",
            phoneNumber: "+1234567890",
            content:     "Hello, World!",
            wantErr:     false,
        },
        {
            name:        "invalid phone number",
            phoneNumber: "invalid",
            content:     "Hello",
            wantErr:     true,
            errType:     domain.ErrInvalidPhoneNumber,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Test Categories

#### Unit Tests

```go
// +build !integration

func TestMessageValidation(t *testing.T) {
    // Fast, isolated test
}
```

#### Integration Tests

```go
// +build integration

func TestMessageRepository_Create(t *testing.T) {
    // Test with real database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    // Test implementation
}
```

#### Benchmark Tests

```go
func BenchmarkMessageProcessing(b *testing.B) {
    // Setup
    service := setupMessageService()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        service.ProcessMessage(testMessage)
    }
}
```

### Test Utilities

Create helper functions for common test scenarios:

```go
// testdata/helpers.go
func CreateTestMessage() *domain.Message {
    return &domain.Message{
        ID:          uuid.New(),
        PhoneNumber: "+1234567890",
        Content:     "Test message",
        Status:      domain.StatusPending,
        CreatedAt:   time.Now(),
    }
}

func SetupTestDB(t *testing.T) *sql.DB {
    // Database setup logic
}
```

### Mock Usage

Use the existing mocks in `internal/repository/mocks.go`:

```go
func TestMessageService_WithMocks(t *testing.T) {
    mockRepo := &repository.MockMessageRepository{}
    service := service.NewMessageService(mockRepo, nil)
    
    // Configure mock expectations
    mockRepo.On("Create", mock.Anything).Return(nil)
    
    // Test implementation
}
```

## Best Practices

### Test Organization

1. **One test file per source file**: `message.go` → `message_test.go`
2. **Test packages**: Use `package_test` for external testing
3. **Test data**: Store in `testdata/` directory
4. **Helpers**: Separate test utilities in `testdata/helpers.go`

### Test Naming

```go
// Good test names
func TestMessageService_CreateMessage_Success(t *testing.T)
func TestMessageService_CreateMessage_InvalidPhoneNumber(t *testing.T)
func TestMessageRepository_GetByID_NotFound(t *testing.T)

// Bad test names
func TestMessage(t *testing.T)
func TestCreate(t *testing.T)
func TestError(t *testing.T)
```

### Test Data

```go
// Use table-driven tests for multiple scenarios
tests := []struct {
    name     string
    input    string
    expected string
    wantErr  bool
}{
    {"valid input", "test", "test", false},
    {"empty input", "", "", true},
}
```

### Error Testing

```go
// Test both success and error cases
err := service.CreateMessage(invalidInput)
if err == nil {
    t.Error("expected error, got nil")
}

// Test specific error types
if !errors.Is(err, domain.ErrInvalidInput) {
    t.Errorf("expected ErrInvalidInput, got %v", err)
}
```

### Performance Testing

```go
func BenchmarkCriticalPath(b *testing.B) {
    // Setup expensive operations outside the benchmark
    service := setupComplexService()
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        service.CriticalOperation()
    }
}
```

## Troubleshooting

### Common Issues

#### Database Connection Failures

```bash
# Check PostgreSQL is running
pg_isready -h localhost -p 5432

# Test connection
psql "postgresql://postgres:password@localhost:5432/ims_test" -c "SELECT 1;"

# Create test database
createdb ims_test
```

#### Race Condition Failures

```bash
# Run specific test with race detection
go test -race -run TestSpecificFunction ./internal/service

# Increase timeout for race tests
go test -race -timeout 10m ./...
```

#### Coverage Issues

```bash
# Debug coverage
go test -coverprofile=debug.out ./internal/service
go tool cover -html=debug.out

# Check excluded files
go test -coverprofile=coverage.out -coverpkg=./... ./...
```

#### Memory Issues

```bash
# Run with memory profiling
go test -memprofile=mem.prof -bench=. ./...
go tool pprof mem.prof

# Check for memory leaks
go test -v -count=100 ./internal/service
```

### Debugging Tests

```bash
# Verbose output
go test -v ./...

# Specific test
go test -v -run TestSpecificFunction ./internal/service

# With timeout
go test -v -timeout 30s ./...

# Print all output
go test -v -count=1 ./...
```

### Performance Debugging

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=. ./...
go tool pprof cpu.prof

# Trace analysis
go test -trace=trace.out ./...
go tool trace trace.out
```

## Test Configuration

The project uses `test.config.yaml` for test configuration:

```yaml
# Coverage thresholds
coverage:
  threshold: 80
  fail_under_threshold: true

# Package-specific settings
packages:
  "./internal/domain":
    coverage_threshold: 90
```

## Integration with IDEs

### VS Code

Add to `.vscode/settings.json`:

```json
{
    "go.testFlags": ["-v"],
    "go.testTimeout": "5m",
    "go.coverOnSave": true,
    "go.coverageDecorator": {
        "type": "gutter",
        "coveredHighlightColor": "rgba(64,128,128,0.5)",
        "uncoveredHighlightColor": "rgba(128,64,64,0.25)"
    }
}
```

### GoLand/IntelliJ

Configure test runners:
1. Go to Run/Debug Configurations
2. Add new Go Test configuration
3. Set Environment: `TEST_ENV=true`
4. Set Program arguments: `-v -timeout 5m`

## Continuous Improvement

### Test Metrics

Monitor test metrics over time:
- **Test execution time**
- **Coverage percentage**
- **Flaky test detection**
- **Performance regression**

### Regular Reviews

1. **Weekly**: Review failed tests and flaky tests
2. **Monthly**: Analyze coverage reports and identify gaps
3. **Quarterly**: Performance benchmark comparisons
4. **Release**: Full test suite validation

### Test Automation

Automate common testing tasks:
- **Pre-commit hooks** for basic tests
- **Scheduled runs** for comprehensive testing
- **Performance monitoring** for regression detection
- **Coverage reporting** for quality gates 