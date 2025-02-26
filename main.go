package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    "village_site/config"
    "village_site/handlers"
    "github.com/gorilla/mux"
)

func main() {
    // Initialize configuration and database connections
    if err := config.InitDBWithRetry(5); err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer config.CloseDB()

    // Initialize cache
    config.InitCache()

    // Create router and set up middleware
    router := mux.NewRouter()
    router.Use(corsMiddleware)
    router.Use(loggingMiddleware)

    // Health check endpoint
    router.HandleFunc("/health", healthCheck).Methods("GET")

    // API routes
    apiRouter := router.PathPrefix("/api/v1").Subrouter()

    // Village routes
    villageRouter := apiRouter.PathPrefix("/village").Subrouter()
    villageRouter.HandleFunc("", handlers.ListVillages).Methods("GET")
    villageRouter.HandleFunc("/search", handlers.SearchVillages).Methods("GET")
    villageRouter.HandleFunc("/details/{id}", handlers.GetVillageDetails).Methods("GET")
    villageRouter.HandleFunc("/nearby", handlers.GetNearbyVillages).Methods("GET")
    villageRouter.HandleFunc("/stats", handlers.GetVillageStats).Methods("GET")
    villageRouter.HandleFunc("/states", handlers.GetStates).Methods("GET")

    // Bank routes
    bankRouter := apiRouter.PathPrefix("/bank").Subrouter()
    bankRouter.HandleFunc("/ifsc/{code}", handlers.GetBankByIFSC).Methods("GET")
    bankRouter.HandleFunc("/search", handlers.SearchBanks).Methods("GET")
    bankRouter.HandleFunc("/branches", handlers.GetBankBranches).Methods("GET")
    bankRouter.HandleFunc("/stats", handlers.GetBankStats).Methods("GET")

    // PIN code routes
    pincodeRouter := apiRouter.PathPrefix("/pincode").Subrouter()
    pincodeRouter.HandleFunc("/{code}", handlers.GetPinCodeDetails).Methods("GET")
    pincodeRouter.HandleFunc("/search", handlers.SearchPinCodes).Methods("GET")
    pincodeRouter.HandleFunc("/post-office", handlers.GetPostOffices).Methods("GET")
    pincodeRouter.HandleFunc("/stats", handlers.GetPinCodeStats).Methods("GET")

    // Start server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    srv := &http.Server{
        Addr:         ":" + port,
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Start server in a goroutine
    go func() {
        log.Printf("Server starting on port %s", port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Server shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }

    log.Println("Server exited properly")
}

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, Origin")
        w.Header().Set("Access-Control-Expose-Headers", "Authorization")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf(
            "%s %s %s %s",
            r.Method,
            r.RequestURI,
            r.RemoteAddr,
            time.Since(start),
        )
    })
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
    status := map[string]string{
        "status": "healthy",
        "mongo":  "up",
        "pg":     "up",
    }

    if err := config.CheckMongoHealth(); err != nil {
        status["mongo"] = "down"
        status["status"] = "degraded"
    }

    if err := config.CheckPostgresHealth(); err != nil {
        status["pg"] = "down"
        status["status"] = "degraded"
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(status)
}