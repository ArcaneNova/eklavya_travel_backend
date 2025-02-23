package config

import (
    "os"
    "strconv"
)

// Database configuration
func getPostgresConnString() string {
    host := getEnvWithDefault("DB_HOST", "localhost")
    port := getEnvWithDefault("DB_PORT", "5432")
    user := getEnvWithDefault("DB_USER", "postgres")
    password := getEnvWithDefault("DB_PASSWORD", "1234")
    dbname := getEnvWithDefault("DB_NAME", "indiavillage")

    return "host=" + host + " port=" + port + " user=" + user + 
           " password=" + password + " dbname=" + dbname + " sslmode=disable"
}

func getMongoURI() string {
    uri := getEnvWithDefault("MONGO_URI", "mongodb://localhost:27017")
    return uri
}

func getMongoDBName() string {
    return getEnvWithDefault("MONGO_DB_NAME", "indiavillage")
}

// Helper functions
func getEnvWithDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        if boolValue, err := strconv.ParseBool(value); err == nil {
            return boolValue
        }
    }
    return defaultValue
} 