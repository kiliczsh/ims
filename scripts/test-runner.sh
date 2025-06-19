#!/usr/bin/env bash

# IMS Test Runner - Comprehensive testing script for the Insider Message Sender
# Usage: ./scripts/test-runner.sh [options]
# 
# This script provides comprehensive testing capabilities including:
# - Unit tests with coverage
# - Integration tests
# - Benchmark tests
# - Race condition detection
# - Test environment setup
# - Coverage reporting and analysis
# - CI/CD integration

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_ROOT}"

# Source common utilities
if [ -f "${SCRIPT_DIR}/common.sh" ]; then
    source "${SCRIPT_DIR}/common.sh"
fi

# Test configuration
TEST_TIMEOUT="5m"
COVERAGE_THRESHOLD="80"
COVERAGE_OUT="coverage.out"
COVERAGE_HTML="coverage.html"
COVERAGE_XML="coverage.xml"
BENCHMARK_COUNT="3"
BENCHMARK_TIME="1s"
TEST_RESULTS_DIR="test-results"
INTEGRATION_TEST_TAG="integration"
BENCHMARK_TEST_TAG="benchmark"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test environments
TEST_ENV_UNIT="unit"
TEST_ENV_INTEGRATION="integration"
TEST_ENV_ALL="all"

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${PURPLE}==== $1 ====${NC}"
}

# Show usage information
show_usage() {
    cat << EOF
IMS Test Runner - Comprehensive testing for Insider Message Sender

USAGE:
    ./scripts/test-runner.sh [OPTIONS]

OPTIONS:
    -t, --type TYPE         Test type: unit, integration, benchmark, all (default: unit)
    -c, --coverage         Generate coverage reports
    -r, --race             Enable race condition detection
    -v, --verbose          Verbose output
    -w, --watch            Watch mode (re-run tests on file changes)
    -p, --package PACKAGE  Run tests for specific package
    -f, --filter FILTER    Run tests matching filter pattern
    -b, --benchmark        Run benchmark tests
    -j, --json             Output results in JSON format
    -x, --xml              Generate XML coverage report
    --threshold N          Coverage threshold percentage (default: 80)
    --timeout DURATION     Test timeout (default: 5m)
    --count N              Run each test N times (default: 1)
    --parallel N           Run tests with N parallel processes
    --clean                Clean test artifacts before running
    --setup                Setup test environment
    --no-cache             Disable test cache
    --failfast             Stop on first test failure
    -h, --help             Show this help message

EXAMPLES:
    # Run unit tests with coverage
    ./scripts/test-runner.sh --type unit --coverage

    # Run integration tests
    ./scripts/test-runner.sh --type integration

    # Run all tests with race detection and verbose output
    ./scripts/test-runner.sh --type all --race --verbose

    # Run benchmarks
    ./scripts/test-runner.sh --benchmark

    # Run tests for specific package
    ./scripts/test-runner.sh --package ./internal/service

    # Watch mode for development
    ./scripts/test-runner.sh --watch --coverage

    # Generate reports for CI
    ./scripts/test-runner.sh --coverage --json --xml

EOF
}

