@echo off
setlocal

rem Load environment variables from .env file
for /f "tokens=1,2 delims==" %%G in (..\..\.env) do (
    set %%G=%%H
)

rem Create backup directory if it doesn't exist
if not exist "..\backup" mkdir "..\backup"

rem Set backup filename with timestamp
set TIMESTAMP=%DATE:~10,4%%DATE:~4,2%%DATE:~7,2%_%TIME:~0,2%%TIME:~3,2%%TIME:~6,2%
set TIMESTAMP=%TIMESTAMP: =0%
set BACKUP_FILE=..\backup\indiavillage_%TIMESTAMP%.sql

echo Creating backup...

rem Set environment variables manually if .env file is not found
if not defined DB_HOST set DB_HOST=localhost
if not defined DB_PORT set DB_PORT=5432
if not defined DB_USER set DB_USER=postgres
if not defined DB_PASSWORD set DB_PASSWORD=1234
if not defined DB_NAME set DB_NAME=indiavillage

echo Using database connection settings:
echo Host: %DB_HOST%
echo Port: %DB_PORT%
echo User: %DB_USER%
echo Database: %DB_NAME%

set PGPASSWORD=%DB_PASSWORD%
pg_dump -h %DB_HOST% -p %DB_PORT% -U %DB_USER% -d %DB_NAME% -F c -b -v -f "%BACKUP_FILE%"

if %ERRORLEVEL% EQU 0 (
    echo Backup completed successfully: %BACKUP_FILE%
    copy "%BACKUP_FILE%" "..\backup\latest.sql"
) else (
    echo Backup failed
    exit /b 1
)

endlocal 