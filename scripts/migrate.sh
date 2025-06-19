#!/bin/bash

# Insider Message Sender - Database Migration Script
# This script handles database migrations and setup

set -e

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common utilities
source "$SCRIPT_DIR/common.sh"

# Load environment variables
load_env

# Check required environment variables
if ! check_required_env "DATABASE_URL"; then
    exit 1
fi

# Validate configuration
if ! validate_config; then
    print_error "Configuration validation failed"
    exit 1
fi

# Check if psql is available
if ! command -v psql &> /dev/null; then
    print_error "psql is not installed. Please install PostgreSQL client."
    echo "On macOS: brew install postgresql"
    echo "On Ubuntu: sudo apt-get install postgresql-client"
    exit 1
fi

print_info "Starting database migration process..."
show_env_summary
echo

# Test database connection
print_info "Testing database connection..."
if psql "$DATABASE_URL" -c "SELECT 1;" > /dev/null 2>&1; then
    print_success "Database connection successful"
else
    print_error "Failed to connect to database"
    print_info "Please check your DATABASE_URL and ensure the database server is running"
    exit 1
fi

# Function to check if a migration has been applied
check_migration() {
    local migration_name="$1"
    local result=$(psql "$DATABASE_URL" -t -c "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'migration_history');")
    
    if [[ "$result" =~ "t" ]]; then
        # Migration history table exists, check if this migration was applied
        local applied=$(psql "$DATABASE_URL" -t -c "SELECT EXISTS(SELECT 1 FROM migration_history WHERE migration_name = '$migration_name');")
        if [[ "$applied" =~ "t" ]]; then
            return 0  # Migration already applied
        else
            return 1  # Migration not applied
        fi
    else
        return 1  # Migration history table doesn't exist, so no migrations applied
    fi
}

# Function to record migration
record_migration() {
    local migration_name="$1"
    psql "$DATABASE_URL" -c "INSERT INTO migration_history (migration_name, applied_at) VALUES ('$migration_name', CURRENT_TIMESTAMP);" > /dev/null
}

# Create migration history table if it doesn't exist
print_info "Setting up migration history table..."
psql "$DATABASE_URL" -c "
CREATE TABLE IF NOT EXISTS migration_history (
    id SERIAL PRIMARY KEY,
    migration_name VARCHAR(255) NOT NULL UNIQUE,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);" > /dev/null

print_success "Migration history table ready"

# Apply migrations
migrations=(
    "001_create_messages.sql"
    "002_create_audit_logs.sql"
)

for migration in "${migrations[@]}"; do
    migration_name=$(basename "$migration" .sql)
    
    if check_migration "$migration_name"; then
        print_info "Migration $migration_name already applied, skipping..."
    else
        print_info "Applying migration: $migration_name"
        
        if psql "$DATABASE_URL" -f "migrations/$migration" > /dev/null; then
            record_migration "$migration_name"
            print_success "Migration $migration_name applied successfully"
        else
            print_error "Failed to apply migration $migration_name"
            exit 1
        fi
    fi
done

# Insert sample data if requested
if [[ "$1" == "--with-sample-data" || "$1" == "-s" ]]; then
    print_info "Inserting sample data..."
    
    # Check if sample data already exists
    local count=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM messages;" | tr -d ' ')
    
    if [ "$count" -gt 0 ]; then
        print_warning "Sample data already exists ($count messages found), skipping..."
    else
        if psql "$DATABASE_URL" -f "scripts/insert_sample_data.sql" > /dev/null; then
            print_success "Sample data inserted successfully"
        else
            print_error "Failed to insert sample data"
            exit 1
        fi
    fi
fi

print_success "All migrations completed successfully!"

# Show current migration status
print_info "Current migration status:"
psql "$DATABASE_URL" -c "SELECT migration_name, applied_at FROM migration_history ORDER BY applied_at;"

print_info "Database tables:"
psql "$DATABASE_URL" -c "\dt"

print_success "Migration process completed!" 