# Parse command line arguments
parse_args() {
    TEST_TYPE="unit"
    ENABLE_COVERAGE=false
    ENABLE_RACE=false
    VERBOSE=false
    WATCH_MODE=false
    PACKAGE_FILTER=""
    TEST_FILTER=""
    ENABLE_BENCHMARK=false
    JSON_OUTPUT=false
    XML_OUTPUT=false
    TEST_COUNT=1
    PARALLEL_PROCS=""
    CLEAN_ARTIFACTS=false
    SETUP_ENV=false
    NO_CACHE=false
    FAIL_FAST=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--type)
                TEST_TYPE="$2"
                shift 2
                ;;
            -c|--coverage)
                ENABLE_COVERAGE=true
                shift
                ;;
            -r|--race)
                ENABLE_RACE=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -w|--watch)
                WATCH_MODE=true
                shift
                ;;
            -p|--package)
                PACKAGE_FILTER="$2"
                shift 2
                ;;
            -f|--filter)
                TEST_FILTER="$2"
                shift 2
                ;;
            -b|--benchmark)
                ENABLE_BENCHMARK=true
                shift
                ;;
            -j|--json)
                JSON_OUTPUT=true
                shift
                ;;
            -x|--xml)
                XML_OUTPUT=true
                shift
                ;;
            --threshold)
                COVERAGE_THRESHOLD="$2"
                shift 2
                ;;
            --timeout)
                TEST_TIMEOUT="$2"
                shift 2
                ;;
            --count)
                TEST_COUNT="$2"
                shift 2
                ;;
            --parallel)
                PARALLEL_PROCS="$2"
                shift 2
                ;;
            --clean)
                CLEAN_ARTIFACTS=true
                shift
                ;;
            --setup)
                SETUP_ENV=true
                shift
                ;;
            --no-cache)
                NO_CACHE=true
                shift
                ;;
            --failfast)
                FAIL_FAST=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # Validate test type
    case $TEST_TYPE in
        unit|integration|benchmark|all) ;;
        *)
            print_error "Invalid test type: $TEST_TYPE"
            show_usage
            exit 1
            ;;
    esac
}

# Setup test environment
setup_test_environment() {
    print_header "Setting up test environment"
    
    # Create test results directory
    mkdir -p "${TEST_RESULTS_DIR}"
    
    # Check for required tools
    if ! command -v go >/dev/null 2>&1; then
        print_error "Go is not installed"
        exit 1
    fi
    
    # Install test dependencies if needed
    if [ ! -f "go.mod" ]; then
        print_error "go.mod not found. Run from project root."
        exit 1
    fi
    
    # Download dependencies
    print_status "Downloading dependencies..."
    go mod download
    go mod tidy
    
    # Install additional test tools
    print_status "Installing test tools..."
    
    # For XML coverage reports
    if $XML_OUTPUT && ! command -v gocov >/dev/null 2>&1; then
        go install github.com/axw/gocov/gocov@latest
    fi
    
    if $XML_OUTPUT && ! command -v gocov-xml >/dev/null 2>&1; then
        go install github.com/AlekSi/gocov-xml@latest
    fi
    
    # For test watching
    if $WATCH_MODE && ! command -v reflex >/dev/null 2>&1; then
        print_warning "Installing reflex for watch mode..."
        go install github.com/cespare/reflex@latest
    fi
    
    print_success "Test environment setup complete"
}

# Clean test artifacts
clean_test_artifacts() {
    print_status "Cleaning test artifacts..."
    rm -rf "${TEST_RESULTS_DIR}"
    rm -f "${COVERAGE_OUT}" "${COVERAGE_HTML}" "${COVERAGE_XML}"
    rm -f test.json benchmark.txt profile.out
    go clean -testcache
    print_success "Test artifacts cleaned"
}

# Build test command with options
build_test_command() {
    local test_cmd="go test"
    local package_pattern="./..."
    
    # Package filter
    if [ -n "$PACKAGE_FILTER" ]; then
        package_pattern="$PACKAGE_FILTER"
    fi
    
    # Basic options
    test_cmd="$test_cmd $package_pattern"
    
    # Timeout
    test_cmd="$test_cmd -timeout $TEST_TIMEOUT"
    
    # Test count
    if [ "$TEST_COUNT" -gt 1 ]; then
        test_cmd="$test_cmd -count $TEST_COUNT"
    fi
    
    # Parallel execution
    if [ -n "$PARALLEL_PROCS" ]; then
        test_cmd="$test_cmd -parallel $PARALLEL_PROCS"
    fi
    
    # Verbose output
    if $VERBOSE; then
        test_cmd="$test_cmd -v"
    fi
    
    # Race detection
    if $ENABLE_RACE; then
        test_cmd="$test_cmd -race"
    fi
    
    # Coverage
    if $ENABLE_COVERAGE; then
        test_cmd="$test_cmd -coverprofile=$COVERAGE_OUT -covermode=atomic"
    fi
    
    # Test filter
    if [ -n "$TEST_FILTER" ]; then
        test_cmd="$test_cmd -run $TEST_FILTER"
    fi
    
    # JSON output
    if $JSON_OUTPUT; then
        test_cmd="$test_cmd -json | tee ${TEST_RESULTS_DIR}/test-results.json"
    fi
    
    # No cache
    if $NO_CACHE; then
        test_cmd="$test_cmd -count=1"
    fi
    
    # Fail fast
    if $FAIL_FAST; then
        test_cmd="$test_cmd -failfast"
    fi
    
    echo "$test_cmd"
}

