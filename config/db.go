package config

import (
    "bufio"
    "context"
    "database/sql"
    "fmt"
    "log"
    "os"
    "strings"
    "time"
    _ "github.com/lib/pq"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/mongo/writeconcern"
    "go.mongodb.org/mongo-driver/mongo/readconcern"
    "go.mongodb.org/mongo-driver/mongo/readpref"
    "go.mongodb.org/mongo-driver/bson"
)

var (
    DB *sql.DB
    MongoDB *mongo.Database
    MongoClient *mongo.Client
)

const (
    maxRetries = 5
    retryDelay = 5 * time.Second
)

// LoadEnv loads environment variables from .env file
func LoadEnv() error {
    // Try multiple possible locations for .env file
    possiblePaths := []string{
        ".env",                    // Current directory
        "../.env",                 // Parent directory
        "../../.env",             // Two levels up
        os.Getenv("VILLAGE_ENV"), // Environment-specified path
    }

    var loadedFile string
    var err error

    for _, path := range possiblePaths {
        if path == "" {
            continue
        }
        if _, err := os.Stat(path); err == nil {
            loadedFile = path
            log.Printf("Found .env file at: %s", path)
            break
        }
    }

    if loadedFile == "" {
        // If no .env file found, check if MONGO_URI is already set in environment
        if uri := os.Getenv("MONGO_URI"); uri != "" {
            return nil // MONGO_URI already set, no need for .env
        }
        return fmt.Errorf("no .env file found and MONGO_URI not set in environment")
    }

    file, err := os.Open(loadedFile)
    if err != nil {
        return fmt.Errorf("error opening .env file: %v", err)
    }
    defer file.Close()

    log.Printf("Loading environment variables from %s", loadedFile)
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
            continue
        }
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
        key := strings.TrimSpace(parts[0])
        value := strings.TrimSpace(parts[1])
        // Remove quotes if present
        value = strings.Trim(value, `"'`)
        os.Setenv(key, value)
        if !strings.Contains(strings.ToLower(key), "password") && !strings.Contains(strings.ToLower(key), "secret") {
            log.Printf("Set environment variable: %s", key)
        }
    }

    // Verify some key environment variables were loaded
    if os.Getenv("DB_HOST") != "" {
        log.Printf("Successfully loaded database configuration")
    }
    
    return scanner.Err()
}

// InitDBWithRetry attempts to connect to PostgreSQL with retries
func InitDBWithRetry(maxRetries int) error {
    var err error
    for i := 0; i < maxRetries; i++ {
        err = InitDB()
        if err == nil {
            return nil
        }
        log.Printf("Failed to connect to PostgreSQL (attempt %d/%d): %v", i+1, maxRetries, err)
        time.Sleep(5 * time.Second)
    }
    return fmt.Errorf("failed to connect to PostgreSQL after %d attempts: %v", maxRetries, err)
}

func InitDB() error {
    dbParams := map[string]string{
        "dbname":   os.Getenv("DB_NAME"),
        "user":     os.Getenv("DB_USER"),
        "password": os.Getenv("DB_PASSWORD"),
        "host":     os.Getenv("DB_HOST"),
        "port":     os.Getenv("DB_PORT"),
        "sslmode":  os.Getenv("DB_SSL_MODE"),
    }

    // Log current environment variables (excluding sensitive data)
    log.Printf("DB Host: %s", dbParams["host"])
    log.Printf("DB Port: %s", dbParams["port"])
    log.Printf("DB Name: %s", dbParams["dbname"])
    log.Printf("DB User: %s", dbParams["user"])
    log.Printf("SSL Mode: %s", dbParams["sslmode"])

    // Use default values if environment variables are not set
    if dbParams["dbname"] == "" {
        dbParams["dbname"] = "defaultdb"
    }
    if dbParams["user"] == "" {
        dbParams["user"] = "avnadmin"
    }
    if dbParams["host"] == "" {
        dbParams["host"] = "localhost"
    }
    if dbParams["port"] == "" {
        dbParams["port"] = "5432"
    }
    if dbParams["sslmode"] == "" {
        // Default to require SSL for Aiven
        if strings.Contains(dbParams["host"], "aivencloud.com") {
            dbParams["sslmode"] = "require"
        } else {
            dbParams["sslmode"] = "disable"
        }
    }

    // Build connection string
    connStr := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        dbParams["host"], dbParams["port"], dbParams["user"], 
        dbParams["password"], dbParams["dbname"], dbParams["sslmode"])

    log.Printf("Connecting to PostgreSQL with sslmode=%s", dbParams["sslmode"])

    var err error
    DB, err = sql.Open("postgres", connStr)
    if err != nil {
        return fmt.Errorf("error opening PostgreSQL database: %v", err)
    }

    // Set connection pool settings
    DB.SetMaxOpenConns(25)  // Reduced from 100 to prevent overwhelming the connection
    DB.SetMaxIdleConns(5)   // Reduced from 25 to prevent idle connections
    DB.SetConnMaxLifetime(5 * time.Minute)  // Reduced from 30 to recycle connections more frequently

    // Verify connection with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err = DB.PingContext(ctx); err != nil {
        return fmt.Errorf("error connecting to PostgreSQL database: %v", err)
    }

    log.Printf("Successfully connected to PostgreSQL database: %s", dbParams["dbname"])

    // Verify villages table exists
    var tableExists bool
    err = DB.QueryRowContext(ctx, `
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_name = 'villages'
        )`).Scan(&tableExists)
    
    if err != nil {
        return fmt.Errorf("error checking villages table: %v", err)
    }

    if !tableExists {
        return fmt.Errorf("villages table does not exist in the database")
    }

    log.Printf("Verified villages table exists")
    return nil
}

