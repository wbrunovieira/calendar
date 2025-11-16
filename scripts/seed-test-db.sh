#!/bin/bash

set -e

echo "🌱 Seeding test database..."

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

cd services/calendar-core
export DATABASE_URL="postgresql://$DB_USER:calendar123@$DB_HOST:$DB_PORT/$DB_NAME"

# Run seed script
npm run prisma:seed

echo "✅ Test database seeded successfully!"