# Run unit tests
run_unit_tests() {
    print_header "Running Unit Tests"
    
    local cmd
    cmd=$(build_test_command)
    
    # Exclude integration tests
    cmd="$cmd -short"
    
    print_status "Command: $cmd"
    eval "$cmd"
}

# Run integration tests
run_integration_tests() {
    print_header "Running Integration Tests"
    
    # Check if database is available for integration tests
    if [ -f ".env" ]; then
        set -a
        source .env
        set +a
        
        # Verify database connection
        if [ -n "${DATABASE_URL:-}" ]; then
            print_status "Testing database connection..."
            if ! go run -tags integration ./cmd/server/main.go --check-db 2>/dev/null; then
                print_warning "Database connection failed, some integration tests may fail"
            fi
        fi
    else
        print_warning "No .env file found, integration tests may fail"
    fi
    
    local cmd
    cmd=$(build_test_command)
    
    # Run only integration tests (tests with build tag or longer running tests)
    cmd="$cmd -tags integration"
    
    print_status "Command: $cmd"
    eval "$cmd"
}

# Run benchmark tests
run_benchmark_tests() {
    print_header "Running Benchmark Tests"
    
    local bench_cmd="go test"
    local package_pattern="./..."
    
    if [ -n "$PACKAGE_FILTER" ]; then
        package_pattern="$PACKAGE_FILTER"
    fi
    
    bench_cmd="$bench_cmd $package_pattern"
    bench_cmd="$bench_cmd -bench=. -benchmem"
    bench_cmd="$bench_cmd -count=$BENCHMARK_COUNT"
    bench_cmd="$bench_cmd -benchtime=$BENCHMARK_TIME"
    bench_cmd="$bench_cmd -run=^$ " # Don't run regular tests
    
    if $VERBOSE; then
        bench_cmd="$bench_cmd -v"
    fi
    
    # Save benchmark results
    bench_cmd="$bench_cmd | tee ${TEST_RESULTS_DIR}/benchmark.txt"
    
    print_status "Command: $bench_cmd"
    eval "$bench_cmd"
}

# Generate coverage reports
generate_coverage_reports() {
    if [ ! -f "$COVERAGE_OUT" ]; then
        print_warning "No coverage file found: $COVERAGE_OUT"
        return 1
    fi
    
    print_header "Generating Coverage Reports"
    
    # HTML report
    print_status "Generating HTML coverage report..."
    go tool cover -html="$COVERAGE_OUT" -o "$COVERAGE_HTML"
    print_success "HTML report: $COVERAGE_HTML"
    
    # Text summary
    print_status "Coverage summary:"
    go tool cover -func="$COVERAGE_OUT" | tail -1
    
    # XML report for CI
    if $XML_OUTPUT; then
        print_status "Generating XML coverage report..."
        if command -v gocov >/dev/null 2>&1 && command -v gocov-xml >/dev/null 2>&1; then
            gocov convert "$COVERAGE_OUT" | gocov-xml > "$COVERAGE_XML"
            print_success "XML report: $COVERAGE_XML"
        else
            print_warning "gocov/gocov-xml not available for XML report"
        fi
    fi
    
    # Check coverage threshold
    local coverage_percent
    coverage_percent=$(go tool cover -func="$COVERAGE_OUT" | tail -1 | awk '{print $3}' | sed 's/%//')
    
    if [ -n "$coverage_percent" ]; then
        print_status "Current coverage: ${coverage_percent}%"
        if (( $(echo "$coverage_percent >= $COVERAGE_THRESHOLD" | bc -l) )); then
            print_success "Coverage threshold met (${coverage_percent}% >= ${COVERAGE_THRESHOLD}%)"
        else
            print_error "Coverage threshold not met (${coverage_percent}% < ${COVERAGE_THRESHOLD}%)"
            return 1
        fi
    fi
}

