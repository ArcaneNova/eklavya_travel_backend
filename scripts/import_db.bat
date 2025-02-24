@echo off
setlocal EnableDelayedExpansion

rem Check if PostgreSQL tools are installed
where pg_restore >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo PostgreSQL tools are not installed or not in PATH
    echo Please install PostgreSQL tools and add them to your PATH
    echo Download from: https://www.postgresql.org/download/windows/
    exit /b 1
)

rem Set database connection details from the connection string
set DB_HOST=villagesite-arshadnoor585-d5bb.i.aivencloud.com
set DB_PORT=17865
set DB_USER=avnadmin
set DB_PASSWORD=AVNS_bn_194vT4dqm-DLrnoL
set DB_NAME=defaultdb

echo Using database connection settings:
echo Host: %DB_HOST%
echo Port: %DB_PORT%
echo User: %DB_USER%
echo Database: %DB_NAME%

rem Check if backup file exists
set BACKUP_FILE=..\backup\indiavillage.sql
if not exist "%BACKUP_FILE%" (
    echo Backup file not found: %BACKUP_FILE%
    exit /b 1
)

echo Starting database restore...
set PGPASSWORD=%DB_PASSWORD%

rem Test connection
echo Testing database connection...
psql "postgresql://%DB_USER%:%DB_PASSWORD%@%DB_HOST%:%DB_PORT%/%DB_NAME%?sslmode=require" -c "\conninfo"
if %ERRORLEVEL% NEQ 0 (
    echo Failed to connect to database
    exit /b 1
)

rem First, terminate all connections to the database
echo Terminating existing connections...
psql "postgresql://%DB_USER%:%DB_PASSWORD%@%DB_HOST%:%DB_PORT%/%DB_NAME%?sslmode=require" -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%DB_NAME%' AND pid <> pg_backend_pid();"

rem Restore the database
echo Restoring database from backup...
pg_restore -h %DB_HOST% -p %DB_PORT% -U %DB_USER% -d %DB_NAME% --clean --if-exists --no-owner --no-privileges -v "%BACKUP_FILE%" --no-password

if %ERRORLEVEL% EQU 0 (
    echo Database restore completed successfully
) else (
    echo Database restore failed
    exit /b 1
)

endlocal 