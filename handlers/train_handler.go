package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "math"
    "net/http"
    "sort"
    "strings"
    "sync"
    "time"
    "runtime"
    "crypto/rand"
    "encoding/hex"
    "strconv"
    "village_site/config"
    "village_site/models"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "github.com/gorilla/mux"
)

// Memory-efficient route info structure
type RouteInfo struct {
    TrainNumber string
    TrainName   string
    TrainType   string
}

// Global variables for station graph
var (
    stationGraph = make(map[string]map[string][]RouteInfo)
    graphMutex   sync.RWMutex
)

const (
    BATCH_SIZE = 100
    TOTAL_TRAINS = 16772
)

func InitializeTrainSystem() error {
    log.Println("Starting train system initialization...")
    
    // Use a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()

    // Create an aggregation pipeline to stream trains in batches
    pipeline := mongo.Pipeline{
        {{"$project", bson.D{
            {"train_number", 1},
            {"name", 1},
            {"type", 1},
            {"schedule", 1},
        }}},
    }

    // Set batch size in options
    opts := options.Aggregate().SetBatchSize(BATCH_SIZE)
    cursor, err := config.MongoDB.Collection("trains").Aggregate(ctx, pipeline, opts)
    if err != nil {
        return fmt.Errorf("failed to create cursor: %v", err)
    }
    defer cursor.Close(ctx)

    var processedCount int
    var batch []models.Train

    for cursor.Next(ctx) {
        var train models.Train
        if err := cursor.Decode(&train); err != nil {
            log.Printf("Warning: Error decoding train: %v", err)
            continue
        }

        batch = append(batch, train)
        processedCount++

        // Process batch when it reaches BATCH_SIZE or at the end
        if len(batch) >= BATCH_SIZE || !cursor.Next(ctx) {
            if err := processBatch(batch); err != nil {
                log.Printf("Warning: Error processing batch: %v", err)
            }
            
            // Clear batch after processing
            batch = batch[:0]

            // Log progress
            progress := float64(processedCount) / float64(TOTAL_TRAINS) * 100
            log.Printf("Processed %d/%d trains (%.1f%%)", processedCount, TOTAL_TRAINS, progress)

            // Force garbage collection after each batch
            runtime.GC()
        }
    }

    if err := cursor.Err(); err != nil {
        return fmt.Errorf("cursor error: %v", err)
    }

    log.Println("Train system initialization completed")
    return nil
}

func processBatch(trains []models.Train) error {
    // Process each train in the batch
    for _, train := range trains {
        if err := processTrainRoutes(&train); err != nil {
            log.Printf("Warning: Error processing train %d: %v", train.TrainNumber, err)
            continue
        }
    }
    return nil
}

func processTrainRoutes(train *models.Train) error {
    // Create route nodes and edges for this train
    for i := 0; i < len(train.Schedule)-1; i++ {
        fromStation := train.Schedule[i].Station
        toStation := train.Schedule[i+1].Station

        // Add stations to graph if they don't exist
        addStationIfNotExists(fromStation)
        addStationIfNotExists(toStation)

        // Add route edge
        addRouteEdge(fromStation, toStation, train)
    }
    return nil
}

func addStationIfNotExists(station string) {
    graphMutex.Lock()
    defer graphMutex.Unlock()

    if _, exists := stationGraph[station]; !exists {
        stationGraph[station] = make(map[string][]RouteInfo)
    }
}

func addRouteEdge(from, to string, train *models.Train) {
    graphMutex.Lock()
    defer graphMutex.Unlock()

    route := RouteInfo{
        TrainNumber: fmt.Sprintf("%d", train.TrainNumber),
        TrainName:   train.Title,
        TrainType:   train.Type,
    }

    // Add route to graph
    stationGraph[from][to] = append(stationGraph[from][to], route)
}

// Handler functions
func GetTrainSuggestionsr(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" {
        http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
        return
    }

    ctx := r.Context()
    filter := bson.M{
        "$or": []bson.M{
            {"train_number": bson.M{"$regex": query, "$options": "i"}},
            {"title": bson.M{"$regex": query, "$options": "i"}},
        },
    }

    opts := options.Find().
        SetLimit(10).
        SetProjection(bson.M{"train_number": 1, "title": 1, "_id": 0})

    cursor, err := config.MongoDB.Collection("trains").Find(ctx, filter, opts)
    if err != nil {
        log.Printf("Error searching trains: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    var suggestions []map[string]interface{}
    for cursor.Next(ctx) {
        var train struct {
            Number int    `bson:"train_number"`
            Title  string `bson:"title"`
        }
        if err := cursor.Decode(&train); err != nil {
            continue
        }
        suggestions = append(suggestions, map[string]interface{}{
            "train_number": train.Number,
            "train_name":   train.Title,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "suggestions": suggestions,
    })
}

func GetTrainDetails(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    trainNumber := vars["train_number"]

    if trainNumber == "" {
        http.Error(w, "Train number is required", http.StatusBadRequest)
        return
    }

    ctx := r.Context()
    num, err := strconv.Atoi(trainNumber)
    if err != nil {
        http.Error(w, "Invalid train number", http.StatusBadRequest)
        return
    }

    var train models.Train
    err = config.MongoDB.Collection("trains").FindOne(ctx, bson.M{
        "train_number": num,
    }).Decode(&train)

    if err == mongo.ErrNoDocuments {
        http.Error(w, "Train not found", http.StatusNotFound)
        return
    }
    if err != nil {
        log.Printf("Error fetching train details: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(train)
}

func GetTrainsByStation(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Station string `json:"station"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }

    if req.Station == "" {
        http.Error(w, "Station code is required", http.StatusBadRequest)
        return
    }

    ctx := r.Context()
    cursor, err := config.MongoDB.Collection("trains").Find(ctx, bson.M{
        "schedule_table.station": req.Station,
    })
    if err != nil {
        log.Printf("Error fetching trains: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    var trains []models.Train
    if err := cursor.All(ctx, &trains); err != nil {
        log.Printf("Error processing trains: %v", err)
        http.Error(w, "Error processing results", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "trains": trains,
        "count":  len(trains),
    })
}

func GetTrainsBetweenStations(w http.ResponseWriter, r *http.Request) {
    var req struct {
        FromStation string `json:"from_station"`
        ToStation   string `json:"to_station"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }

    if req.FromStation == "" || req.ToStation == "" {
        http.Error(w, "Both from_station and to_station are required", http.StatusBadRequest)
        return
    }

    graphMutex.RLock()
    routes := stationGraph[req.FromStation][req.ToStation]
    graphMutex.RUnlock()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "routes": routes,
        "count":  len(routes),
    })
}

func GetStationSuggestionsr(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" {
        http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
        return
    }

    ctx := r.Context()
    filter := bson.M{
        "$or": []bson.M{
            {"code": bson.M{"$regex": query, "$options": "i"}},
            {"name": bson.M{"$regex": query, "$options": "i"}},
        },
    }

    opts := options.Find().
        SetLimit(10).
        SetProjection(bson.M{"code": 1, "name": 1, "_id": 0})

    cursor, err := config.MongoDB.Collection("stations").Find(ctx, filter, opts)
    if err != nil {
        log.Printf("Error searching stations: %v", err)
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    var suggestions []map[string]interface{}
    for cursor.Next(ctx) {
        var station struct {
            Code string `bson:"code"`
            Name string `bson:"name"`
        }
        if err := cursor.Decode(&station); err != nil {
            continue
        }
        suggestions = append(suggestions, map[string]interface{}{
            "station_code": station.Code,
            "station_name": station.Name,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "suggestions": suggestions,
    })
} 