// ConnectWithRetry attempts to connect to MongoDB with retries
func ConnectWithRetry(maxRetries int) error {
    // Load environment variables from .env file
    if err := LoadEnv(); err != nil {
        log.Printf("Warning: Could not load .env file: %v", err)
    }

    mongoURI := os.Getenv("MONGO_URI")
    if mongoURI == "" {
        return fmt.Errorf("MONGO_URI environment variable is required but not set")
    }

    var err error
    for i := 0; i < maxRetries; i++ {
        err = connectMongo(mongoURI)
        if err == nil {
            return nil
        }
        log.Printf("Failed to connect to MongoDB (attempt %d/%d): %v", i+1, maxRetries, err)
        time.Sleep(5 * time.Second)
    }
    return fmt.Errorf("failed to connect after %d attempts: %v", maxRetries, err)
}

// connectMongo initializes the MongoDB connection
func connectMongo(uri string) error {
    clientOptions := options.Client().ApplyURI(uri).
        SetMaxPoolSize(100).
        SetMinPoolSize(20).
        SetMaxConnecting(50).
        SetConnectTimeout(10*time.Second).
        SetServerSelectionTimeout(10*time.Second).
        SetSocketTimeout(30*time.Second).
        SetRetryWrites(true).
        SetRetryReads(true).
        SetMaxConnIdleTime(60*time.Minute).
        SetWriteConcern(writeconcern.New(writeconcern.WMajority())).
        SetReadConcern(readconcern.Majority()).
        SetReadPreference(readpref.Primary())

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    var err error
    MongoClient, err = mongo.Connect(ctx, clientOptions)
    if err != nil {
        return fmt.Errorf("error connecting to MongoDB: %v", err)
    }

    if err = MongoClient.Ping(ctx, nil); err != nil {
        return fmt.Errorf("error pinging MongoDB: %v", err)
    }

    dbName := os.Getenv("MONGO_DB_NAME")
    if dbName == "" {
        dbName = "train_database"
    }
    MongoDB = MongoClient.Database(dbName)
    log.Printf("Successfully connected to MongoDB database: %s", dbName)

    if err := createIndexes(ctx); err != nil {
        return fmt.Errorf("error creating indexes: %v", err)
    }

    return nil
}

func createIndexes(ctx context.Context) error {
    // Create train collection indexes
    trainCollection := MongoDB.Collection("trains")
    trainIndexes := []mongo.IndexModel{
        {
            Keys: bson.D{
                {Key: "train_number", Value: 1},
            },
            Options: options.Index().SetUnique(true).SetName("train_number_idx"),
        },
        {
            Keys: bson.D{
                {Key: "schedule_table.station", Value: 1},
            },
            Options: options.Index().SetName("station_schedule_idx"),
        },
        {
            Keys: bson.D{
                {Key: "title", Value: "text"},
                {Key: "schedule_table.station", Value: "text"},
            },
            Options: options.Index().SetName("train_text_search_idx"),
        },
    }

    // Drop existing indexes before creating new ones
    if _, err := trainCollection.Indexes().DropAll(ctx); err != nil {
        log.Printf("Warning: Failed to drop existing train indexes: %v", err)
    }
    
    _, err := trainCollection.Indexes().CreateMany(ctx, trainIndexes)
    if err != nil {
        return fmt.Errorf("error creating train indexes: %v", err)
    }
    log.Printf("Successfully created train indexes")

    // Create bus routes collection indexes
    busCollection := MongoDB.Collection("bus_routes")
    busIndexes := []mongo.IndexModel{
        {
            Keys: bson.D{
                {Key: "city", Value: 1},
                {Key: "route_name", Value: 1},
            },
            Options: options.Index().SetName("city_route_idx"),
        },
        {
            Keys: bson.D{
                {Key: "route.stop_name", Value: 1},
            },
            Options: options.Index().SetName("stop_name_idx"),
        },
        {
            Keys: bson.D{
                {Key: "city", Value: "text"},
                {Key: "route.stop_name", Value: "text"},
            },
            Options: options.Index().SetName("bus_text_search_idx"),
        },
    }
    
    // Drop existing indexes before creating new ones
    if _, err := busCollection.Indexes().DropAll(ctx); err != nil {
        log.Printf("Warning: Failed to drop existing bus route indexes: %v", err)
    }
    
    _, err = busCollection.Indexes().CreateMany(ctx, busIndexes)
    if err != nil {
        return fmt.Errorf("error creating bus route indexes: %v", err)
    }
    log.Printf("Successfully created bus route indexes")

    return nil
}

