import os
import sys
import glob
import subprocess
from urllib.parse import urlparse

# Database connection string
CONNECTION_STRING = "postgres://avnadmin:AVNS_bn_194vT4dqm-DLrnoL@villagesite-arshadnoor585-d5bb.i.aivencloud.com:17865/defaultdb?sslmode=require"

# Parse connection string
url = urlparse(CONNECTION_STRING)
DB_HOST = url.hostname
DB_PORT = url.port
DB_USER = url.username
DB_PASSWORD = url.password
DB_NAME = url.path[1:]  # Remove leading slash

# Find the backup file
script_dir = os.path.dirname(os.path.abspath(__file__))
backup_dir = os.path.join(script_dir, "..", "backup")
backup_files = glob.glob(os.path.join(backup_dir, "*.sql"))

if not backup_files:
    print(f"Error: No .sql files found in {backup_dir}")
    sys.exit(1)

# Use the first .sql file found
BACKUP_FILE = backup_files[0]
print(f"Using backup file: {BACKUP_FILE}")

def run_command(command):
    """Run a command and return its output"""
    try:
        env = os.environ.copy()
        env["PGPASSWORD"] = DB_PASSWORD
        print(f"Executing: {command}")
        result = subprocess.run(
            command,
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            shell=True
        )
        if result.stdout:
            print("Output:", result.stdout)
        if result.stderr:
            print("Warning/Info:", result.stderr)
        return result.returncode == 0, result.stdout
    except Exception as e:
        print(f"Error executing command: {e}")
        return False, str(e)

def main():
    # Test connection
    print("Testing database connection...")
    success, output = run_command(
        f'psql "{CONNECTION_STRING}" -c "\\conninfo"'
    )
    if not success:
        print("Failed to connect to database")
        sys.exit(1)

    # Terminate existing connections
    print("Terminating existing connections...")
    run_command(
        f'psql "{CONNECTION_STRING}" -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = \'{DB_NAME}\' AND pid <> pg_backend_pid();"'
    )

    # Try restore using pg_restore first
    print("Attempting database restore using pg_restore...")
    success, _ = run_command(
        f'pg_restore --no-owner --no-privileges --clean --if-exists -v -d "{CONNECTION_STRING}" "{BACKUP_FILE}"'
    )

    # If pg_restore fails, try psql
    if not success:
        print("pg_restore failed, trying psql...")
        success, _ = run_command(
            f'psql "{CONNECTION_STRING}" -f "{BACKUP_FILE}"'
        )

    if success:
        print("Database restore completed successfully")
    else:
        print("Database restore failed")
        sys.exit(1)

if __name__ == "__main__":
    main() 