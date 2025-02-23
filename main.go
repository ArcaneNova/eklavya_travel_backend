package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"
    "github.com/gorilla/mux"
    "github.com/rs/cors"
    _ "github.com/lib/pq"
    "village_site/config"
    "village_site/handlers"
)

func main() {
    startTime := time.Now()
    log.Printf("Starting server initialization at %s", startTime.Format(time.RFC3339))

    // Initialize databases concurrently
    var wg sync.WaitGroup
    errChan := make(chan error, 2)

    // Initialize PostgreSQL
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := config.InitDBWithRetry(5); err != nil {
            errChan <- fmt.Errorf("failed to initialize PostgreSQL: %v", err)
            return
        }
        log.Println("Successfully connected to PostgreSQL database")
    }()

    // Initialize MongoDB
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := config.ConnectWithRetry(5); err != nil {
            errChan <- fmt.Errorf("failed to initialize MongoDB: %v", err)
            return
        }
        log.Println("Successfully connected to MongoDB database")
    }()

    // Wait for database initialization
    wg.Wait()
    close(errChan)

    // Check for initialization errors
    for err := range errChan {
        log.Fatalf("Initialization error: %v", err)
    }

    // Initialize train system in background
    trainInitChan := make(chan error, 1)
    go func() {
        if err := handlers.InitializeTrainSystem(); err != nil {
            trainInitChan <- err
            return
        }
        trainInitChan <- nil
    }()

    // Set up HTTP server
    router := mux.NewRouter()
    api := router.PathPrefix("/api/v1").Subrouter()
    registerRoutes(api)

    // Configure CORS
    corsHandler := cors.New(cors.Options{
        AllowedOrigins: []string{"http://localhost:3000", "http://localhost:3001", "https://village2025.com"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"Content-Type", "Authorization"},
        MaxAge: 86400, // 24 hours
    })

    // Configure server
    srv := &http.Server{
        Addr:         getPort(),
        Handler:      corsHandler.Handler(router),
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Start server
    go func() {
        log.Printf("Server initialization completed in %v", time.Since(startTime))
        log.Printf("Server starting on port %s", srv.Addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed to start: %v", err)
        }
    }()

    // Wait for train system initialization
    if err := <-trainInitChan; err != nil {
        log.Printf("Warning: Train system initialization failed: %v", err)
        log.Println("Server will continue running with limited train functionality")
    }

    // Handle graceful shutdown
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop

    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Printf("Error during server shutdown: %v", err)
    }
}

func registerRoutes(api *mux.Router) {
    // Location routes
    api.HandleFunc("/locations", handlers.GetLocations).Methods("POST")
    
    // Village routes
    api.HandleFunc("/village-details", handlers.GetVillageDetails).Methods("POST")
    
    // Census routes
    api.HandleFunc("/census", handlers.GetCensusDetails).Methods("POST")
    
    // Train routes
    api.HandleFunc("/trains/suggest", handlers.GetTrainSuggestionsr).Methods("GET") 
    api.HandleFunc("/trains/{train_number}", handlers.GetTrainDetails).Methods("GET")
    api.HandleFunc("/trains/by-station", handlers.GetTrainsByStation).Methods("POST")
    api.HandleFunc("/trains/between-stations", handlers.GetTrainsBetweenStations).Methods("POST")
    api.HandleFunc("/trains/stations/suggest", handlers.GetStationSuggestionsr).Methods("GET")
    
    // Mandal routes
    api.HandleFunc("/mandal", handlers.GetMandalDetails).Methods("POST")
    api.HandleFunc("/mandal/distance", handlers.GetMandalDistance).Methods("POST")
    api.HandleFunc("/mandal/districts/suggest", handlers.GetDistrictSuggestions).Methods("GET")
    api.HandleFunc("/mandal/subdistricts/suggest", handlers.GetSubdistrictSuggestions).Methods("GET")
    
    // Bus routes
    api.HandleFunc("/bus/routes", handlers.GetCityRoutes).Methods("POST")
    api.HandleFunc("/bus/route", handlers.GetBusRoute).Methods("POST")
    api.HandleFunc("/bus/stops", handlers.GetCityStops).Methods("POST")
    api.HandleFunc("/bus/find-route", handlers.FindBusRoute).Methods("POST")
    api.HandleFunc("/bus/stops/suggest", handlers.GetBusStopSuggestions).Methods("GET")
    api.HandleFunc("/bus/cities/suggest", handlers.GetCitySuggestions).Methods("GET")
    api.HandleFunc("/bus/cities", handlers.GetAllCities).Methods("GET")

    // Company routes
    api.HandleFunc("/company", handlers.GetCompany).Methods("GET")
    api.HandleFunc("/companies", handlers.ListCompanies).Methods("GET")
    api.HandleFunc("/companies/search", handlers.SearchCompanies).Methods("GET")

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
    api.HandleFunc("/sitemaps/bus-routes", handlers.GetBusRoutesSitemap).Methods("GET")
    api.HandleFunc("/sitemaps/train-routes", handlers.GetTrainRoutesSitemap).Methods("GET")
    api.HandleFunc("/sitemaps/banks", handlers.GetBanksSitemap).Methods("GET")
    api.HandleFunc("/sitemaps/companies", handlers.GetCompaniesSitemap).Methods("GET")
    api.HandleFunc("/sitemaps/villages", handlers.GetVillagesSitemap).Methods("GET")
    api.HandleFunc("/sitemaps/mandals", handlers.GetMandalsSitemap).Methods("GET")
    api.HandleFunc("/sitemaps/pincodes", handlers.GetPincodesSitemap).Methods("GET")
    api.HandleFunc("/sitemaps/distances", handlers.GetDistancesSitemap).Methods("GET")
}

func getPort() string {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    return ":" + port
}