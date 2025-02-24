package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "math"
    "net/http"
    "regexp"
    "sort"
    "strings"
    "time"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "village_site/config"
)

// Struct definitions
type BusStop struct {
    StopName string  `json:"stop_name" bson:"stop_name"`
    Lat      float64 `json:"lat" bson:"lat"`
    Lng      float64 `json:"lng" bson:"lng"`
}

type BusRoute struct {
    ID            primitive.ObjectID `json:"_id" bson:"_id"`
    City          string            `json:"city" bson:"city"`
    RouteName     string            `json:"route_name" bson:"route_name"`
    StartingStage string            `json:"starting_stage" bson:"starting_stage"`
    EndingStage   string            `json:"ending_stage" bson:"ending_stage"`
    Distance      string            `json:"distance" bson:"distance"`
    Route         []BusStop         `json:"route" bson:"route"`
}

type BusRouteDetails struct {
    RouteName     string    `json:"route_name"`
    StartingStage string    `json:"starting_stage"`
    EndingStage   string    `json:"ending_stage"`
    Distance      string    `json:"distance"`
    Duration      string    `json:"duration"`
    Stops         []BusStop `json:"stops"`
    StopCount     int       `json:"stop_count"`
}

type InterchangeRoute struct {
    FirstRoute    BusRouteDetails `json:"first_route"`
    InterchangeAt string          `json:"interchange_at"`
    SecondRoute   BusRouteDetails `json:"second_route"`
    TotalDistance string          `json:"total_distance"`
    TotalDuration string          `json:"total_duration"`
    TotalStops    int             `json:"total_stops"`
}

type MultiInterchangeRoute struct {
    Routes          []BusRouteDetails `json:"routes"`
    InterchangeAt   []string          `json:"interchange_at"`
    TotalDistance   string            `json:"total_distance"`
    TotalDuration   string            `json:"total_duration"`
    TotalStops      int               `json:"total_stops"`
    InterchangeCount int              `json:"interchange_count"`
}

type BusRouteResponse struct {
    DirectRoutes          []BusRouteDetails     `json:"direct_routes,omitempty"`
    SingleInterchanges    []InterchangeRoute    `json:"single_interchange_routes,omitempty"`
    MultipleInterchanges  []MultiInterchangeRoute `json:"multiple_interchange_routes,omitempty"`
    TotalDistance        string                 `json:"total_distance,omitempty"`
    TotalDuration        string                 `json:"total_duration,omitempty"`
    InterchangeCount     int                    `json:"interchange_count"`
    TotalStops           int                    `json:"total_stops"`
}

// Helper functions
func normalizeStopName(name string) string {
    name = strings.ToLower(strings.TrimSpace(name))
    replacements := map[string]string{
        "terminal": "",
        "bus stop": "",
        "station":  "",
        "stand":    "",
        ".":        "",
        ",":        "",
    }
    for old, new := range replacements {
        name = strings.ReplaceAll(name, old, new)
    }
    return strings.TrimSpace(name)
}

func stopsMatch(stop1, stop2 string) bool {
    stop1 = normalizeStopName(stop1)
    stop2 = normalizeStopName(stop2)
    
    words1 := strings.Fields(stop1)
    words2 := strings.Fields(stop2)
    
    matchCount := 0
    for _, w1 := range words1 {
        for _, w2 := range words2 {
            if strings.Contains(w1, w2) || strings.Contains(w2, w1) {
                matchCount++
            }
        }
    }
    
    return matchCount >= len(words1)/2 || matchCount >= len(words2)/2
}

func parseDistance(distance string) float64 {
    var dist float64
    distStr := strings.TrimSuffix(strings.TrimSpace(distance), "K.M.")
    fmt.Sscanf(strings.TrimSpace(distStr), "%f", &dist)
    return dist
}

func formatDistanceBus(distance float64) string {
    return fmt.Sprintf("%.1f K.M.", distance)
}

func formatDurationBus(distance string) string {
    dist := parseDistance(distance)
    totalMinutes := int((dist / 20.0) * 60)
    hours := totalMinutes / 60
    minutes := totalMinutes % 60
    
    if hours > 0 {
        return fmt.Sprintf("%d hours %d minutes", hours, minutes)
    }
    return fmt.Sprintf("%d minutes", minutes)
}

