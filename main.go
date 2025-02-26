package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "runtime"
    "syscall"
    "time"
    "github.com/gorilla/mux"
    "github.com/rs/cors"
    _ "github.com/lib/pq"
    "village_site/config"
    "village_site/handlers"
)

type HealthResponse struct {
    Status    string `json:"status"`
    DBStatus  string `json:"db_status"`
    DBDetails struct {
        Host     string `json:"host"`
        Port     string `json:"port"`
        Database string `json:"database"`
        Tables   []string `json:"tables,omitempty"`
    } `json:"db_details"`
    Error     string `json:"error,omitempty"`
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
    response := HealthResponse{
        Status: "ok",
    }

    // Check database connection
    if config.DB == nil {
        response.Status = "error"
        response.DBStatus = "not_initialized"
        response.Error = "Database connection not initialized"
    } else {
        // Try to ping the database
        err := config.DB.Ping()
        if err != nil {
            response.Status = "error"
            response.DBStatus = "connection_error"
            response.Error = fmt.Sprintf("Database ping failed: %v", err)
        } else {
            response.DBStatus = "connected"
            
            // Get database details
            response.DBDetails.Host = os.Getenv("DB_HOST")
            response.DBDetails.Port = os.Getenv("DB_PORT")
            response.DBDetails.Database = os.Getenv("DB_NAME")

            // Check for required tables
            tables := []string{"ifsc_details", "micr_details", "bank_details"}
            var existingTables []string

            for _, table := range tables {
                var exists bool
                err := config.DB.QueryRow(`
                    SELECT EXISTS (
                        SELECT FROM information_schema.tables 
                        WHERE table_name = $1
                    )`, table).Scan(&exists)
                
                if err == nil && exists {
                    existingTables = append(existingTables, table)
                }
            }
            response.DBDetails.Tables = existingTables
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// CORS middleware function
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set CORS headers
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, Origin")
        w.Header().Set("Access-Control-Expose-Headers", "Authorization")

        // Handle preflight requests
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}

func main() {
    // Set memory optimization settings from environment
    if batchSize := os.Getenv("BATCH_SIZE"); batchSize != "" {
        log.Printf("Using batch size: %s", batchSize)
    }
    if gogc := os.Getenv("GOGC"); gogc != "" {
        log.Printf("Using GOGC: %s", gogc)
    }

    startTime := time.Now()
    log.Printf("Starting server initialization at %s", startTime.Format(time.RFC3339))

    // Load environment variables first
    if err := config.LoadEnv(); err != nil {
        log.Printf("Warning: Error loading .env file: %v", err)
    }

    // Load environment variables
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
        log.Printf("No PORT environment variable found, using default: %s", port)
    }

    // Initialize PostgreSQL database with retries
    log.Println("Initializing PostgreSQL database...")
    if err := config.InitDBWithRetry(5); err != nil {
        log.Fatalf("Failed to initialize PostgreSQL: %v", err)
    }
    log.Println("PostgreSQL database initialized successfully")
    defer config.CloseDB()

    // Create router with memory-optimized settings
    r := mux.NewRouter()
    
    // CORS configuration
    corsHandler := cors.New(cors.Options{
        AllowedOrigins: []string{
            "http://localhost:3000",
            "https://eklavyatravel.com",
            "https://www.eklavyatravel.com",
        },
        AllowedMethods: []string{
            "GET", "POST", "PUT", "DELETE", "OPTIONS",
        },
        AllowedHeaders: []string{
            "Accept", "Content-Type", "Content-Length", 
            "Accept-Encoding", "Authorization", "X-CSRF-Token",
        },
        AllowCredentials: true,
        MaxAge: 86400, // Cache preflight requests for 24 hours
    })

    // Apply CORS middleware
    r.Use(corsHandler.Handler)

    // API routes
    api := r.PathPrefix("/api/v1").Subrouter()
    registerRoutes(api)
    log.Println("Routes registered successfully")

    // Health check endpoint
    api.HandleFunc("/health/detailed", healthCheck).Methods("GET")

    // Create server with optimized timeouts
    srv := &http.Server{
        Handler:           r,
        Addr:             ":" + port,
        WriteTimeout:      15 * time.Second,
        ReadTimeout:      15 * time.Second,
        IdleTimeout:      60 * time.Second,
        ReadHeaderTimeout: 5 * time.Second,
        MaxHeaderBytes:   1 << 20,
    }

    // Create error channel for server errors
    serverErrors := make(chan error, 1)

    // Start server in a goroutine
    go func() {
        log.Printf("Starting server on port %s...", port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("Server error: %v", err)
            serverErrors <- err
        }
    }()

    // Wait for server to start
    time.Sleep(1 * time.Second)
    log.Printf("Server is running at http://localhost:%s", port)
    log.Printf("Health check endpoint: http://localhost:%s/api/v1/health", port)
    log.Printf("Sitemap endpoint: http://localhost:%s/api/v1/sitemaps", port)

    // Handle graceful shutdown
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

    // Wait for shutdown signal or server error
    select {
    case <-stop:
        log.Println("Shutdown signal received")
    case err := <-serverErrors:
        log.Printf("Server error received: %v", err)
    }

    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Printf("Error during server shutdown: %v", err)
    } else {
        log.Println("Server shutdown completed successfully")
    }
    
    // Final garbage collection
    runtime.GC()
}