// Health check functions
func CheckMongoHealth() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := MongoClient.Ping(ctx, nil); err != nil {
        return fmt.Errorf("MongoDB health check failed: %v", err)
    }
    return nil
}

func CheckPostgresHealth() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := DB.PingContext(ctx); err != nil {
        return fmt.Errorf("PostgreSQL health check failed: %v", err)
    }
    return nil
}

// Graceful shutdown
func CloseDB() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if DB != nil {
        if err := DB.Close(); err != nil {
            log.Printf("Error closing PostgreSQL connection: %v", err)
        }
    }

    if MongoClient != nil {
        if err := MongoClient.Disconnect(ctx); err != nil {
            log.Printf("Error closing MongoDB connection: %v", err)
        }
    }
}

// Transaction support for PostgreSQL
func WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
    tx, err := DB.BeginTx(ctx, &sql.TxOptions{
        Isolation: sql.LevelSerializable,
    })
    if err != nil {
        return err
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)
        }
    }()

    if err := fn(tx); err != nil {
        tx.Rollback()
        return err
    }

    return tx.Commit()
}

// MongoDB session handling
func WithSession(ctx context.Context, fn func(mongo.SessionContext) error) error {
    session, err := MongoClient.StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(ctx)

    return mongo.WithSession(ctx, session, fn)
}

func InitPostgreSQL() error {
    // Create PostgreSQL indexes
    indexes := []string{
        `CREATE INDEX IF NOT EXISTS idx_villages_location ON villages (state_name, district_name, subdistrict_name)`,
        `CREATE INDEX IF NOT EXISTS idx_mandals_location ON mandals (district, subdistrict)`,
        `CREATE INDEX IF NOT EXISTS idx_pin_details_location ON pin_details (state, district, pincode, officename)`,
        `CREATE INDEX IF NOT EXISTS idx_ifsc_details_location ON ifsc_details (bank, state, district, branch_city, ifsc)`,
    }

    for _, idx := range indexes {
        _, err := DB.Exec(idx)
        if err != nil {
            return fmt.Errorf("failed to create index: %v", err)
        }
    }

    // Create materialized views for frequently accessed data
    views := []string{
        `CREATE MATERIALIZED VIEW IF NOT EXISTS mv_district_counts AS
         SELECT district, COUNT(*) as village_count
         FROM villages
         GROUP BY district`,
        
        `CREATE MATERIALIZED VIEW IF NOT EXISTS mv_state_counts AS
         SELECT state_name, COUNT(*) as village_count
         FROM villages
         GROUP BY state_name`,
    }

    for _, view := range views {
        _, err := DB.Exec(view)
        if err != nil {
            return fmt.Errorf("failed to create materialized view: %v", err)
        }
    }

    return nil
}

// Add function to refresh materialized views periodically
func RefreshMaterializedViews() {
    ticker := time.NewTicker(24 * time.Hour)
    go func() {
        for range ticker.C {
            _, err := DB.Exec(`REFRESH MATERIALIZED VIEW CONCURRENTLY mv_district_counts`)
            if err != nil {
                log.Printf("Error refreshing district counts view: %v", err)
            }
            _, err = DB.Exec(`REFRESH MATERIALIZED VIEW CONCURRENTLY mv_state_counts`)
            if err != nil {
                log.Printf("Error refreshing state counts view: %v", err)
            }
        }
    }()
}

func Connect() error {
    // Load MongoDB URI from environment variable
    mongoURI := os.Getenv("MONGODB_URI")
    if mongoURI == "" {
        return fmt.Errorf("MONGODB_URI environment variable not set")
    }

    // Configure client options with optimized settings
    clientOptions := options.Client().ApplyURI(mongoURI)
    clientOptions.SetMaxPoolSize(100)
    clientOptions.SetMinPoolSize(10)
    clientOptions.SetMaxConnecting(20)
    clientOptions.SetConnectTimeout(2 * time.Minute)
    clientOptions.SetSocketTimeout(3 * time.Minute)
    clientOptions.SetServerSelectionTimeout(2 * time.Minute)
    clientOptions.SetRetryWrites(true)
    clientOptions.SetRetryReads(true)

    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    // Connect to MongoDB
    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        return fmt.Errorf("failed to connect to MongoDB: %v", err)
    }

    // Ping the database to verify connection
    if err = client.Ping(ctx, nil); err != nil {
        return fmt.Errorf("failed to ping MongoDB: %v", err)
    }

    // Set global MongoDB client
    MongoDB = client.Database(os.Getenv("MONGODB_DATABASE"))

    // Create indexes
    if err := createIndexes(ctx); err != nil {
        log.Printf("Warning: Failed to create indexes: %v", err)
    }

    return nil
}