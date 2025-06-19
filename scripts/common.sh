#!/bin/bash

# Common utilities for Insider Message Sender scripts

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to load environment variables from .env file
load_env() {
    local env_file="${1:-.env}"
    
    if [[ -f "$env_file" ]]; then
        print_info "Loading environment variables from $env_file..."
        
        # Validate .env file format
        if ! grep -q '^[A-Z_][A-Z0-9_]*=' "$env_file" 2>/dev/null; then
            print_warning "$env_file exists but appears to be empty or invalid"
            return 1
        fi
        
        # Load variables (automatically export them)
        set -a
        source "$env_file"
        set +a
        
        print_success "Environment variables loaded from $env_file"
        return 0
    else
        print_warning "$env_file file not found"
        
        if [[ "$env_file" == ".env" ]]; then
            print_info "You can create one with:"
            print_info "  cp .env.example .env"
            print_info "  # Edit .env with your actual values"
        fi
        
        return 1
    fi
}

# Function to check if required environment variables are set
check_required_env() {
    local missing_vars=()
    
    for var in "$@"; do
        if [[ -z "${!var}" ]]; then
            missing_vars+=("$var")
        fi
    done
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        print_error "Missing required environment variables:"
        for var in "${missing_vars[@]}"; do
            print_error "  - $var"
        done
        
        print_info "Please set these in your .env file or environment"
        return 1
    fi
    
    return 0
}

# Function to display environment summary
show_env_summary() {
    print_info "Environment configuration summary:"
    echo "  📄 Database: ${DATABASE_URL:-'Not set'}"
    echo "  🔗 Webhook: ${WEBHOOK_URL:-'Not set'}"
    echo "  🔑 Auth Key: ${WEBHOOK_AUTH_KEY:0:20}... (${#WEBHOOK_AUTH_KEY} chars)"
    echo "  🌐 Server Port: ${SERVER_PORT:-'8080'}"
    echo "  📊 Redis: ${REDIS_URL:-'Not configured'}"
    echo "  ⏰ Scheduler: Every ${SCHEDULER_INTERVAL:-'2m'}, ${SCHEDULER_BATCH_SIZE:-'2'} msgs/batch"
}

# Function to create .env from template if it doesn't exist
ensure_env_file() {
    if [[ ! -f ".env" ]]; then
        if [[ -f ".env.example" ]]; then
            print_info "Creating .env from .env.example template..."
            cp .env.example .env
            print_success ".env file created from template"
            print_warning "Please edit .env with your actual configuration values"
            return 0
        else
            print_warning "Neither .env nor .env.example found"
            return 1
        fi
    else
        return 0
    fi
}

# Function to validate critical paths/URLs
validate_config() {
    local errors=0
    
    # Check database URL format
    if [[ -n "$DATABASE_URL" ]]; then
        if [[ ! "$DATABASE_URL" =~ ^postgresql:// ]]; then
            print_error "DATABASE_URL must start with 'postgresql://'"
            errors=$((errors + 1))
        fi
    fi
    
    # Check webhook URL format
    if [[ -n "$WEBHOOK_URL" ]]; then
        if [[ ! "$WEBHOOK_URL" =~ ^https?:// ]]; then
            print_error "WEBHOOK_URL must start with 'http://' or 'https://'"
            errors=$((errors + 1))
        fi
    fi
    
    # Check server port is numeric
    if [[ -n "$SERVER_PORT" ]]; then
        if ! [[ "$SERVER_PORT" =~ ^[0-9]+$ ]]; then
            print_error "SERVER_PORT must be a number"
            errors=$((errors + 1))
        fi
    fi
    
    return $errors
} 