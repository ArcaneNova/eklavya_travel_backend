#!/bin/bash

# Function to check if database exists
check_db_exists() {
    PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -lqt | cut -d \| -f 1 | grep -qw "$DB_NAME"
}

# Function to check if database is empty
check_db_empty() {
    table_count=$(PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'")
    [ "$table_count" -eq 0 ]
}

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
until PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d "postgres" -c '\q' 2>/dev/null; do
    echo "PostgreSQL is unavailable - sleeping"
    sleep 1
done

echo "PostgreSQL is up - checking database"

# Create database if it doesn't exist
if ! check_db_exists; then
    echo "Creating database $DB_NAME..."
    PGPASSWORD=$DB_PASSWORD createdb -h "$DB_HOST" -U "$DB_USER" "$DB_NAME"
fi

# Check if we need to restore the database
if check_db_empty && [ -f /app/backup/latest.sql ]; then
    echo "Database is empty and backup exists - starting restore..."
    PGPASSWORD=$DB_PASSWORD pg_restore \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -v \
        --clean \
        --if-exists \
        "/app/backup/latest.sql"
    
    if [ $? -eq 0 ]; then
        echo "Database restore completed successfully"
    else
        echo "Database restore failed"
        exit 1
    fi
else
    echo "Database already exists and is not empty, or no backup file found"
fi

# Start the application
echo "Starting the application..."
exec ./main 