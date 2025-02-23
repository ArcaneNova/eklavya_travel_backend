package handlers

import (
    "encoding/json"
    "net/http"
    "context"
    "time"
    "village_site/config"
    "village_site/models"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

// GetStationSuggestions handles station search/autocomplete requests
func GetStationSuggestions(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" {
        sendErrorResponse(w, "Query parameter 'q' is required", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Search in MongoDB using text index
    filter := bson.M{
        "$or": []bson.M{
            {"code": bson.M{"$regex": query, "$options": "i"}},
            {"name": bson.M{"$regex": query, "$options": "i"}},
            {"city": bson.M{"$regex": query, "$options": "i"}},
        },
    }

    opts := options.Find().
        SetLimit(10).
        SetProjection(bson.M{
            "code": 1,
            "name": 1,
            "city": 1,
            "state": 1,
            "location": 1,
        })

    cursor, err := config.MongoDB.Collection("stations").Find(ctx, filter, opts)
    if err != nil {
        sendErrorResponse(w, "Database error", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    var stations []models.Station
    if err := cursor.All(ctx, &stations); err != nil {
        sendErrorResponse(w, "Error processing results", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "suggestions": formatStationSuggestions(stations),
        "count":      len(stations),
        "timestamp":  time.Now().Format(time.RFC3339),
    })
}

// GetStationDetails handles requests for detailed station information
func GetStationDetails(w http.ResponseWriter, r *http.Request) {
    stationCode := r.URL.Query().Get("code")
    if stationCode == "" {
        sendErrorResponse(w, "Station code is required", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var station models.Station
    err := config.MongoDB.Collection("stations").FindOne(ctx, 
        bson.M{"code": stationCode}).Decode(&station)

    if err == mongo.ErrNoDocuments {
        sendErrorResponse(w, "Station not found", http.StatusNotFound)
        return
    }
    if err != nil {
        sendErrorResponse(w, "Database error", http.StatusInternalServerError)
        return
    }

    // Get trains passing through this station
    cursor, err := config.MongoDB.Collection("trains").Find(ctx, 
        bson.M{"schedule.station": stationCode})
    if err != nil {
        sendErrorResponse(w, "Error fetching train details", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    var trains []models.Train
    if err := cursor.All(ctx, &trains); err != nil {
        sendErrorResponse(w, "Error processing train details", http.StatusInternalServerError)
        return
    }

    response := map[string]interface{}{
        "station": station,
        "trains": formatStationTrains(trains, stationCode),
        "facilities": getStationFacilities(stationCode),
        "timestamp": time.Now().Format(time.RFC3339),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// GetNearbyStations handles requests for finding stations near a location
func GetNearbyStations(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Latitude  float64 `json:"latitude"`
        Longitude float64 `json:"longitude"`
        Radius    float64 `json:"radius"` // in kilometers
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendErrorResponse(w, "Invalid request format", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Create geospatial query
    filter := bson.M{
        "location": bson.M{
            "$nearSphere": bson.M{
                "$geometry": bson.M{
                    "type": "Point",
                    "coordinates": []float64{req.Longitude, req.Latitude},
                },
                "$maxDistance": req.Radius * 1000, // convert to meters
            },
        },
    }

    cursor, err := config.MongoDB.Collection("stations").Find(ctx, filter)
    if err != nil {
        sendErrorResponse(w, "Database error", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    var stations []models.Station
    if err := cursor.All(ctx, &stations); err != nil {
        sendErrorResponse(w, "Error processing results", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "stations":   stations,
        "count":      len(stations),
        "radius_km":  req.Radius,
        "timestamp":  time.Now().Format(time.RFC3339),
    })
}

// Helper functions
func formatStationSuggestions(stations []models.Station) []map[string]interface{} {
    suggestions := make([]map[string]interface{}, len(stations))
    for i, station := range stations {
        suggestions[i] = map[string]interface{}{
            "code":     station.Code,
            "name":     station.Name,
            "city":     station.City,
            // "state":    station.State, 
            "location": station.Location,
        }
    }
    return suggestions
}

func formatStationTrains(trains []models.Train, stationCode string) []map[string]interface{} {
    formattedTrains := make([]map[string]interface{}, 0)
    for _, train := range trains {
        stopInfo := findStationStop(train.Schedule, stationCode)
        if stopInfo != nil {
            formattedTrains = append(formattedTrains, map[string]interface{}{
                "train_number": train.TrainNumber,
                "train_name":   train.Name,
                "type":         train.Type,
                "arrival":      stopInfo.Arrival,
                "departure":    stopInfo.Departure,
                "platform":     stopInfo.Platform,
                "halt":         stopInfo.Halt,
            })
        }
    }
    return formattedTrains
}

func findStationStop(schedule []models.TrainStop, stationCode string) *models.TrainStop {
    for _, stop := range schedule {
        if stop.Station == stationCode {
            return &stop
        }
    }
    return nil
}

func getStationFacilities(stationCode string) []string {
    // This could be expanded to fetch from database
    return []string{
        "Waiting Room",
        "Parking",
        "Food Court",
        "ATM",
        "Wheelchair Access",
    }
}