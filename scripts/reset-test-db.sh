#!/bin/bash

set -e

echo "🔄 Resetting test database..."

# Database connection details
DB_HOST="localhost"
DB_PORT="5433"
DB_USER="calendar"
DB_NAME="calendar_test_db"

# Check if PostgreSQL container is running
if ! docker ps | grep -q calendar-postgres; then
    echo "❌ PostgreSQL container is not running. Please start it with 'docker-compose up -d'"
    exit 1
fi

echo "🗑️  Cleaning test database..."
cd services/calendar-core
export DATABASE_URL="postgresql://$DB_USER:calendar123@$DB_HOST:$DB_PORT/$DB_NAME"

# Reset database using Prisma
npx prisma migrate reset --force --skip-seed

echo "✅ Test database reset complete!"