func registerRoutes(api *mux.Router) {
    // Location routes
    api.HandleFunc("/locations", handlers.GetLocations).Methods("POST")
    
    // Village routes
    api.HandleFunc("/village-details", handlers.GetVillageDetails).Methods("POST")
    
    // Census routes
    api.HandleFunc("/census", handlers.GetCensusDetails).Methods("POST")
    
    // Mandal routes
    api.HandleFunc("/mandal", handlers.GetMandalDetails).Methods("POST")
    api.HandleFunc("/mandal/distance", handlers.GetMandalDistance).Methods("POST")
    api.HandleFunc("/mandal/districts/suggest", handlers.GetDistrictSuggestions).Methods("GET")
    api.HandleFunc("/mandal/subdistricts/suggest", handlers.GetSubdistrictSuggestions).Methods("GET")

    // IFSC routes
    api.HandleFunc("/bank/list", handlers.GetBankList).Methods("GET")
    api.HandleFunc("/bank/states", handlers.GetBankStates).Methods("GET")
    api.HandleFunc("/bank/districts", handlers.GetBankDistricts).Methods("GET")
    api.HandleFunc("/bank/cities", handlers.GetBankBranchCities).Methods("GET")
    api.HandleFunc("/bank/branches", handlers.GetBankBranches).Methods("GET")
    api.HandleFunc("/bank/ifsc", handlers.GetIFSCDetails).Methods("GET")
    api.HandleFunc("/bank/debug-ifsc", handlers.DebugIFSCData).Methods("GET")

    // PIN code routes
    api.HandleFunc("/pincode/states", handlers.GetPinStates).Methods("GET")
    api.HandleFunc("/pincode/districts", handlers.GetPinDistricts).Methods("GET")
    api.HandleFunc("/pincode/offices", handlers.GetPostOffices).Methods("GET")
    api.HandleFunc("/pincode/details", handlers.GetPinCodeDetails).Methods("GET")

    // Health check
    api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    }).Methods("GET")

    // Sitemap routes
    api.HandleFunc("/sitemaps", handlers.GetSitemapIndex).Methods("GET")
    api.HandleFunc("/sitemaps/villages", handlers.GetVillagesSitemap).Methods("GET")
    api.HandleFunc("/sitemaps/mandals", handlers.GetMandalsSitemap).Methods("GET")
    api.HandleFunc("/sitemaps/pincodes", handlers.GetPincodesSitemap).Methods("GET")
    api.HandleFunc("/sitemaps/distances", handlers.GetDistancesSitemap).Methods("GET")
}