import os
import sys
from pathlib import Path
import psycopg2
from dotenv import load_dotenv

# Load environment variables from .env file
env_path = Path(__file__).parent.parent / '.env'
load_dotenv(env_path)

# Get database connection parameters from environment variables
DB_HOST = os.getenv('DB_HOST')
DB_PORT = os.getenv('DB_PORT')
DB_USER = os.getenv('DB_USER')
DB_PASSWORD = os.getenv('DB_PASSWORD')
DB_NAME = os.getenv('DB_NAME')

print("Database connection parameters:")
print(f"Host: {DB_HOST}")
print(f"Port: {DB_PORT}")
print(f"Database: {DB_NAME}")
print(f"User: {DB_USER}")

# Find the backup file in the current directory first, then try backup directory
script_dir = Path(__file__).parent
backup_files = list(script_dir.glob('*.sql'))

if not backup_files:
    # If no files in current directory, try backup directory
    backup_dir = script_dir.parent / 'backup'
    backup_files = list(backup_dir.glob('*.sql'))

if not backup_files:
    print("Error: No .sql files found in current directory or backup directory")
    sys.exit(1)

# Use the first .sql file found
BACKUP_FILE = str(backup_files[0])
print(f"Using backup file: {BACKUP_FILE}")

def main():
    try:
        # Connect to the database
        print("Connecting to database...")
        conn = psycopg2.connect(
            host=DB_HOST,
            port=DB_PORT,
            user=DB_USER,
            password=DB_PASSWORD,
            database=DB_NAME,
            sslmode='require'  # Required for Aiven
        )
        conn.autocommit = True
        cur = conn.cursor()
        
        print("Successfully connected to database")

        # Drop all existing tables in public schema
        print("Dropping existing tables...")
        try:
            cur.execute("""
                DO $$ 
                DECLARE 
                    r RECORD;
                BEGIN
                    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
                        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
                    END LOOP;
                END $$;
            """)
            print("Successfully dropped existing tables")
        except psycopg2.Error as e:
            print(f"Warning: Could not drop all tables: {e}")

        # Read and process the backup file
        print(f"Reading backup file: {BACKUP_FILE}")
        with open(BACKUP_FILE, 'rb') as f:
            # Check if it's a custom format backup
            header = f.read(5)
            if header.startswith(b'PGDMP'):
                print("This is a PostgreSQL custom format backup file.")
                print("To restore this backup, you need to:")
                print("1. Use pgAdmin's restore feature:")
                print(f"   - Host: {DB_HOST}")
                print(f"   - Port: {DB_PORT}")
                print(f"   - Database: {DB_NAME}")
                print(f"   - Username: {DB_USER}")
                print("   - SSL Mode: Require")
                print("   - Format: Custom or tar")
                print(f"   - Filename: {BACKUP_FILE}")
                print("2. Or use pg_restore command:")
                print(f"   pg_restore -h {DB_HOST} -p {DB_PORT} -U {DB_USER} -d {DB_NAME} --clean --if-exists -v {BACKUP_FILE}")
                sys.exit(1)

            # If not custom format, try to restore as plain SQL
            f.seek(0)
            try:
                content = f.read().decode('utf-8')
            except UnicodeDecodeError:
                print("Error: Backup file is not in a readable format")
                sys.exit(1)

            print("Executing SQL statements...")
            cur.execute(content)
            print("Database restore completed successfully!")

    except psycopg2.Error as e:
        print(f"Database error: {e}")
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)
    finally:
        if 'cur' in locals():
            cur.close()
        if 'conn' in locals():
            conn.close()

if __name__ == "__main__":
    main() 