func getDurationMinutes(distance string) int {
    dist := parseDistance(distance)
    return int((dist / 20.0) * 60)
}

func calculateTotalDistance(distances ...string) string {
    var total float64
    for _, distance := range distances {
        total += parseDistance(distance)
    }
    return formatDistanceBus(total)
}

// Debug function
func debugBusRoutes(ctx context.Context, city string) {
    collection := config.MongoClient.Database("train_database").Collection("bus_routes")
    
    filter := bson.M{
        "city": bson.M{
            "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(city)),
            "$options": "i",
        },
    }
    
    count, err := collection.CountDocuments(ctx, filter)
    if err != nil {
        log.Printf("Error counting routes: %v", err)
        return
    }
    log.Printf("Total routes in %s: %d", city, count)

    cursor, err := collection.Find(ctx, filter, options.Find().SetLimit(2))
    if err != nil {
        log.Printf("Error finding routes: %v", err)
        return
    }
    defer cursor.Close(ctx)

    var routes []BusRoute
    if err = cursor.All(ctx, &routes); err != nil {
        log.Printf("Error decoding routes: %v", err)
        return
    }

    for _, route := range routes {
        log.Printf("Route %s: %s -> %s", route.RouteName, route.StartingStage, route.EndingStage)
        log.Printf("Number of stops: %d", len(route.Route))
        if len(route.Route) > 0 {
            log.Printf("First stop: %s", route.Route[0].StopName)
            log.Printf("Last stop: %s", route.Route[len(route.Route)-1].StopName)
        }
    }
}

