#!/bin/bash

# Insider Message Sender - Run Script
# This script sets up environment variables and runs the application

set -e

echo "🚀 Starting Insider Message Sender..."

# Load environment variables from .env file if it exists
if [[ -f ".env" ]]; then
    echo "📝 Loading environment variables from .env file..."
    set -a  # Automatically export all variables
    source .env
    set +a  # Stop automatically exporting
    echo "✅ Environment variables loaded from .env"
else
    echo "⚠️  .env file not found. Using default configuration."
    echo "💡 For custom configuration, create a .env file:"
    echo "   cp .env.example .env"
    echo "   # Edit .env with your values"
    echo ""
    exit 1
fi

# Check if database setup is needed
if [[ "$1" == "--setup-db" || "$1" == "-s" ]]; then
    echo "🗄️  Setting up database..."
    
    # Check if psql is available
    if ! command -v psql &> /dev/null; then
        echo "❌ psql is not installed. Please install PostgreSQL client."
        exit 1
    fi
    
    # Run database migrations
    echo "Running database migrations..."
    psql "$DATABASE_URL" -f migrations/001_create_messages.sql
    
    # Insert sample data
    echo "Inserting sample data..."
    psql "$DATABASE_URL" -f scripts/insert_sample_data.sql
    
    echo "✅ Database setup completed"
fi

# Build the application if needed
if [[ ! -f "bin/ims" ]]; then
    echo "🔨 Building application..."
    mkdir -p bin
    go build -o bin/ims cmd/server/main.go
    echo "✅ Build completed"
fi

# Start the application
echo "🌟 Starting Insider Message Sender..."
echo ""
echo "📍 Server will start on: http://localhost:$SERVER_PORT"
echo "🏥 Health check: http://localhost:$SERVER_PORT/api/health"
echo "📚 API docs: http://localhost:$SERVER_PORT/api/docs"
echo "🔑 API Key: $WEBHOOK_AUTH_KEY"
echo ""
echo "💡 To test the API:"
echo "   curl http://localhost:$SERVER_PORT/api/health"
echo ""
echo "💡 To start the scheduler:"
echo "   curl -X POST http://localhost:$SERVER_PORT/api/control \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -H 'x-ins-auth-key: $WEBHOOK_AUTH_KEY' \\"
echo "     -d '{\"action\": \"start\"}'"
echo ""
echo "⏹️  Press Ctrl+C to stop the server"
echo ""

# Run the application
./bin/ims 