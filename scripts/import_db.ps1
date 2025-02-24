# Database connection parameters
$connString = "postgresql://postgres:postgres@localhost:5432/indiavillage"

# Parse connection string
$uri = [System.Uri]$connString
$dbname = $uri.AbsolutePath.TrimStart('/')
$dbuser = $uri.UserInfo.Split(':')[0]
$dbpassword = $uri.UserInfo.Split(':')[1]
$dbhost = $uri.Host
$dbport = $uri.Port

# Set up environment
$env:PGPASSWORD = $dbpassword

# Get script directory and backup directory
$scriptPath = $PSScriptRoot
$backupDir = Join-Path $scriptPath ".." "backup"
Write-Host "Looking for backup files in: $backupDir"

# Find backup files
$backupFiles = Get-ChildItem -Path $backupDir -Filter "*.sql"
if ($backupFiles.Count -eq 0) {
    Write-Host "No .sql files found in $backupDir"
    exit 1
}

# Use the first backup file
$backupFile = $backupFiles[0].FullName
Write-Host "Found backup file: $backupFile"

# Try pg_restore first
Write-Host "Attempting restore with pg_restore..."
$pgRestoreCmd = "pg_restore -h $dbhost -p $dbport -U $dbuser -d $dbname -c -v `"$backupFile`""
Write-Host "Running command: $pgRestoreCmd"

try {
    $result = Invoke-Expression $pgRestoreCmd
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Database restore completed successfully!"
        exit 0
    }
} catch {
    Write-Host "pg_restore failed: $_"
}

# If pg_restore fails, try psql
Write-Host "Attempting restore with psql..."
$psqlCmd = "psql -h $dbhost -p $dbport -U $dbuser -d $dbname -f `"$backupFile`""
Write-Host "Running command: $psqlCmd"

try {
    $result = Invoke-Expression $psqlCmd
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Database restore completed successfully using psql!"
        exit 0
    }
} catch {
    Write-Host "psql failed: $_"
}

Write-Host "`nBoth pg_restore and psql failed. Please ensure PostgreSQL tools are installed:"
Write-Host "1. Download PostgreSQL installer from: https://www.postgresql.org/download/"
Write-Host "2. Run the installer and select 'Command Line Tools'"
Write-Host "3. Add PostgreSQL bin directory to your PATH"
Write-Host "4. Restart PowerShell and try again"
exit 1 