# Watch mode for development
run_watch_mode() {
    print_header "Starting Watch Mode"
    print_status "Watching for file changes..."
    print_status "Press Ctrl+C to stop"
    
    if ! command -v reflex >/dev/null 2>&1; then
        print_error "reflex not found. Install with: go install github.com/cespare/reflex@latest"
        exit 1
    fi
    
    # Watch Go files and re-run tests
    reflex -r '\.go$' -- bash -c "
        echo '==== Running tests due to file change ====' &&
        $0 --type $TEST_TYPE $([ $ENABLE_COVERAGE = true ] && echo '--coverage') $([ $VERBOSE = true ] && echo '--verbose')
    "
}

# Display test results summary
show_test_summary() {
    print_header "Test Summary"
    
    if [ -f "${TEST_RESULTS_DIR}/test-results.json" ]; then
        # Parse JSON results if available
        local total_tests passed_tests failed_tests
        total_tests=$(jq -r '.Action | select(. == "pass" or . == "fail")' "${TEST_RESULTS_DIR}/test-results.json" 2>/dev/null | wc -l || echo "0")
        passed_tests=$(jq -r '.Action | select(. == "pass")' "${TEST_RESULTS_DIR}/test-results.json" 2>/dev/null | wc -l || echo "0")
        failed_tests=$(jq -r '.Action | select(. == "fail")' "${TEST_RESULTS_DIR}/test-results.json" 2>/dev/null | wc -l || echo "0")
        
        echo "Total Tests: $total_tests"
        echo "Passed: $passed_tests"
        echo "Failed: $failed_tests"
    fi
    
    if [ -f "$COVERAGE_OUT" ]; then
        echo "Coverage Report: $COVERAGE_HTML"
    fi
    
    if [ -f "${TEST_RESULTS_DIR}/benchmark.txt" ]; then
        echo "Benchmark Results: ${TEST_RESULTS_DIR}/benchmark.txt"
    fi
    
    echo ""
    echo "Test artifacts saved in: $TEST_RESULTS_DIR"
}

# Main execution function
main() {
    parse_args "$@"
    
    print_header "IMS Test Runner"
    print_status "Test type: $TEST_TYPE"
    print_status "Project root: $PROJECT_ROOT"
    
    # Setup environment if requested
    if $SETUP_ENV; then
        setup_test_environment
    fi
    
    # Clean artifacts if requested
    if $CLEAN_ARTIFACTS; then
        clean_test_artifacts
    fi
    
    # Create test results directory
    mkdir -p "${TEST_RESULTS_DIR}"
    
    # Handle watch mode
    if $WATCH_MODE; then
        run_watch_mode
        exit $?
    fi
    
    # Track overall success
    local overall_success=true
    
    # Run tests based on type
    case $TEST_TYPE in
        unit)
            if ! run_unit_tests; then
                overall_success=false
            fi
            ;;
        integration)
            if ! run_integration_tests; then
                overall_success=false
            fi
            ;;
        benchmark)
            if ! run_benchmark_tests; then
                overall_success=false
            fi
            ;;
        all)
            print_status "Running all test types..."
            if ! run_unit_tests; then
                overall_success=false
            fi
            echo ""
            if ! run_integration_tests; then
                overall_success=false
            fi
            echo ""
            if $ENABLE_BENCHMARK; then
                if ! run_benchmark_tests; then
                    overall_success=false
                fi
            fi
            ;;
    esac
    
    # Run standalone benchmarks if requested
    if $ENABLE_BENCHMARK && [ "$TEST_TYPE" != "benchmark" ] && [ "$TEST_TYPE" != "all" ]; then
        echo ""
        if ! run_benchmark_tests; then
            overall_success=false
        fi
    fi
    
    # Generate coverage reports
    if $ENABLE_COVERAGE; then
        echo ""
        if ! generate_coverage_reports; then
            overall_success=false
        fi
    fi
    
    # Show summary
    echo ""
    show_test_summary
    
    # Final result
    if $overall_success; then
        print_success "All tests completed successfully!"
        exit 0
    else
        print_error "Some tests failed!"
        exit 1
    fi
}

# Run main function with all arguments
main "$@" 