#!/bin/bash

# Get environment variables
source ../.env

# Create backup directory if it doesn't exist
mkdir -p ../backup

# Backup filename with timestamp
BACKUP_FILE="../backup/indiavillage_$(date +%Y%m%d_%H%M%S).sql"

# Create the backup
PGPASSWORD=$DB_PASSWORD pg_dump \
  -h $DB_HOST \
  -p $DB_PORT \
  -U $DB_USER \
  -d $DB_NAME \
  -F c \
  -b \
  -v \
  -f "$BACKUP_FILE"

echo "Backup completed: $BACKUP_FILE" 