#!/bin/bash

set -e

echo "🗄️  Setting up test database..."

# Database connection details
DB_HOST="localhost"
DB_PORT="5433"
DB_USER="calendar"
DB_NAME="calendar_test_db"
MAIN_DB="calendar_db"

# Check if PostgreSQL container is running
if ! docker ps | grep -q calendar-postgres; then
    echo "❌ PostgreSQL container is not running. Please start it with 'docker-compose up -d'"
    exit 1
fi

echo "📦 Checking if test database exists..."

# Check if test database exists
DB_EXISTS=$(docker exec calendar-postgres psql -U $DB_USER -d $MAIN_DB -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'")

if [ "$DB_EXISTS" = "1" ]; then
    echo "🗑️  Dropping existing test database..."
    docker exec calendar-postgres psql -U $DB_USER -d $MAIN_DB -c "DROP DATABASE $DB_NAME;"
fi

echo "✨ Creating test database..."
docker exec calendar-postgres psql -U $DB_USER -d $MAIN_DB -c "CREATE DATABASE $DB_NAME;"

echo "🔄 Running migrations on test database..."
cd services/calendar-core
export DATABASE_URL="postgresql://$DB_USER:calendar123@$DB_HOST:$DB_PORT/$DB_NAME"
npx prisma migrate deploy

echo "✅ Test database setup complete!"
echo ""
echo "📝 Test database URL: $DATABASE_URL"
echo ""
echo "You can now run tests with:"
echo "  npm run test"
echo "  npm run test:e2e"
