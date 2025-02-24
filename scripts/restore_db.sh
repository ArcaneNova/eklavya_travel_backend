#!/bin/bash

# Wait for PostgreSQL to be ready
until PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; do
  echo "Waiting for PostgreSQL to be ready..."
  sleep 1
done

# Check if backup file exists
BACKUP_FILE="/app/backup/latest.sql"
if [ ! -f "$BACKUP_FILE" ]; then
    echo "No backup file found at $BACKUP_FILE"
    exit 1
fi

# Restore the database
echo "Starting database restore..."
PGPASSWORD=$DB_PASSWORD pg_restore \
    -h $DB_HOST \
    -p $DB_PORT \
    -U $DB_USER \
    -d $DB_NAME \
    -v \
    --clean \
    --if-exists \
    "$BACKUP_FILE"

echo "Database restore completed" 