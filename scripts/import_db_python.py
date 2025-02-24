import os
import subprocess
from pathlib import Path
import psycopg2
from urllib.parse import urlparse

def parse_connection_string(conn_string):
    # Parse connection string into components
    result = urlparse(conn_string)
    return {
        'dbname': result.path[1:],
        'user': result.username,
        'password': result.password,
        'host': result.hostname,
        'port': result.port or 5432
    }

def find_backup_file():
    # Get the directory where this script is located
    script_dir = Path(__file__).parent.parent
    backup_dir = script_dir / 'backup'
    print(f"Looking for backup files in: {backup_dir}")
    
    # Look for .sql or .dump files
    backup_files = list(backup_dir.glob('*.sql')) + list(backup_dir.glob('*.dump'))
    if not backup_files:
        raise FileNotFoundError(f"No backup files found in {backup_dir}")
    
    # Use the first backup file found
    return str(backup_files[0])

def import_database():
    # Connection string (replace with your actual connection details)
    conn_string = "postgresql://postgres:postgres@localhost:5432/indiavillage"
    
    try:
        # Find backup file
        backup_file = find_backup_file()
        print(f"Found backup file: {backup_file}")
        
        # Parse connection string
        db_params = parse_connection_string(conn_string)
        
        # Construct pg_restore command
        cmd = [
            'pg_restore',
            '-h', db_params['host'],
            '-p', str(db_params['port']),
            '-U', db_params['user'],
            '-d', db_params['dbname'],
            '-c',  # Clean (drop) database objects before recreating
            '-v',  # Verbose mode
            backup_file
        ]
        
        # Set PGPASSWORD environment variable
        env = os.environ.copy()
        env['PGPASSWORD'] = db_params['password']
        
        print("Starting database restore...")
        result = subprocess.run(cmd, env=env, capture_output=True, text=True)
        
        if result.returncode == 0:
            print("Database restore completed successfully!")
        else:
            print(f"Error during restore: {result.stderr}")
            
            # If pg_restore fails, try psql as fallback
            print("Attempting fallback with psql...")
            psql_cmd = [
                'psql',
                '-h', db_params['host'],
                '-p', str(db_params['port']),
                '-U', db_params['user'],
                '-d', db_params['dbname'],
                '-f', backup_file
            ]
            
            psql_result = subprocess.run(psql_cmd, env=env, capture_output=True, text=True)
            
            if psql_result.returncode == 0:
                print("Database restore completed successfully using psql!")
            else:
                print(f"Error during psql restore: {psql_result.stderr}")
                print("\nPlease ensure PostgreSQL tools (pg_restore, psql) are installed and in your PATH.")
                print("You can download them from: https://www.postgresql.org/download/")
                print("Make sure to select 'Command Line Tools' during installation.")
                print("After installation, add the PostgreSQL bin directory to your PATH and restart your terminal.")
                
    except Exception as e:
        print(f"Error: {str(e)}")

if __name__ == "__main__":
    import_database() 