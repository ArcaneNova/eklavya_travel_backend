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
    DB          *sql.DB
    MongoClient *mongo.Client
    MongoDB     *mongo.Database
)

// GetDB returns the global database connection
func GetDB() *sql.DB {
    return DB
}

func LoadEnv() error {
    file, err := os.Open(".env")
    if err != nil {
        return nil // Not an error if .env doesn't exist
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        if equal := strings.Index(line, "="); equal >= 0 {
            if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
                value := ""
                if len(line) > equal {
                    value = strings.TrimSpace(line[equal+1:])
                }
                os.Setenv(key, value)
            }
        }
    }

    return scanner.Err()
}

func InitDBWithRetry(maxRetries int) error {
    var err error
    for i := 0; i < maxRetries; i++ {
        err = InitDB()
        if err == nil {
            return nil
        }
        log.Printf("Failed to initialize DB (attempt %d/%d): %v", i+1, maxRetries, err)
        time.Sleep(time.Second * time.Duration(i+1))
    }
    return err
}

func InitDB() error {
    if err := LoadEnv(); err != nil {
        return fmt.Errorf("error loading env: %v", err)
    }

    if err := InitPostgreSQL(); err != nil {
        return fmt.Errorf("error initializing PostgreSQL: %v", err)
    }

    if err := Connect(); err != nil {
        return fmt.Errorf("error connecting to MongoDB: %v", err)
    }

    return nil
}

func ConnectWithRetry(maxRetries int) error {
    var err error
    for i := 0; i < maxRetries; i++ {
        err = Connect()
        if err == nil {
            return nil
        }
        log.Printf("Failed to connect to MongoDB (attempt %d/%d): %v", i+1, maxRetries, err)
        time.Sleep(time.Second * time.Duration(i+1))
    }
    return err
}

func connectMongo(uri string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
    defer cancel()

    clientOptions := options.Client().
        ApplyURI(uri).
        SetWriteConcern(writeconcern.New(writeconcern.WMajority())).
        SetReadConcern(readconcern.Majority()).
        SetReadPreference(readpref.Primary())

    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        return fmt.Errorf("failed to create MongoDB client: %v", err)
    }

    if err = client.Ping(ctx, nil); err != nil {
        return fmt.Errorf("failed to ping MongoDB: %v", err)
    }

    MongoClient = client
    MongoDB = client.Database(getMongoDBName())

    return createIndexes(ctx)
}

func createIndexes(ctx context.Context) error {
    // Create indexes for village collection
    villageIndexes := []mongo.IndexModel{
        {
            Keys: bson.D{
                {Key: "state", Value: 1},
                {Key: "district", Value: 1},
                {Key: "subdistrict", Value: 1},
            },
        },
        {
            Keys: bson.D{
                {Key: "village_name", Value: "text"},
            },
        },
    }

    // Create indexes for bank collection
    bankIndexes := []mongo.IndexModel{
        {
            Keys: bson.D{
                {Key: "ifsc", Value: 1},
            },
            Options: options.Index().SetUnique(true),
        },
        {
            Keys: bson.D{
                {Key: "bank_name", Value: 1},
                {Key: "branch", Value: 1},
            },
        },
    }

    // Create indexes for pincode collection
    pincodeIndexes := []mongo.IndexModel{
        {
            Keys: bson.D{
                {Key: "pincode", Value: 1},
            },
            Options: options.Index().SetUnique(true),
        },
        {
            Keys: bson.D{
                {Key: "office_name", Value: "text"},
            },
        },
    }

    // Apply indexes to collections
    collections := map[string][]mongo.IndexModel{
        "villages":  villageIndexes,
        "banks":    bankIndexes,
        "pincodes": pincodeIndexes,
    }

    for collection, indexes := range collections {
        _, err := MongoDB.Collection(collection).Indexes().CreateMany(ctx, indexes)
        if err != nil {
            return fmt.Errorf("failed to create indexes for %s: %v", collection, err)
        }
    }

    return nil
}

func CheckMongoHealth() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return MongoClient.Ping(ctx, nil)
}

func CheckPostgresHealth() error {
    return DB.Ping()
}

func CloseDB() {
    if DB != nil {
        DB.Close()
    }
    if MongoClient != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        MongoClient.Disconnect(ctx)
    }
}

func WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
    tx, err := DB.BeginTx(ctx, nil)
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

func WithSession(ctx context.Context, fn func(mongo.SessionContext) error) error {
    session, err := MongoClient.StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(ctx)

    return mongo.WithSession(ctx, session, fn)
}

func InitPostgreSQL() error {
    host := os.Getenv("DB_HOST")
    port := os.Getenv("DB_PORT")
    user := os.Getenv("DB_USER")
    password := os.Getenv("DB_PASSWORD")
    dbname := os.Getenv("DB_NAME")
    sslMode := os.Getenv("DB_SSL_MODE")

    // Build connection string with SSL configuration
    connStr := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        host, port, user, password, dbname, sslMode,
    )

    var err error
    DB, err = sql.Open("postgres", connStr)
    if err != nil {
        return fmt.Errorf("error connecting to the database: %v", err)
    }

    // Test the connection
    err = DB.Ping()
    if err != nil {
        return fmt.Errorf("error pinging the database: %v", err)
    }

    // Configure connection pool settings
    DB.SetMaxOpenConns(25)
    DB.SetMaxIdleConns(5)
    DB.SetConnMaxLifetime(5 * time.Minute)

    return nil
}

func RefreshMaterializedViews() {
    ctx := context.Background()
    views := []string{
        "village_stats",
        "bank_stats",
        "pincode_stats",
    }

    for _, view := range views {
        _, err := DB.ExecContext(ctx, fmt.Sprintf("REFRESH MATERIALIZED VIEW %s", view))
        if err != nil {
            log.Printf("Error refreshing materialized view %s: %v", view, err)
        }
    }
}

func Connect() error {
    uri := getMongoURI()
    return connectMongo(uri)
}