// Main handler functions
func GetCityRoutes(w http.ResponseWriter, r *http.Request) {
    var req struct {
        City string `json:"city"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("Error decoding request: %v", err)
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := config.MongoClient.Database("train_database").Collection("bus_routes")
    
    filter := bson.M{
        "city": bson.M{
            "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(req.City)),
            "$options": "i",
        },
    }

    opts := options.Find().SetSort(bson.D{{Key: "route_name", Value: 1}})
    cursor, err := collection.Find(ctx, filter, opts)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching routes", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    var routes []BusRoute
    if err = cursor.All(ctx, &routes); err != nil {
        log.Printf("Error processing routes: %v", err)
        http.Error(w, "Error processing routes", http.StatusInternalServerError)
        return
    }

    type RouteInfo struct {
        RouteName     string `json:"route_name"`
        StartingStage string `json:"starting_stage"`
        EndingStage   string `json:"ending_stage"`
        Distance      string `json:"distance"`
        Duration      string `json:"duration"`
    }

    var routeInfos []RouteInfo
    for _, route := range routes {
        routeInfos = append(routeInfos, RouteInfo{
            RouteName:     route.RouteName,
            StartingStage: route.StartingStage,
            EndingStage:   route.EndingStage,
            Distance:      route.Distance,
            Duration:      formatDurationBus(route.Distance),
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "city":   req.City,
        "routes": routeInfos,
        "count":  len(routeInfos),
    })
}

func GetBusRoute(w http.ResponseWriter, r *http.Request) {
    var req struct {
        City      string `json:"city"`
        RouteName string `json:"route_name"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("Error decoding request: %v", err)
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := config.MongoClient.Database("train_database").Collection("bus_routes")
    
    filter := bson.M{
        "city": bson.M{
            "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(req.City)),
            "$options": "i",
        },
        "route_name": bson.M{
            "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(req.RouteName)),
            "$options": "i",
        },
    }

    var route BusRoute
    err := collection.FindOne(ctx, filter).Decode(&route)
    if err == mongo.ErrNoDocuments {
        log.Printf("Route not found: city=%s, route=%s", req.City, req.RouteName)
        http.Error(w, "Route not found", http.StatusNotFound)
        return
    } else if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching route", http.StatusInternalServerError)
        return
    }

    response := struct {
        BusRoute
        Duration string `json:"duration"`
    }{
        BusRoute: route,
        Duration: formatDurationBus(route.Distance),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func GetCityStops(w http.ResponseWriter, r *http.Request) {
    var req struct {
        City string `json:"city"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("Error decoding request: %v", err)
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := config.MongoClient.Database("train_database").Collection("bus_routes")
    
    pipeline := []bson.M{
        {
            "$match": bson.M{
                "city": bson.M{
                    "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(req.City)),
                    "$options": "i",
                },
            },
        },
        {"$unwind": "$route"},
        {
            "$group": bson.M{
                "_id": "$route.stop_name",
                "lat": bson.M{"$first": "$route.lat"},
                "lng": bson.M{"$first": "$route.lng"},
            },
        },
        {"$sort": bson.M{"_id": 1}},
    }

    cursor, err := collection.Aggregate(ctx, pipeline)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching stops", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    type StopInfo struct {
        Name string  `json:"name"`
        Lat  float64 `json:"lat"`
        Lng  float64 `json:"lng"`
    }

    var stops []StopInfo
    for cursor.Next(ctx) {
        var result struct {
            ID  string  `bson:"_id"`
            Lat float64 `bson:"lat"`
            Lng float64 `bson:"lng"`
        }
        if err := cursor.Decode(&result); err != nil {
            continue
        }
        if result.ID != "" {
            stops = append(stops, StopInfo{
                Name: result.ID,
                Lat:  result.Lat,
                Lng:  result.Lng,
            })
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "city":  req.City,
        "stops": stops,
        "count": len(stops),
    })
}

func findRoutePath(
    ctx context.Context,
    collection *mongo.Collection,
    city, targetStop string,
    currentRoute BusRoute,
    visited map[string]bool,
    currentPath []BusRoute,
    interchangePoints []string,
    maxInterchanges int,
    results *[]MultiInterchangeRoute) {

    if len(currentPath) > maxInterchanges+1 {
        return
    }

    // Check if current route contains target stop
    targetIdx := -1
    for i, stop := range currentRoute.Route {
        if stopsMatch(stop.StopName, targetStop) {
            targetIdx = i
            break
        }
    }

    if targetIdx != -1 {
        // Found a valid path to target
        var routeDetails []BusRouteDetails
        var totalDistance float64
        var routeSegments [][]BusStop

        // Process each route segment
        for i, route := range currentPath {
            startIdx := 0
            endIdx := len(route.Route)

            if i == 0 { // First route
                for j, stop := range route.Route {
                    if stopsMatch(stop.StopName, currentPath[0].Route[0].StopName) {
                        startIdx = j
                        break
                    }
                }
            }

            if i == len(currentPath)-1 { // Last route
                endIdx = targetIdx + 1
            }

            routeSegment := route.Route[startIdx:endIdx]
            routeSegments = append(routeSegments, routeSegment)

            routeDetail := BusRouteDetails{
                RouteName:     route.RouteName,
                StartingStage: route.StartingStage,
                EndingStage:   route.EndingStage,
                Distance:      route.Distance,
                Duration:      formatDurationBus(route.Distance),
                Stops:         routeSegment,
                StopCount:     len(routeSegment),
            }
            routeDetails = append(routeDetails, routeDetail)
            totalDistance += parseDistance(route.Distance)
        }

        // Calculate total stops excluding duplicates at interchange points
        totalStops := 0
        for i, segment := range routeSegments {
            if i == 0 {
                totalStops += len(segment)
            } else {
                // Add length of segment minus the interchange point
                totalStops += len(segment) - 1
            }
        }

        multiRoute := MultiInterchangeRoute{
            Routes:           routeDetails,
            InterchangeAt:    interchangePoints,
            TotalDistance:    formatDistanceBus(totalDistance),
            TotalDuration:    formatDurationBus(formatDistanceBus(totalDistance)),
            TotalStops:       totalStops,
            InterchangeCount: len(currentPath) - 1,
        }

        *results = append(*results, multiRoute)
        return
    }

    // Try finding connecting routes through interchange points
    visited[currentRoute.RouteName] = true
    
    for i, stop := range currentRoute.Route {
        if i < len(currentRoute.Route)-1 {
            // Find routes that contain this stop
            connectingRoutes, err := findRoutesWithStop(ctx, collection, city, stop.StopName)
            if err != nil {
                continue
            }

            for _, nextRoute := range connectingRoutes {
                if !visited[nextRoute.RouteName] {
                    // Check if this is a viable interchange point
                    viability := calculateInterchangeViability(currentRoute, nextRoute, stop.StopName)
                    if viability < 100 { // Only consider viable interchanges
                        newPath := append(currentPath, nextRoute)
                        newInterchangePoints := append(interchangePoints, stop.StopName)
                        findRoutePath(ctx, collection, city, targetStop, nextRoute, visited,
                            newPath, newInterchangePoints, maxInterchanges, results)
                    }
                }
            }
        }
    }

    delete(visited, currentRoute.RouteName)
}

func calculateTotalStops(routes []BusRouteDetails) int {
    totalStops := 0
    for _, route := range routes {
        totalStops += route.StopCount
    }
    return totalStops - (len(routes) - 1) // Subtract interchange points as they're counted twice
}

func FindBusRoute(w http.ResponseWriter, r *http.Request) {
    var req struct {
        City     string `json:"city"`
        FromStop string `json:"from_stop"`
        ToStop   string `json:"to_stop"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("Error decoding request: %v", err)
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    log.Printf("Searching route: city=%s, from=%s, to=%s", req.City, req.FromStop, req.ToStop)

    var response BusRouteResponse

    // Step 1: Try direct routes first
    directRoutes, err := findDirectBusRoutes(ctx, req.City, req.FromStop, req.ToStop)
    if err != nil {
        log.Printf("Error finding direct routes: %v", err)
    }

    if len(directRoutes) > 0 {
        // If we have direct routes, only use those
        processDirectRoutes(&response, directRoutes)
    } else {
        // Step 2: Try single interchange routes
        singleInterchanges, err := findInterchangeRoutes(ctx, req.City, req.FromStop, req.ToStop)
        if err != nil {
            log.Printf("Error finding single interchange routes: %v", err)
        }

        if len(singleInterchanges) > 0 {
            // Filter single interchange routes by total distance
            filteredInterchanges := filterBestInterchangeRoutes(singleInterchanges)
            processSingleInterchangeRoutes(&response, filteredInterchanges)
        } else {
            // Step 3: Only try multiple interchanges if no better options exist
            multiInterchanges, err := findMultipleInterchangeRoutes(ctx, req.City, req.FromStop, req.ToStop)
            if err != nil {
                log.Printf("Error finding multiple interchange routes: %v", err)
            }

            if len(multiInterchanges) > 0 {
                // Filter multiple interchange routes to only show if they're reasonable
                filteredMultiInterchanges := filterBestMultiInterchangeRoutes(multiInterchanges)
                if len(filteredMultiInterchanges) > 0 {
                    processMultipleInterchangeRoutes(&response, filteredMultiInterchanges)
                }
            }
        }
    }

    if len(response.DirectRoutes) == 0 && 
       len(response.SingleInterchanges) == 0 && 
       len(response.MultipleInterchanges) == 0 {
        log.Printf("No routes found")
        http.Error(w, "No routes found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func filterBestInterchangeRoutes(routes []InterchangeRoute) []InterchangeRoute {
    if len(routes) == 0 {
        return routes
    }

    // Sort by total distance first
    sort.Slice(routes, func(i, j int) bool {
        distI := parseDistance(routes[i].TotalDistance)
        distJ := parseDistance(routes[j].TotalDistance)
        return distI < distJ
    })

    // Get the shortest distance
    shortestDist := parseDistance(routes[0].TotalDistance)
    
    // Only keep routes that are at most 30% longer than the shortest route
    var filtered []InterchangeRoute
    for _, route := range routes {
        dist := parseDistance(route.TotalDistance)
        if dist <= shortestDist*1.3 { // Allow up to 30% longer
            filtered = append(filtered, route)
        }
    }

    // Sort final results by multiple criteria
    sort.Slice(filtered, func(i, j int) bool {
        distI := parseDistance(filtered[i].TotalDistance)
        distJ := parseDistance(filtered[j].TotalDistance)
        if math.Abs(distI - distJ) > 0.1 { // If distances differ by more than 100m
            return distI < distJ
        }
        // If distances are very close, prefer fewer stops
        return filtered[i].TotalStops < filtered[j].TotalStops
    })

    // Return at most 3 best routes
    if len(filtered) > 3 {
        filtered = filtered[:3]
    }
    return filtered
}

func filterBestMultiInterchangeRoutes(routes []MultiInterchangeRoute) []MultiInterchangeRoute {
    if len(routes) == 0 {
        return routes
    }

    // Sort by total distance first
    sort.Slice(routes, func(i, j int) bool {
        distI := parseDistance(routes[i].TotalDistance)
        distJ := parseDistance(routes[j].TotalDistance)
        return distI < distJ
    })

    // Get the shortest distance
    shortestDist := parseDistance(routes[0].TotalDistance)
    
    // Only keep routes that are at most 40% longer than the shortest route
    // and have reasonable number of interchanges
    var filtered []MultiInterchangeRoute
    for _, route := range routes {
        dist := parseDistance(route.TotalDistance)
        if dist <= shortestDist*1.4 && route.InterchangeCount <= 2 {
            filtered = append(filtered, route)
        }
    }

    // Sort final results by multiple criteria
    sort.Slice(filtered, func(i, j int) bool {
        distI := parseDistance(filtered[i].TotalDistance)
        distJ := parseDistance(filtered[j].TotalDistance)
        if math.Abs(distI - distJ) > 0.1 {
            return distI < distJ
        }
        if filtered[i].InterchangeCount != filtered[j].InterchangeCount {
            return filtered[i].InterchangeCount < filtered[j].InterchangeCount
        }
        return filtered[i].TotalStops < filtered[j].TotalStops
    })

    // Return at most 2 best routes
    if len(filtered) > 2 {
        filtered = filtered[:2]
    }
    return filtered
}

// Route finding functions
func findDirectBusRoutes(ctx context.Context, city, fromStop, toStop string) ([]BusRoute, error) {
    collection := config.MongoClient.Database("train_database").Collection("bus_routes")
    
    log.Printf("Searching for direct routes from '%s' to '%s' in %s", fromStop, toStop, city)
    
    filter := bson.M{
        "city": bson.M{
            "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(city)),
            "$options": "i",
        },
        "route": bson.M{
            "$all": []bson.M{
                {
                    "$elemMatch": bson.M{
                        "stop_name": bson.M{
                            "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(fromStop)),
                            "$options": "i",
                        },
                    },
                },
                {
                    "$elemMatch": bson.M{
                        "stop_name": bson.M{
                            "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(toStop)),
                            "$options": "i",
                        },
                    },
                },
            },
        },
    }

    cursor, err := collection.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var routesWithMetrics []struct {
        Route   BusRoute
        Metrics RouteMetrics
    }

    for cursor.Next(ctx) {
        var route BusRoute
        if err := cursor.Decode(&route); err != nil {
            continue
        }

        fromIdx := -1
        toIdx := -1
        for i, stop := range route.Route {
            if stopsMatch(stop.StopName, fromStop) {
                fromIdx = i
            }
            if stopsMatch(stop.StopName, toStop) {
                toIdx = i
            }
        }

        if fromIdx != -1 && toIdx != -1 && fromIdx < toIdx {
            metrics := calculateRouteMetrics(route, fromIdx, toIdx)
            route.Route = route.Route[fromIdx : toIdx+1]
            routesWithMetrics = append(routesWithMetrics, struct {
                Route   BusRoute
                Metrics RouteMetrics
            }{route, metrics})
        }
    }

    // Sort by multiple criteria
    sort.Slice(routesWithMetrics, func(i, j int) bool {
        mi := routesWithMetrics[i].Metrics
        mj := routesWithMetrics[j].Metrics
        
        // First prioritize distance
        if math.Abs(mi.TotalDistance - mj.TotalDistance) > 0.1 {
            return mi.TotalDistance < mj.TotalDistance
        }
        
        // Then duration
        if mi.Duration != mj.Duration {
            return mi.Duration < mj.Duration
        }
        
        // Finally comfort
        return mi.Comfort > mj.Comfort
    })

    // Convert back to []BusRoute
    result := make([]BusRoute, len(routesWithMetrics))
    for i, r := range routesWithMetrics {
        result[i] = r.Route
    }

    return result, nil
}

func findInterchangeRoutes(ctx context.Context, city, fromStop, toStop string) ([]InterchangeRoute, error) {
    collection := config.MongoClient.Database("train_database").Collection("bus_routes")
    
    log.Printf("Searching for single interchange routes from '%s' to '%s' in %s", fromStop, toStop, city)
    
    fromRoutes, err := findRoutesWithStop(ctx, collection, city, fromStop)
    if err != nil {
        return nil, err
    }

    type routeWithViability struct {
        Route     InterchangeRoute
        Viability float64
    }
    var viableRoutes []routeWithViability

    for _, fromRoute := range fromRoutes {
        fromIdx := -1
        for i, stop := range fromRoute.Route {
            if stopsMatch(stop.StopName, fromStop) {
                fromIdx = i
                break
            }
        }

        if fromIdx == -1 {
            continue
        }

        for i := fromIdx + 1; i < len(fromRoute.Route); i++ {
            interchangeStop := fromRoute.Route[i].StopName
            
            toRoutes, err := findRoutesWithStop(ctx, collection, city, toStop)
            if err != nil {
                continue
            }

            for _, toRoute := range toRoutes {
                if fromRoute.RouteName == toRoute.RouteName {
                    continue
                }

                interIdx := -1
                toIdx := -1
                for j, stop := range toRoute.Route {
                    if stopsMatch(stop.StopName, interchangeStop) {
                        interIdx = j
                    }
                    if stopsMatch(stop.StopName, toStop) {
                        toIdx = j
                    }
                }

                if interIdx != -1 && toIdx != -1 && interIdx < toIdx {
                    viability := calculateInterchangeViability(fromRoute, toRoute, interchangeStop)
                    if viability < 100 { // Threshold for viable routes
                        firstRouteStops := fromRoute.Route[fromIdx : i+1]
                        secondRouteStops := toRoute.Route[interIdx : toIdx+1]

                        totalDistance := calculateTotalDistance(fromRoute.Distance, toRoute.Distance)
                        
                        interchangeRoute := InterchangeRoute{
                            FirstRoute: BusRouteDetails{
                                RouteName:     fromRoute.RouteName,
                                StartingStage: fromRoute.StartingStage,
                                EndingStage:   fromRoute.EndingStage,
                                Distance:      fromRoute.Distance,
                                Duration:      formatDurationBus(fromRoute.Distance),
                                Stops:         firstRouteStops,
                                StopCount:     len(firstRouteStops),
                            },
                            InterchangeAt: interchangeStop,
                            SecondRoute: BusRouteDetails{
                                RouteName:     toRoute.RouteName,
                                StartingStage: toRoute.StartingStage,
                                EndingStage:   toRoute.EndingStage,
                                Distance:      toRoute.Distance,
                                Duration:      formatDurationBus(toRoute.Distance),
                                Stops:         secondRouteStops,
                                StopCount:     len(secondRouteStops),
                            },
                            TotalDistance: totalDistance,
                            TotalDuration: formatDurationBus(totalDistance),
                            TotalStops:    len(firstRouteStops) + len(secondRouteStops) - 1,
                        }
                        viableRoutes = append(viableRoutes, routeWithViability{
                            Route:     interchangeRoute,
                            Viability: viability,
                        })
                    }
                }
            }
        }
    }

    // Sort by viability
    sort.Slice(viableRoutes, func(i, j int) bool {
        return viableRoutes[i].Viability < viableRoutes[j].Viability
    })

    // Take top 5 most viable routes
    result := make([]InterchangeRoute, 0)
    for i := 0; i < len(viableRoutes) && i < 5; i++ {
        result = append(result, viableRoutes[i].Route)
    }

    return result, nil
}

func findMultipleInterchangeRoutes(ctx context.Context, city, fromStop, toStop string) ([]MultiInterchangeRoute, error) {
    collection := config.MongoClient.Database("train_database").Collection("bus_routes")
    
    log.Printf("Searching for multiple interchange routes from '%s' to '%s' in %s", fromStop, toStop, city)
    
    fromRoutes, err := findRoutesWithStop(ctx, collection, city, fromStop)
    if err != nil {
        return nil, err
    }

    type routeWithScore struct {
        Route MultiInterchangeRoute
        Score float64
    }
    var scoredRoutes []routeWithScore
    var multiInterchangeRoutes []MultiInterchangeRoute
    maxInterchanges := 2 // Maximum 2 interchanges (3 buses)

    for _, startRoute := range fromRoutes {
        visited := make(map[string]bool)
        path := []BusRoute{startRoute}
        interchangePoints := []string{}
        
        findRoutePath(ctx, collection, city, toStop, startRoute, visited, path, 
            interchangePoints, maxInterchanges, &multiInterchangeRoutes)
    }

    // Score and sort routes
    for _, route := range multiInterchangeRoutes {
        var routeList []BusRoute
        for _, rd := range route.Routes {
            var br BusRoute
            br.RouteName = rd.RouteName
            br.StartingStage = rd.StartingStage
            br.EndingStage = rd.EndingStage
            br.Distance = rd.Distance
            br.Route = rd.Stops
            routeList = append(routeList, br)
        }
        
        score := optimizeRoutePath(routeList, route.InterchangeAt)
        scoredRoutes = append(scoredRoutes, routeWithScore{
            Route: route,
            Score: score,
        })
    }

    // Sort by score (lower is better)
    sort.Slice(scoredRoutes, func(i, j int) bool {
        if math.Abs(scoredRoutes[i].Score - scoredRoutes[j].Score) > 0.1 {
            return scoredRoutes[i].Score < scoredRoutes[j].Score
        }
        // If scores are very close, prefer routes with fewer interchanges
        return scoredRoutes[i].Route.InterchangeCount < scoredRoutes[j].Route.InterchangeCount
    })

    // Take top 3 best routes
    result := make([]MultiInterchangeRoute, 0)
    for i := 0; i < len(scoredRoutes) && i < 3; i++ {
        result = append(result, scoredRoutes[i].Route)
    }

    return result, nil
}

func findRoutesWithStop(ctx context.Context, collection *mongo.Collection, city, stopName string) ([]BusRoute, error) {
    filter := bson.M{
        "city": bson.M{
            "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(city)),
            "$options": "i",
        },
        "route": bson.M{
            "$elemMatch": bson.M{
                "stop_name": bson.M{
                    "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(stopName)),
                    "$options": "i",
                },
            },
        },
    }

    cursor, err := collection.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var routes []BusRoute
    if err = cursor.All(ctx, &routes); err != nil {
        return nil, err
    }

    return routes, nil
}

// Helper functions for processing routes
func processDirectRoutes(response *BusRouteResponse, routes []BusRoute) {
    // Sort direct routes by duration and stops
    sort.Slice(routes, func(i, j int) bool {
        durationI := getDurationMinutes(routes[i].Distance)
        durationJ := getDurationMinutes(routes[j].Distance)
        if durationI != durationJ {
            return durationI < durationJ
        }
        return len(routes[i].Route) < len(routes[j].Route)
    })

    for _, route := range routes {
        routeDetails := BusRouteDetails{
            RouteName:     route.RouteName,
            StartingStage: route.StartingStage,
            EndingStage:   route.EndingStage,
            Distance:      route.Distance,
            Duration:      formatDurationBus(route.Distance),
            Stops:         route.Route,
            StopCount:     len(route.Route),
        }
        response.DirectRoutes = append(response.DirectRoutes, routeDetails)
    }
    response.TotalDistance = routes[0].Distance
    response.TotalDuration = formatDurationBus(routes[0].Distance)
    response.TotalStops = len(routes[0].Route)
    response.InterchangeCount = 0
}

func processSingleInterchangeRoutes(response *BusRouteResponse, routes []InterchangeRoute) {
    sort.Slice(routes, func(i, j int) bool {
        durationI := getDurationMinutes(routes[i].TotalDistance)
        durationJ := getDurationMinutes(routes[j].TotalDistance)
        if durationI != durationJ {
            return durationI < durationJ
        }
        return routes[i].TotalStops < routes[j].TotalStops
    })

    if len(routes) > 3 {
        routes = routes[:3]
    }

    response.SingleInterchanges = routes
    response.TotalDistance = routes[0].TotalDistance
    response.TotalDuration = routes[0].TotalDuration
    response.TotalStops = routes[0].TotalStops
    response.InterchangeCount = 1
}

func processMultipleInterchangeRoutes(response *BusRouteResponse, routes []MultiInterchangeRoute) {
    response.MultipleInterchanges = routes
    response.TotalDistance = routes[0].TotalDistance
    response.TotalDuration = routes[0].TotalDuration
    response.TotalStops = routes[0].TotalStops
    response.InterchangeCount = routes[0].InterchangeCount
}

// Add these new handler functions
func GetBusStopSuggestions(w http.ResponseWriter, r *http.Request) {
    city := r.URL.Query().Get("city")
    searchTerm := r.URL.Query().Get("q")

    if city == "" {
        http.Error(w, "City is required", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := config.MongoClient.Database("train_database").Collection("bus_routes")

    // Normalize city name for Howrah special case
    normalizedCity := strings.ToLower(city)
    if normalizedCity == "howrah" {
        city = "Kolkata" // Use Kolkata for Howrah searches
    }

    pipeline := []bson.M{
        {
            "$match": bson.M{
                "city": bson.M{
                    "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(city)),
                    "$options": "i",
                },
            },
        },
        {"$unwind": "$route"},
        {
            "$match": bson.M{
                "route.stop_name": bson.M{
                    "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(searchTerm)),
                    "$options": "i",
                },
            },
        },
        {
            "$group": bson.M{
                "_id": "$route.stop_name",
                "lat": bson.M{"$first": "$route.lat"},
                "lng": bson.M{"$first": "$route.lng"},
            },
        },
        {"$sort": bson.M{"_id": 1}},
        {"$limit": 10},
    }

    cursor, err := collection.Aggregate(ctx, pipeline)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching suggestions", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    type StopSuggestion struct {
        Name      string  `json:"name"`
        Latitude  float64 `json:"lat"`
        Longitude float64 `json:"lng"`
    }

    var suggestions []StopSuggestion
    for cursor.Next(ctx) {
        var result struct {
            ID  string  `bson:"_id"`
            Lat float64 `bson:"lat"`
            Lng float64 `bson:"lng"`
        }
        if err := cursor.Decode(&result); err != nil {
            continue
        }
        if result.ID != "" {
            suggestions = append(suggestions, StopSuggestion{
                Name:      result.ID,
                Latitude:  result.Lat,
                Longitude: result.Lng,
            })
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "suggestions": suggestions,
    })
}

// Add this function to handle city suggestions
func GetCitySuggestions(w http.ResponseWriter, r *http.Request) {
    searchTerm := r.URL.Query().Get("q")
    if searchTerm == "" {
        http.Error(w, "Search term is required", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := config.MongoClient.Database("train_database").Collection("bus_routes")

    pipeline := []bson.M{
        {
            "$match": bson.M{
                "city": bson.M{
                    "$regex": fmt.Sprintf(".*%s.*", regexp.QuoteMeta(searchTerm)),
                    "$options": "i",
                },
            },
        },
        {
            "$group": bson.M{
                "_id": "$city",
            },
        },
        {"$sort": bson.M{"_id": 1}},
        {"$limit": 10},
    }

    cursor, err := collection.Aggregate(ctx, pipeline)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching suggestions", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    var suggestions []string
    for cursor.Next(ctx) {
        var result struct {
            ID string `bson:"_id"`
        }
        if err := cursor.Decode(&result); err != nil {
            continue
        }
        if result.ID != "" {
            suggestions = append(suggestions, result.ID)
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "suggestions": suggestions,
    })
}

func GetAllCities(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := config.MongoClient.Database("train_database").Collection("bus_routes")

    pipeline := []bson.M{
        {
            "$group": bson.M{
                "_id": "$city",
                "route_count": bson.M{"$sum": 1},
            },
        },
        {
            "$project": bson.M{
                "city": "$_id",
                "route_count": 1,
                "_id": 0,
            },
        },
        {"$sort": bson.M{"city": 1}},
    }

    cursor, err := collection.Aggregate(ctx, pipeline)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching cities", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    type CityInfo struct {
        City       string `json:"city"`
        RouteCount int    `json:"route_count"`
    }

    var cities []CityInfo
    for cursor.Next(ctx) {
        var result CityInfo
        if err := cursor.Decode(&result); err != nil {
            continue
        }
        if result.City != "" {
            cities = append(cities, result)
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "cities": cities,
        "count":  len(cities),
    })
}