package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "sort"
    "strings"
    "sync"
    "time"
    "math"
    "village_site/config"
    "village_site/utils"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "github.com/gorilla/mux"
    "strconv"
    "sync/atomic"
)

const (
    maxRoutes = 5
    maxInterchanges = 3
    maxInterchangeRoutes = 3
    searchTimeout = 5 * time.Second  // Increased from 2s to 5s
    minWaitingTime = 15 * time.Minute
    maxWaitingTime = 6 * time.Hour
    maxDistance = 3000.0
)

// Cache structures
var (
    trainCache = struct {
        sync.RWMutex
        routes map[string]map[string][]*TrainRoute
        expiry time.Time
    }{
        routes: make(map[string]map[string][]*TrainRoute),
    }

    stationCache = struct {
        sync.RWMutex
        codes map[string]*StationInfo
        names map[string]string
        expiry time.Time
    }{
        codes: make(map[string]*StationInfo),
        names: make(map[string]string),
    }
)

// Add at the top with other global variables
var equivalentStations = map[string][]string{
    // Delhi Region
    "NDLS": {"DLI", "DEE", "NZM"},  // New Delhi, Delhi Junction, Delhi Sarai Rohilla, Hazrat Nizamuddin
    "DLI": {"NDLS", "DEE", "NZM"},
    "DEE": {"NDLS", "DLI", "NZM"},
    "NZM": {"NDLS", "DLI", "DEE"},
    
    // Anand Vihar and Ghaziabad
    "ANVT": {"ANVR"},  // Anand Vihar Terminal
    "ANVR": {"ANVT"},
    "GZB": {"GUH"},    // Ghaziabad and Guldhar
    "GUH": {"GZB"},

    // Faridabad Area
    "FDB": {"FDN", "BVH"},  // Faridabad, Faridabad New Town, Ballabgarh
    "FDN": {"FDB", "BVH"},
    "BVH": {"FDB", "FDN"},

        // Mumbai Central Area
        "BCT": {"MMCT", "CCG"},  // Mumbai Central, Mumbai Central Local
        "MMCT": {"BCT", "CCG"},
        "CCG": {"BCT", "MMCT"},
    
        // CST Mumbai Area
        "CSMT": {"CSTM", "ST"},  // Chhatrapati Shivaji Terminus
        "CSTM": {"CSMT", "ST"},
        "ST": {"CSMT", "CSTM"},
    
        // Kurla and Lokmanya Tilak
        "LTT": {"KLVA", "CLA"},  // Lokmanya Tilak Terminus, Kurla
        "KLVA": {"LTT", "CLA"},
        "CLA": {"LTT", "KLVA"},
    
        // Bandra and Dadar
        "BDTS": {"BVI", "DDR"},  // Bandra Terminus, Bandra
        "BVI": {"BDTS", "DDR"},
        "DDR": {"BDTS", "BVI"},  // Dadar

            // Howrah Area
    "HWH": {"BWN", "KOAA"},  // Howrah Junction, Barddhaman, Kolkata
    "KOAA": {"HWH", "BWN"},
    "BWN": {"HWH", "KOAA"},

    // Sealdah Area
    "SDAH": {"DDJ", "NKDM"},  // Sealdah, Dankuni Junction, Naihati
    "DDJ": {"SDAH", "NKDM"},
    "NKDM": {"SDAH", "DDJ"},

    // Kolkata Area
    "CP": {"CG", "KOAA"},    // Kolkata Chitpur, Kolkata Terminal
    "CG": {"CP", "KOAA"},

        // Chennai Central Area
        "MAS": {"MSB", "MS"},    // Chennai Central, Chennai Basin Bridge
        "MSB": {"MAS", "MS"},
        "MS": {"MAS", "MSB"},    // Chennai Egmore
    
        // Tambaram Area
        "TBM": {"MKK", "CGL"},   // Tambaram, Maraimalai Nagar, Chengalpattu
        "MKK": {"TBM", "CGL"},
        "CGL": {"TBM", "MKK"},
    
        // Avadi Area
        "AVD": {"PTW", "VLK"},   // Avadi, Pattabiram, Villivakkam
        "PTW": {"AVD", "VLK"},
        "VLK": {"AVD", "PTW"},

            // Bangalore City Area
    "SBC": {"BNC", "BNCE"},  // KSR Bangalore, Bangalore Cantonment
    "BNC": {"SBC", "BNCE"},
    "BNCE": {"SBC", "BNC"},

    // Yeshwantpur Area
    "YPR": {"YFR", "BAND"},  // Yeshwantpur, Yeswanthpur, Banasawadi
    "YFR": {"YPR", "BAND"},
    "BAND": {"YPR", "YFR"},

    // Whitefield Area
    "WTFD": {"BYPL", "KJM"}, // Whitefield, Baiyappanahalli, Krishnarajapuram
    "BYPL": {"WTFD", "KJM"},
    "KJM": {"WTFD", "BYPL"},

        // Secunderabad Area
        "SC": {"HYB", "KCG"},    // Secunderabad, Hyderabad, Kacheguda
        "HYB": {"SC", "KCG"},
        "KCG": {"SC", "HYB"},
    
        // Begumpet Area
        "BMT": {"NZB", "MJF"},   // Begumpet, Nampally, Malakpet
        "NZB": {"BMT", "MJF"},
        "MJF": {"BMT", "NZB"},

            // Pune Area
    "PUNE": {"SVD", "GPR"},  // Pune Junction, Shivajinagar, Ghorpuri
    "SVD": {"PUNE", "GPR"},
    "GPR": {"PUNE", "SVD"},

    // Nagpur Area
    "NGP": {"AQ", "PPZ"},    // Nagpur Junction, Ajni, Parseoni
    "AQ": {"NGP", "PPZ"},
    "PPZ": {"NGP", "AQ"},

        // Ahmedabad Area
        "ADI": {"ASV", "SBT"},   // Ahmedabad Junction, Asarva, Sabarmati
        "ASV": {"ADI", "SBT"},
        "SBT": {"ADI", "ASV"},
    
        // Vadodara Area
        "BRC": {"VDA", "MSH"},   // Vadodara Junction, Vadodara, Makarpura
        "VDA": {"BRC", "MSH"},
        "MSH": {"BRC", "VDA"},

            // Lucknow Area
    "LKO": {"LJN", "LC"},    // Lucknow Charbagh NR, Lucknow Junction NER
    "LJN": {"LKO", "LC"},
    "LC": {"LKO", "LJN"},

    // Kanpur Area
    "CNB": {"CPA", "GMC"},   // Kanpur Central, Kanpur Anwarganj, Govindpuri
    "CPA": {"CNB", "GMC"},
    "GMC": {"CNB", "CPA"},

        // Allahabad/Prayagraj Area
        "PRYJ": {"ALD", "PFM"},  // Prayagraj Junction, Allahabad, Prayag
        "ALD": {"PRYJ", "PFM"},
        "PFM": {"PRYJ", "ALD"},
    
        // Varanasi Area
        "BSB": {"BCY", "BSBS"},  // Varanasi Junction, Varanasi City
        "BCY": {"BSB", "BSBS"},
        "BSBS": {"BSB", "BCY"},
    
        // Patna Area
        "PNBE": {"PNC", "PTJ"},  // Patna Junction, Patna City
        "PNC": {"PNBE", "PTJ"},
        "PTJ": {"PNBE", "PNC"},
    
        // Jaipur Area
        "JP": {"GADJ", "DPA"},   // Jaipur Junction, Gandhi Nagar Jaipur
        "GADJ": {"JP", "DPA"},
        "DPA": {"JP", "GADJ"},
    
        // Bhopal Area
        "BPL": {"HBJ", "MSO"},   // Bhopal Junction, Habibganj
        "HBJ": {"BPL", "MSO"},
        "MSO": {"BPL", "HBJ"},
}

// Basic types and structures
type TrainRoute struct {
    TrainNumber int      `json:"train_number" bson:"train_number"`
    Name        string   `json:"name"`
    Type        string   `json:"type"`
    FromStation string   `json:"from_station"`
    ToStation   string   `json:"to_station"`
    Departure   string   `json:"departure"`
    Arrival     string   `json:"arrival"`
    Platform    string   `json:"platform"`
    Classes     []string `json:"classes"`
    Distance    float64  `json:"distance"`
    Stops       []Stop   `json:"stops"`
}

type RouteResult struct {
    Type          string          `json:"type"`
    Trains        []*TrainRoute   `json:"trains"`
    Interchanges  []*Interchange  `json:"interchanges,omitempty"`
    TotalDuration time.Duration   `json:"total_duration"`
    TotalDistance float64         `json:"total_distance"`
    Score         float64         `json:"score"`
}

type InterchangePath struct {
    stations []string
    trains   []*TrainRoute
    waitTime time.Duration
    distance float64
    score    float64
}

// Define a station group structure
type StationGroup struct {
    Stations  map[string]bool
    MainCode  string
    Distance  float64  // Distance between stations in the group
}

// Create a station grouping manager
type StationManager struct {
    sync.RWMutex
    groups     map[string]*StationGroup  // Map of group ID to station group
    stationMap map[string]string         // Map of station code to group ID
}

var stationManager = &StationManager{
    groups:     make(map[string]*StationGroup),
    stationMap: make(map[string]string),
}

type StationSuggestion struct {
    StationCode string `json:"station_code"`
    StationName string `json:"station_name"`
}

type TrainSuggestion struct {
    TrainNumber int    `json:"train_number"`
    TrainName   string `json:"train_name"`
}

type Stop struct {
    Station   string  `json:"station"`
    Arrival   string  `json:"arrival"`
    Departure string  `json:"departure"`
    Platform  string  `json:"platform"`
    Distance  float64 `json:"distance"`
}

type Interchange struct {
    Station      string        `json:"station"`
    WaitingTime  time.Duration `json:"waiting_time"`
    FromTrain    int          `json:"from_train"`
    ToTrain      int          `json:"to_train"`
    Platform     string        `json:"platform"`
}

type Train struct {
    TrainNumber int      `bson:"train_number"`
    Title       string   `bson:"title"`
    Type        string   `bson:"type"`
    Schedule    []TrainSchedule `bson:"schedule_table"`
    Classes     []string `bson:"classes"`
}

type TrainSchedule struct {
    Day       int     `bson:"day"`
    Station   string  `bson:"station"`
    Arrival   string  `bson:"arrival"`
    Departure string  `bson:"departure"`
    Distance  string  `bson:"distance"`
    Platform  string  `bson:"platform"`
    Halt      string  `bson:"halt"`
}

type StationInfo struct {
    Code     string
    Name     string
    City     string
    Location struct {
        Lat float64
        Lon float64
    }
}

func initializeStationGroups() error {
    // Initialize from database or configuration
    collection := config.MongoDB.Collection("station_groups")
    ctx := context.Background()
    
    cursor, err := collection.Find(ctx, bson.M{})
    if err != nil {
        // If no configuration exists, use geographical proximity
        return initializeStationGroupsByProximity()
    }
    defer cursor.Close(ctx)
    
    for cursor.Next(ctx) {
        var group StationGroup
        if err := cursor.Decode(&group); err != nil {
            log.Printf("Warning: Failed to decode station group: %v", err)
            continue
        }
        addStationGroup(&group)
    }
    
    return nil
}

func initializeStationGroupsByProximity() error {
    // Get all stations with their coordinates
    stations := getAllStationsWithCoordinates()
    
    // Group stations that are within 2km of each other
    const maxDistance = 2.0 // kilometers
    
    for i, station1 := range stations {
        for j := i + 1; j < len(stations); j++ {
            station2 := stations[j]
            distance := calculateDistance(
                station1.Location.Lat, station1.Location.Lon,
                station2.Location.Lat, station2.Location.Lon,
            )
            
            if distance <= maxDistance {
                createOrUpdateStationGroup(station1, station2, distance)
            }
        }
    }
    
    return nil
}

func findRouteVia(from, via, to string) *RouteResult {
    // Find first leg
    leg1 := findDirectRoutes(from, via)
    if len(leg1) == 0 {
        return nil
    }

    // Find second leg
    leg2 := findDirectRoutes(via, to)
    if len(leg2) == 0 {
        return nil
    }

    // Calculate waiting time
    waitTime := calculateWaitingTime(
        leg1[0].Trains[0].Arrival,
        leg2[0].Trains[0].Departure,
    )

    if waitTime < minWaitingTime || waitTime > maxWaitingTime {
        return nil
    }

    return &RouteResult{
        Type:   "single-interchange",
        Trains: []*TrainRoute{leg1[0].Trains[0], leg2[0].Trains[0]},
        Interchanges: []*Interchange{{
            Station:     via,
            WaitingTime: waitTime,
            FromTrain:   leg1[0].Trains[0].TrainNumber,
            ToTrain:     leg2[0].Trains[0].TrainNumber,
            Platform:    leg2[0].Trains[0].Platform,
        }},
        TotalDuration: leg1[0].TotalDuration + leg2[0].TotalDuration + waitTime,
        TotalDistance: leg1[0].TotalDistance + leg2[0].TotalDistance,
        Score:         calculateRouteScore(leg1[0].Trains[0]) * 0.8, // Apply interchange penalty
    }
}

func calculateMultiLegScore(leg1, leg2, leg3 *RouteResult, wait1, wait2 time.Duration) float64 {
    // Base scores for each leg
    score1 := calculateRouteScore(leg1.Trains[0])
    score2 := calculateRouteScore(leg2.Trains[0])
    score3 := calculateRouteScore(leg3.Trains[0])
    
    // Average base score
    baseScore := (score1 + score2 + score3) / 3
    
    // Penalties
    waitingPenalty := ((wait1 + wait2).Hours() / (2 * maxWaitingTime.Hours())) * 0.3
    interchangePenalty := 0.2 // Fixed penalty for double interchange
    
    return baseScore * (1 - waitingPenalty - interchangePenalty)
}

func findRoutes(from, to string) []*RouteResult {
    startTime := time.Now()
    log.Printf("Starting route search from %s to %s", from, to)
    
    allRoutes := make([]*RouteResult, 0)

    // First check for direct routes including those that go through intermediate stations
    directRoutes := findDirectRoutesWithStops(from, to)
    if len(directRoutes) > 0 {
        log.Printf("Found %d direct routes", len(directRoutes))
        allRoutes = append(allRoutes, directRoutes...)
        return sortRoutes(allRoutes)
    }

    // Only proceed with interchange routes if no direct routes found
    log.Printf("No direct routes found, searching for interchange routes")
    
    // Get all connected stations for both source and destination
    fromConnections := getConnectedStations(from)
    toConnections := getConnectedStations(to)
    
    log.Printf("Source station %s has %d connections", from, len(fromConnections))
    log.Printf("Destination station %s has %d connections", to, len(toConnections))
    
    // Create a map for faster lookup of destination connections
    toConnectionMap := make(map[string]bool)
    for _, station := range toConnections {
        toConnectionMap[station] = true
    }

    // Try all possible interchange stations
    checkedStations := make(map[string]bool)
    for _, via := range fromConnections {
        if via == from || via == to || checkedStations[via] {
            continue
        }
        checkedStations[via] = true

        // Check if this station has connections to destination or its connected stations
        hasPathToDestination := false
        viaConnections := getConnectedStations(via)
        for _, nextStation := range viaConnections {
            if nextStation == to || toConnectionMap[nextStation] {
                hasPathToDestination = true
                break
            }
        }

        if !hasPathToDestination {
            continue
        }

        // Try creating interchange route
        fromViaRoutes := findDirectRoutesWithStops(from, via)
        if len(fromViaRoutes) == 0 {
            continue
        }

        viaToRoutes := findDirectRoutesWithStops(via, to)
        if len(viaToRoutes) > 0 {
            // Try all combinations of routes
            for _, leg1 := range fromViaRoutes {
                for _, leg2 := range viaToRoutes {
                    route := createInterchangeRoute(leg1, leg2, via)
                    if route != nil {
                        allRoutes = append(allRoutes, route)
                        log.Printf("Found valid interchange route via %s: %s -> %s", 
                            via, leg1.Trains[0].Name, leg2.Trains[0].Name)
                    }
                }
            }
        }

        if len(allRoutes) >= maxInterchangeRoutes {
            break
        }
    }

    // If still no routes found and we haven't reached max interchanges, try double interchange
    if len(allRoutes) == 0 {
        log.Printf("Trying double interchange routes")
        majorJunctions := getMajorJunctions()
        for i, via1 := range majorJunctions {
            if via1 == from || via1 == to {
                continue
            }
            
            for j := i + 1; j < len(majorJunctions); j++ {
                via2 := majorJunctions[j]
                if via2 == from || via2 == to || via2 == via1 {
                    continue
                }

                route := findDoubleInterchangeRoute(from, via1, via2, to)
                if route != nil {
                    allRoutes = append(allRoutes, route)
                    log.Printf("Found double interchange route via %s and %s", via1, via2)
                    
                    if len(allRoutes) >= maxInterchangeRoutes {
                        break
                    }
                }
            }
            
            if len(allRoutes) >= maxInterchangeRoutes {
                break
            }
        }
    }

    // Sort routes by duration and score
    sortedRoutes := sortRoutes(allRoutes)

    // Log summary
    log.Printf("Found %d total routes (%d direct, %d interchange) in %v", 
        len(sortedRoutes), 
        len(directRoutes),
        len(sortedRoutes)-len(directRoutes),
        time.Since(startTime))

    return sortedRoutes
}

// Modify the areStationsEquivalent function
func areStationsEquivalent(station1, station2 string) bool {
    if station1 == station2 {
        return true
    }

    // Check if stations are equivalent
    if equivalents, exists := equivalentStations[station1]; exists {
        for _, eq := range equivalents {
            if eq == station2 {
                return true
            }
        }
    }

    // Check reverse equivalence
    if equivalents, exists := equivalentStations[station2]; exists {
        for _, eq := range equivalents {
            if eq == station1 {
                return true
            }
        }
    }

    return false
}


// Add this helper function
func getEquivalentStations(stationCode string) []string {
    if equivalents, exists := equivalentStations[stationCode]; exists {
        return append(equivalents, stationCode)
    }
    return []string{stationCode}
}


// Modify findDirectRoutesWithStops function
func findDirectRoutesWithStops(from, to string) []*RouteResult {
    trainCache.RLock()
    defer trainCache.RUnlock()

    var directRoutes []*RouteResult
    log.Printf("Checking direct routes from %s to %s (including equivalents)", from, to)

    // Get all equivalent station codes
    fromStations := getEquivalentStations(from)
    toStations := getEquivalentStations(to)

    // Check all combinations of equivalent stations
    for _, fromStation := range fromStations {
        if routes, exists := trainCache.routes[fromStation]; exists {
            for _, routeList := range routes {
                for _, route := range routeList {
                    // Check if destination matches any equivalent station
                    for _, toStation := range toStations {
                        if route.ToStation == toStation {
                            directRoutes = append(directRoutes, &RouteResult{
                                Type:          "direct",
                                Trains:        []*TrainRoute{route},
                                TotalDuration: calculateDuration(route.Departure, route.Arrival),
                                TotalDistance: route.Distance,
                                Score:         calculateRouteScore(route),
                            })
                            log.Printf("Found direct train %d (%s) from %s to %s", 
                                route.TrainNumber, route.Name, fromStation, toStation)
                            continue
                        }

                        // Check intermediate stops
                        for _, stop := range route.Stops {
                            if stop.Station == toStation {
                                directRoute := createDirectRouteFromStop(route, fromStation, toStation, stop)
                                directRoutes = append(directRoutes, directRoute)
                                log.Printf("Found direct train through stops: %d (%s) from %s to %s", 
                                    route.TrainNumber, route.Name, fromStation, toStation)
                            }
                        }
                    }
                }
            }
        }
    }

    return directRoutes
}

func addStationGroup(group *StationGroup) {
    stationManager.Lock()
    defer stationManager.Unlock()

    groupID := generateGroupID()
    stationManager.groups[groupID] = group

    // Map each station to this group
    for station := range group.Stations {
        stationManager.stationMap[station] = groupID
    }
}

func getAllStationsWithCoordinates() []StationInfo {
    var stations []StationInfo
    collection := config.MongoDB.Collection("stations")
    ctx := context.Background()

    cursor, err := collection.Find(ctx, bson.M{
        "location": bson.M{"$exists": true},
    })
    if err != nil {
        log.Printf("Error fetching stations: %v", err)
        return stations
    }
    defer cursor.Close(ctx)

    for cursor.Next(ctx) {
        var station StationInfo
        if err := cursor.Decode(&station); err != nil {
            continue
        }
        stations = append(stations, station)
    }

    return stations
}

func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
    const R = 6371 // Earth's radius in kilometers

    // Convert degrees to radians
    lat1Rad := lat1 * math.Pi / 180
    lon1Rad := lon1 * math.Pi / 180
    lat2Rad := lat2 * math.Pi / 180
    lon2Rad := lon2 * math.Pi / 180

    // Haversine formula
    dlat := lat2Rad - lat1Rad
    dlon := lon2Rad - lon1Rad
    a := math.Sin(dlat/2)*math.Sin(dlat/2) +
        math.Cos(lat1Rad)*math.Cos(lat2Rad)*
            math.Sin(dlon/2)*math.Sin(dlon/2)
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
    distance := R * c

    return distance
}

func createOrUpdateStationGroup(station1, station2 StationInfo, distance float64) {
    stationManager.Lock()
    defer stationManager.Unlock()

    // Check if either station is already in a group
    group1ID := stationManager.stationMap[station1.Code]
    group2ID := stationManager.stationMap[station2.Code]

    if group1ID == "" && group2ID == "" {
        // Create new group
        group := &StationGroup{
            Stations: map[string]bool{
                station1.Code: true,
                station2.Code: true,
            },
            MainCode:  selectMainCode(station1, station2),
            Distance:  distance,
        }
        addStationGroup(group)
    } else if group1ID != "" && group2ID == "" {
        // Add station2 to station1's group
        stationManager.groups[group1ID].Stations[station2.Code] = true
        stationManager.stationMap[station2.Code] = group1ID
    } else if group1ID == "" && group2ID != "" {
        // Add station1 to station2's group
        stationManager.groups[group2ID].Stations[station1.Code] = true
        stationManager.stationMap[station1.Code] = group2ID
    }
    // If both stations are already in groups, do nothing
}

func createDirectRouteFromStop(route *TrainRoute, from, to string, destinationStop Stop) *RouteResult {
    directRoute := &TrainRoute{
        TrainNumber: route.TrainNumber,
        Name:        route.Name,
        Type:        route.Type,
        FromStation: from,
        ToStation:   to,
        Departure:   route.Departure,
        Arrival:     destinationStop.Arrival,
        Platform:    destinationStop.Platform,
        Classes:     route.Classes,
        Distance:    destinationStop.Distance,
        Stops:       filterStopsUpTo(route.Stops, from, to),
    }

    return &RouteResult{
        Type:          "direct",
        Trains:        []*TrainRoute{directRoute},
        TotalDuration: calculateDuration(directRoute.Departure, directRoute.Arrival),
        TotalDistance: directRoute.Distance,
        Score:         calculateRouteScore(directRoute),
    }
}

func generateGroupID() string {
    return primitive.NewObjectID().Hex()
}

func isTerminal(stationCode string) bool {
    majorTerminals := map[string]bool{
        // Delhi Region
        "NDLS": true, // New Delhi
        "DLI": true,  // Delhi Junction (Old Delhi)
        "DEE": true,  // Delhi Sarai Rohilla
        "ANVT": true, // Anand Vihar Terminal
        "ANVR": true, // Anand Vihar
        
        // Mumbai Region
        "BCT": true,  // Mumbai Central
        "CSMT": true, // Chhatrapati Shivaji Maharaj Terminus
        "BDTS": true, // Bandra Terminus
        "LTT": true,  // Lokmanya Tilak Terminus
        "PNVL": true, // Panvel
        
        // Kolkata Region
        "HWH": true,  // Howrah
        "SDAH": true, // Sealdah
        "KOAA": true, // Kolkata
        
        // Chennai Region
        "MAS": true,  // Chennai Central
        "MS": true,   // Chennai Egmore
        "TBM": true,  // Tambaram
        
        // Bengaluru Region
        "SBC": true,  // KSR Bengaluru
        "YPR": true,  // Yeshwantpur
        "BNCE": true, // Bangalore Cantonment
        
        // Hyderabad Region
        "SC": true,   // Secunderabad
        "HYB": true,  // Hyderabad
        "KCG": true,  // Kacheguda
        
        // Other Major Cities
        "PNBE": true, // Patna
        "MGS": true,  // Mughalsarai/DDU
        "CNB": true,  // Kanpur Central
        "LKO": true,  // Lucknow
        "ADI": true,  // Ahmedabad
        "BRC": true,  // Vadodara
        "PUNE": true, // Pune
        "NGP": true,  // Nagpur
        "BPL": true,  // Bhopal
        "JP": true,   // Jaipur
        "ASR": true,  // Amritsar
        "CDG": true,  // Chandigarh
        "BBS": true,  // Bhubaneswar
        "VSKP": true, // Visakhapatnam
        "SWV": true,  // Secunderabad Wagon Shop
        "TVC": true,  // Thiruvananthapuram
        "ERS": true,  // Ernakulam
        "MAQ": true,  // Mangalore
        "UBL": true,  // Hubballi
        
        // Major Junction Stations
        "GZB": true,  // Ghaziabad
        "ALD": true,  // Allahabad/Prayagraj
        "BSB": true,  // Varanasi
        "JHS": true,  // Jhansi
        "BZA": true,  // Vijayawada
        "GTL": true,  // Guntakal
        "SLI": true,  // Shimoga
        "RNC": true,  // Ranchi
        "BSP": true,  // Bilaspur
        "NJP": true,  // New Jalpaiguri
        "GHY": true,  // Guwahati
        "LMG": true,  // Lumding
        "GKP": true,  // Gorakhpur
        "KGP": true,  // Kharagpur
        "TATA": true, // Tatanagar
    }
    return majorTerminals[stationCode]
}

func selectMainCode(station1, station2 StationInfo) string {
    // Prefer the station with more connections or the one that's a major junction
    // This is a simplified version - you might want to add more sophisticated logic
    if isTerminal(station1.Code) {
        return station1.Code
    }
    if isTerminal(station2.Code) {
        return station2.Code
    }
    return station1.Code
}

func filterStopsUpTo(stops []Stop, from, to string) []Stop {
    var filteredStops []Stop
    var started bool
    
    for _, stop := range stops {
        if stop.Station == from {
            started = true
        }
        if started {
            filteredStops = append(filteredStops, stop)
        }
        if stop.Station == to {
            break
        }
    }
    return filteredStops
}

func findAllDirectRoutes(from, to string) []*RouteResult {
    trainCache.RLock()
    defer trainCache.RUnlock()

    var directRoutes []*RouteResult
    log.Printf("Searching for direct routes from %s to %s", from, to)

    // Check all routes starting from source station
    if routes, exists := trainCache.routes[from]; exists {
        for _, routeList := range routes {
            for _, route := range routeList {
                // First check if this is already a direct route to destination
                if route.ToStation == to {
                    directRoutes = append(directRoutes, &RouteResult{
                        Type:          "direct",
                        Trains:        []*TrainRoute{route},
                        TotalDuration: calculateDuration(route.Departure, route.Arrival),
                        TotalDistance: route.Distance,
                        Score:         calculateRouteScore(route),
                    })
                    continue
                }

                // Then check if this train continues to the destination
                isDirectToDestination := false
                var finalStop *Stop

                for _, stop := range route.Stops {
                    if stop.Station == to {
                        isDirectToDestination = true
                        finalStop = &stop
                        break
                    }
                }

                if isDirectToDestination && finalStop != nil {
                    // Create a modified route that shows the direct journey
                    modifiedRoute := &TrainRoute{
                        TrainNumber: route.TrainNumber,
                        Name:        route.Name,
                        Type:        route.Type,
                        FromStation: from,
                        ToStation:   to,
                        Departure:   route.Departure,
                        Arrival:     finalStop.Arrival,
                        Platform:    finalStop.Platform,
                        Classes:     route.Classes,
                        Distance:    finalStop.Distance,
                        Stops:      filterStops(route.Stops, from, to),
                    }

                    directRoutes = append(directRoutes, &RouteResult{
                        Type:          "direct",
                        Trains:        []*TrainRoute{modifiedRoute},
                        TotalDuration: calculateDuration(modifiedRoute.Departure, modifiedRoute.Arrival),
                        TotalDistance: modifiedRoute.Distance,
                        Score:         calculateRouteScore(modifiedRoute),
                    })
                    log.Printf("Found direct train %s (%d) from %s to %s", 
                        route.Name, route.TrainNumber, from, to)
                }
            }
        }
    }

    return directRoutes
}

func filterStops(stops []Stop, from, to string) []Stop {
    var filteredStops []Stop
    started := false

    for _, stop := range stops {
        if stop.Station == from {
            started = true
        }
        if started {
            filteredStops = append(filteredStops, stop)
        }
        if stop.Station == to {
            break
        }
    }
    return filteredStops
}

func findInterchangeRoute(from, via, to string) *RouteResult {
    // Find first leg
    fromViaRoutes := findDirectRoutes(from, via)
    if len(fromViaRoutes) == 0 {
        return nil
    }

    // Find second leg
    viaToRoutes := findDirectRoutes(via, to)
    if len(viaToRoutes) == 0 {
        return nil
    }

    // Try to find the best combination
    var bestRoute *RouteResult
    var bestScore float64

    for _, leg1 := range fromViaRoutes {
        for _, leg2 := range viaToRoutes {
            waitTime := calculateWaitingTime(
                leg1.Trains[0].Arrival,
                leg2.Trains[0].Departure,
            )

            if waitTime < minWaitingTime || waitTime > maxWaitingTime {
                continue
            }

            totalDuration := leg1.TotalDuration + leg2.TotalDuration + waitTime
            totalDistance := leg1.TotalDistance + leg2.TotalDistance
            score := calculateInterchangeScore(leg1, leg2, waitTime)

            if bestRoute == nil || score > bestScore {
                bestRoute = &RouteResult{
                    Type:   "single-interchange",
                    Trains: []*TrainRoute{leg1.Trains[0], leg2.Trains[0]},
                    Interchanges: []*Interchange{{
                        Station:     via,
                        WaitingTime: waitTime,
                        FromTrain:   leg1.Trains[0].TrainNumber,
                        ToTrain:     leg2.Trains[0].TrainNumber,
                        Platform:    leg2.Trains[0].Platform,
                    }},
                    TotalDuration: totalDuration,
                    TotalDistance: totalDistance,
                    Score:         score,
                }
                bestScore = score
            }
        }
    }

    return bestRoute
}

func createModifiedRoute(originalRoute *TrainRoute, destinationStation string) *TrainRoute {
    var destinationStop *Stop
    var modifiedStops []Stop

    // Find the destination stop and collect stops up to it
    for _, stop := range originalRoute.Stops {
        modifiedStops = append(modifiedStops, stop)
        if stop.Station == destinationStation {
            destinationStop = &stop
            break
        }
    }

    if destinationStop == nil {
        return nil
    }

    // Create modified route
    return &TrainRoute{
        TrainNumber: originalRoute.TrainNumber,
        Name:        originalRoute.Name,
        Type:        originalRoute.Type,
        FromStation: originalRoute.FromStation,
        ToStation:   destinationStation,
        Departure:   originalRoute.Departure,
        Arrival:     destinationStop.Arrival,
        Platform:    destinationStop.Platform,
        Classes:     originalRoute.Classes,
        Distance:    destinationStop.Distance,
        Stops:       modifiedStops,
    }
}

func calculateInterchangeScore(leg1, leg2 *RouteResult, waitTime time.Duration) float64 {
    // Base scores for each leg
    score1 := leg1.Score
    score2 := leg2.Score
    
    // Average base score
    baseScore := (score1 + score2) / 2
    
    // Penalties
    waitingPenalty := (waitTime.Hours() / maxWaitingTime.Hours()) * 0.2
    interchangePenalty := 0.1 // Base penalty for interchange
    
    return baseScore * (1 - waitingPenalty - interchangePenalty)
}

func createInterchangeRoute(leg1, leg2 *RouteResult, via string) *RouteResult {
    // Add debug logging
    log.Printf("Attempting to create interchange route via %s", via)
    log.Printf("Leg1: %s -> %s (%s to %s)", 
        leg1.Trains[0].FromStation, leg1.Trains[0].ToStation,
        leg1.Trains[0].Departure, leg1.Trains[0].Arrival)
    log.Printf("Leg2: %s -> %s (%s to %s)", 
        leg2.Trains[0].FromStation, leg2.Trains[0].ToStation,
        leg2.Trains[0].Departure, leg2.Trains[0].Arrival)

    waitTime := calculateWaitingTime(
        leg1.Trains[0].Arrival,
        leg2.Trains[0].Departure,
    )

    log.Printf("Calculated waiting time: %v", waitTime)

    // More lenient waiting time check
    if waitTime < minWaitingTime {
        log.Printf("Waiting time too short: %v", waitTime)
        return nil
    }

    route := &RouteResult{
        Type:   "single-interchange",
        Trains: []*TrainRoute{leg1.Trains[0], leg2.Trains[0]},
        Interchanges: []*Interchange{{
            Station:     via,
            WaitingTime: waitTime,
            FromTrain:   leg1.Trains[0].TrainNumber,
            ToTrain:     leg2.Trains[0].TrainNumber,
            Platform:    leg2.Trains[0].Platform,
        }},
        TotalDuration: leg1.TotalDuration + leg2.TotalDuration + waitTime,
        TotalDistance: leg1.TotalDistance + leg2.TotalDistance,
        Score:         calculateRouteScore(leg1.Trains[0]) * 0.8,
    }

    log.Printf("Successfully created interchange route via %s", via)
    return route
}

func findDoubleInterchangeRoute(from, via1, via2, to string) *RouteResult {
    // Find first leg
    leg1 := findDirectRoutes(from, via1)
    if len(leg1) == 0 {
        return nil
    }

    // Find second leg
    leg2 := findDirectRoutes(via1, via2)
    if len(leg2) == 0 {
        return nil
    }

    // Find third leg
    leg3 := findDirectRoutes(via2, to)
    if len(leg3) == 0 {
        return nil
    }

    // Calculate waiting times
    wait1 := calculateWaitingTime(leg1[0].Trains[0].Arrival, leg2[0].Trains[0].Departure)
    wait2 := calculateWaitingTime(leg2[0].Trains[0].Arrival, leg3[0].Trains[0].Departure)

    if wait1 < minWaitingTime || wait1 > maxWaitingTime || 
       wait2 < minWaitingTime || wait2 > maxWaitingTime {
        return nil
    }

    return &RouteResult{
        Type:   "double-interchange",
        Trains: []*TrainRoute{leg1[0].Trains[0], leg2[0].Trains[0], leg3[0].Trains[0]},
        Interchanges: []*Interchange{
            {
                Station:     via1,
                WaitingTime: wait1,
                FromTrain:   leg1[0].Trains[0].TrainNumber,
                ToTrain:     leg2[0].Trains[0].TrainNumber,
                Platform:    leg2[0].Trains[0].Platform,
            },
            {
                Station:     via2,
                WaitingTime: wait2,
                FromTrain:   leg2[0].Trains[0].TrainNumber,
                ToTrain:     leg3[0].Trains[0].TrainNumber,
                Platform:    leg3[0].Trains[0].Platform,
            },
        },
        TotalDuration: leg1[0].TotalDuration + leg2[0].TotalDuration + leg3[0].TotalDuration + wait1 + wait2,
        TotalDistance: leg1[0].TotalDistance + leg2[0].TotalDistance + leg3[0].TotalDistance,
        Score:         calculateMultiLegScore(leg1[0], leg2[0], leg3[0], wait1, wait2),
    }
}

func findAllInterchangePaths(from, to string, visited map[string]bool, depth int) []InterchangePath {
    if depth >= maxInterchanges {
        return nil
    }

    visited[from] = true
    defer delete(visited, from)

    var paths []InterchangePath
    connectedStations := getConnectedStations(from)

    // Try each connected station as potential interchange
    for _, via := range connectedStations {
        if visited[via] {
            continue
        }

        // Check direct route from via to destination
        if directRoutes := findDirectRoutes(via, to); len(directRoutes) > 0 {
            // Found a path with one interchange
            leg1 := findDirectRoutes(from, via)
            if len(leg1) == 0 {
                continue
            }

            path := createInterchangePath(leg1[0], directRoutes[0], via)
            if path != nil {
                paths = append(paths, *path)
            }
        }

        // Recursively try more interchanges
        subPaths := findAllInterchangePaths(via, to, visited, depth+1)
        for _, subPath := range subPaths {
            leg1 := findDirectRoutes(from, via)
            if len(leg1) == 0 {
                continue
            }

            newPath := extendInterchangePath(leg1[0], subPath, via)
            if newPath != nil {
                paths = append(paths, *newPath)
            }
        }
    }

    return paths
}

func createInterchangePath(leg1, leg2 *RouteResult, via string) *InterchangePath {
    waitTime := calculateWaitingTime(
        leg1.Trains[0].Arrival,
        leg2.Trains[0].Departure,
    )

    if waitTime < minWaitingTime || waitTime > maxWaitingTime {
        return nil
    }

    return &InterchangePath{
        stations: []string{leg1.Trains[0].FromStation, via, leg2.Trains[0].ToStation},
        trains:   append(leg1.Trains, leg2.Trains...),
        waitTime: waitTime,
        distance: leg1.TotalDistance + leg2.TotalDistance,
        score:    calculatePathScore(leg1, leg2, waitTime),
    }
}

func extendInterchangePath(firstLeg *RouteResult, existingPath InterchangePath, via string) *InterchangePath {
    waitTime := calculateWaitingTime(
        firstLeg.Trains[0].Arrival,
        existingPath.trains[0].Departure,
    )

    if waitTime < minWaitingTime || waitTime > maxWaitingTime {
        return nil
    }

    newStations := append([]string{firstLeg.Trains[0].FromStation}, existingPath.stations...)
    newTrains := append([]*TrainRoute{firstLeg.Trains[0]}, existingPath.trains...)

    return &InterchangePath{
        stations: newStations,
        trains:   newTrains,
        waitTime: existingPath.waitTime + waitTime,
        distance: firstLeg.TotalDistance + existingPath.distance,
        score:    calculateExtendedPathScore(firstLeg, existingPath, waitTime),
    }
}

func convertPathToRouteResult(path InterchangePath) *RouteResult {
    interchanges := make([]*Interchange, len(path.stations)-2)
    totalDuration := time.Duration(0)
    
    // Calculate total duration and create interchanges
    for i := 0; i < len(path.trains)-1; i++ {
        currentTrain := path.trains[i]
        nextTrain := path.trains[i+1]
        
        waitTime := calculateWaitingTime(currentTrain.Arrival, nextTrain.Departure)
        totalDuration += calculateDuration(currentTrain.Departure, currentTrain.Arrival)
        totalDuration += waitTime

        interchanges[i] = &Interchange{
            Station:     path.stations[i+1],
            WaitingTime: waitTime,
            FromTrain:   currentTrain.TrainNumber,
            ToTrain:     nextTrain.TrainNumber,
            Platform:    nextTrain.Platform,
        }
    }

    // Add last train duration
    lastTrain := path.trains[len(path.trains)-1]
    totalDuration += calculateDuration(lastTrain.Departure, lastTrain.Arrival)

    return &RouteResult{
        Type:          fmt.Sprintf("interchange-%d", len(interchanges)),
        Trains:        path.trains,
        Interchanges:  interchanges,
        TotalDuration: totalDuration,
        TotalDistance: path.distance,
        Score:         path.score,
    }
}

func calculatePathScore(leg1, leg2 *RouteResult, waitTime time.Duration) float64 {
    baseScore := (leg1.Score + leg2.Score) / 2
    
    // Penalties
    waitingPenalty := (waitTime.Hours() / maxWaitingTime.Hours()) * 0.2
    interchangePenalty := 0.1 // Base penalty for one interchange
    
    return baseScore * (1 - waitingPenalty - interchangePenalty)
}

func calculateExtendedPathScore(firstLeg *RouteResult, existingPath InterchangePath, additionalWaitTime time.Duration) float64 {
    // Base score calculation
    totalLegs := float64(len(existingPath.trains) + 1)
    baseScore := (firstLeg.Score + existingPath.score*(totalLegs-1)) / totalLegs
    
    // Additional penalties for multiple interchanges
    waitingPenalty := ((existingPath.waitTime + additionalWaitTime).Hours() / maxWaitingTime.Hours()) * 0.2
    interchangePenalty := 0.1 * float64(len(existingPath.stations)-1) // Penalty increases with more interchanges
    
    return baseScore * (1 - waitingPenalty - interchangePenalty)
}

func formatSingleRoute(route *RouteResult) map[string]interface{} {
    result := map[string]interface{}{
        "type":           route.Type,
        "total_duration": formatDuration(route.TotalDuration),
        "total_distance": fmt.Sprintf("%.1f KM", route.TotalDistance),
        "trains":         formatTrains(route.Trains),
    }

    if strings.HasPrefix(route.Type, "interchange") {
        result["interchanges"] = formatInterchanges(route.Interchanges)
        result["interchange_count"] = len(route.Interchanges)
        result["total_waiting_time"] = formatDuration(calculateTotalWaitingTime(route.Interchanges))
    }

    return result
}

func calculateTotalWaitingTime(interchanges []*Interchange) time.Duration {
    total := time.Duration(0)
    for _, ic := range interchanges {
        total += ic.WaitingTime
    }
    return total
}

// Enhanced getMajorJunctions to include more strategic stations
func getMajorJunctions() []string {
    return []string{
        // Major Metro Cities
        "NDLS", "BCT", "MAS", "HWH", "SBC", 
        // Important Junctions
        "PUNE", "NGP", "BZA", "GKP", "ADI",
        "JP", "CNB", "ALD", "BSB", "PNBE",
        // Regional Hubs
        "NZM", "DEE", "DLI", // Delhi area
        "LKO", "AGC", "JHS", // UP region
        "BPL", "JBP", "BRC", // Central region
        "VSKP", "SC", "BBS",  // South-East
        "MAO", "MAQ", "TVC",  // West coast
        "ASR", "UMB", "LDH",  // North region
        "RNC", "KGP", "TATA", // East region
    }
}

func getConnectedStations(station string) []string {
    trainCache.RLock()
    defer trainCache.RUnlock()

    if routes, exists := trainCache.routes[station]; exists {
        // Create a map to ensure unique stations
        uniqueStations := make(map[string]bool)
        
        // Add directly connected stations
        for dest := range routes {
            uniqueStations[dest] = true
        }
        
        // Convert map to slice
        connected := make([]string, 0, len(uniqueStations))
        for station := range uniqueStations {
            connected = append(connected, station)
        }
        
        // Sort for consistent results
        sort.Strings(connected)
        return connected
    }
    return nil
}

// Helper functions
func extractStationCode(stationName string) string {
    parts := strings.Split(stationName, " - ")
    if len(parts) > 1 {
        code := strings.TrimSpace(parts[0])
        return code
    }
    return ""
}

func formatRouteResponse(routes []*RouteResult, from, to string, startTime time.Time) map[string]interface{} {
    fromInfo := getStationInfo(from)
    toInfo := getStationInfo(to)

    // Separate direct and interchange routes
    var directRoutes, interchangeRoutes []map[string]interface{}
    
    for _, route := range routes {
        formattedRoute := formatSingleRoute(route)
        if route.Type == "direct" {
            directRoutes = append(directRoutes, formattedRoute)
        } else {
            interchangeRoutes = append(interchangeRoutes, formattedRoute)
        }
    }

    response := map[string]interface{}{
        "from_station": map[string]interface{}{
            "code": from,
            "name": fromInfo.Name,
            "city": fromInfo.City,
        },
        "to_station": map[string]interface{}{
            "code": to,
            "name": toInfo.Name,
            "city": toInfo.City,
        },
        "has_routes":     len(routes) > 0,
        "direct_routes":  directRoutes,
        "interchange_routes": interchangeRoutes,
        "total_direct":   len(directRoutes),
        "total_interchange": len(interchangeRoutes),
        "search_time_ms": time.Since(startTime).Milliseconds(),
        "timestamp":      time.Now().Format(time.RFC3339),
    }

    return response
}

func InitializeTrainSystem() error {
    log.Println("Starting train system initialization...")
    startTime := time.Now()

    collection := config.MongoDB.Collection("trains")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    // Initialize station groups concurrently
    var wg sync.WaitGroup
    errChan := make(chan error, 1)
    
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := initializeStationGroups(); err != nil {
            log.Printf("Warning: Failed to initialize station groups: %v", err)
            errChan <- err
        }
    }()

    // Initialize route graph
    routeGraphChan := make(chan error, 1)
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := initializeRouteGraph(ctx); err != nil {
            log.Printf("Warning: Failed to initialize route graph: %v", err)
            routeGraphChan <- err
            return
        }
        routeGraphChan <- nil
    }()

    // Use a more efficient batch size
    batchSize := 200
    maxConcurrentBatches := 10
    sem := make(chan struct{}, maxConcurrentBatches)
    batchErrChan := make(chan error, maxConcurrentBatches)
    
    // Get total count for progress tracking
    total, err := collection.CountDocuments(ctx, bson.M{})
    if err != nil {
        return fmt.Errorf("failed to count trains: %v", err)
    }

    // Create progress tracker
    processed := int64(0)
    progressTicker := time.NewTicker(2 * time.Second)
    defer progressTicker.Stop()

    go func() {
        for range progressTicker.C {
            percentage := float64(processed) / float64(total) * 100
            log.Printf("Processed %d/%d trains (%.1f%%)", processed, total, percentage)
        }
    }()

    // Process trains in batches
    cursor, err := collection.Find(ctx, bson.M{}, options.Find().SetBatchSize(int32(batchSize)))
    if err != nil {
        return fmt.Errorf("failed to get trains cursor: %v", err)
    }
    defer cursor.Close(ctx)

    var batch []Train
    for cursor.Next(ctx) {
        var train Train
        if err := cursor.Decode(&train); err != nil {
            log.Printf("Warning: Failed to decode train: %v", err)
            continue
        }

        batch = append(batch, train)
        if len(batch) >= batchSize {
            // Process batch concurrently
            currentBatch := batch
            sem <- struct{}{}
            wg.Add(1)
            go func() {
                defer func() {
                    <-sem
                    wg.Done()
                }()
                if err := processBatchWithContext(ctx, currentBatch); err != nil {
                    batchErrChan <- err
                }
                atomic.AddInt64(&processed, int64(len(currentBatch)))
            }()
            batch = nil
        }
    }

    // Process remaining trains
    if len(batch) > 0 {
        sem <- struct{}{}
        wg.Add(1)
        go func() {
            defer func() {
                <-sem
                wg.Done()
            }()
            if err := processBatchWithContext(ctx, batch); err != nil {
                batchErrChan <- err
            }
            atomic.AddInt64(&processed, int64(len(batch)))
        }()
    }

    // Wait for all goroutines to finish
    wg.Wait()
    close(batchErrChan)
    close(errChan)
    close(routeGraphChan)

    // Check for errors
    var errors []error
    
    // Check station group initialization error
    if err := <-errChan; err != nil {
        errors = append(errors, fmt.Errorf("station group initialization error: %v", err))
    }

    // Check route graph initialization error
    if err := <-routeGraphChan; err != nil {
        errors = append(errors, fmt.Errorf("route graph initialization error: %v", err))
    }
    
    // Check batch processing errors
    for err := range batchErrChan {
        errors = append(errors, err)
    }

    if len(errors) > 0 {
        log.Printf("Warning: Encountered %d errors during train initialization", len(errors))
        for _, err := range errors {
            log.Printf("Error: %v", err)
        }
        // Return the first error but log all of them
        return errors[0]
    }

    duration := time.Since(startTime)
    log.Printf("Train system initialization completed in %v", duration)
    
    // Log cache statistics
    trainCache.RLock()
    stationCount := len(trainCache.routes)
    var routeCount int
    for _, routes := range trainCache.routes {
        for _, trainList := range routes {
            routeCount += len(trainList)
        }
    }
    trainCache.RUnlock()
    log.Printf("Cache initialized with %d stations and %d routes", stationCount, routeCount)
    
    return nil
}

func processBatchWithContext(ctx context.Context, trains []Train) error {
    for _, train := range trains {
        if len(train.Schedule) == 0 {
            // Skip trains with empty schedules instead of logging
            continue
        }

        // Process valid trains
        if err := processTrainSchedule(&train); err != nil {
            return fmt.Errorf("error processing train %d: %v", train.TrainNumber, err)
        }
    }
    return nil
}

func logSampleRoutes() {
    trainCache.RLock()
    defer trainCache.RUnlock()

    log.Printf("Cache contains routes for %d stations", len(trainCache.routes))
    
    // Log a few sample routes
    count := 0
    for from, routes := range trainCache.routes {
        if count >= 3 {
            break
        }
        log.Printf("Sample routes from %s to %d destinations", from, len(routes))
        count++
    }
}

func getStationFullInfo(stationCode string) *StationInfo {
    // First check cache
    stationCache.RLock()
    if info, exists := stationCache.codes[stationCode]; exists {
        stationCache.RUnlock()
        return info
    }
    stationCache.RUnlock()

    // If not in cache, fetch from database
    collection := config.MongoDB.Collection("trains")
    ctx := context.Background()

    pipeline := []bson.M{
        {
            "$unwind": "$schedule_table",
        },
        {
            "$match": bson.M{
                "schedule_table.station": bson.M{
                    "$regex": primitive.Regex{
                        Pattern: fmt.Sprintf("^%s\\s*-", stationCode),
                        Options: "i",
                    },
                },
            },
        },
        {
            "$limit": 1,
        },
    }

    cursor, err := collection.Aggregate(ctx, pipeline)
    if err != nil {
        return &StationInfo{
            Code: stationCode,
            Name: stationCode,
        }
    }
    defer cursor.Close(ctx)

    if cursor.Next(ctx) {
        var result struct {
            Schedule struct {
                Station string `bson:"station"`
            } `bson:"schedule_table"`
        }
        if err := cursor.Decode(&result); err == nil {
            parts := strings.Split(result.Schedule.Station, " - ")
            if len(parts) > 1 {
                info := &StationInfo{
                    Code: stationCode,
                    Name: strings.TrimSpace(parts[1]),
                }
                // Cache the result
                stationCache.Lock()
                stationCache.codes[stationCode] = info
                stationCache.Unlock()
                return info
            }
        }
    }

    return &StationInfo{
        Code: stationCode,
        Name: stationCode,
    }
}

func formatTrains(trains []*TrainRoute) []map[string]interface{} {
    result := make([]map[string]interface{}, len(trains))
    for i, train := range trains {
        fromStationInfo := getStationFullInfo(train.FromStation)
        toStationInfo := getStationFullInfo(train.ToStation)
        
        result[i] = map[string]interface{}{
            "train_number": train.TrainNumber,
            "name":        train.Name,
            "type":        train.Type,
            "from": map[string]interface{}{
                "code": train.FromStation,
                "name": fromStationInfo.Name,
            },
            "to": map[string]interface{}{
                "code": toStationInfo.Code,
                "name": toStationInfo.Name,
            },
            "departure":   train.Departure,
            "arrival":     train.Arrival,
            "platform":    train.Platform,
            "classes":     train.Classes,
            "stops":       formatStopsWithNames(train.Stops),
        }
    }
    return result
}

func formatStopsWithNames(stops []Stop) []map[string]interface{} {
    result := make([]map[string]interface{}, len(stops))
    for i, stop := range stops {
        stationInfo := getStationFullInfo(stop.Station)
        result[i] = map[string]interface{}{
            "station": map[string]interface{}{
                "code": stop.Station,
                "name": stationInfo.Name,
            },
            "arrival":   stop.Arrival,
            "departure": stop.Departure,
            "platform": stop.Platform,
            "distance": fmt.Sprintf("%.1f KM", stop.Distance),
        }
    }
    return result
}

func formatInterchanges(interchanges []*Interchange) []map[string]interface{} {
    result := make([]map[string]interface{}, len(interchanges))
    for i, ic := range interchanges {
        stationInfo := getStationFullInfo(ic.Station)
        result[i] = map[string]interface{}{
            "station": map[string]interface{}{
                "code": ic.Station,
                "name": stationInfo.Name,
            },
            "waiting_time": formatDuration(ic.WaitingTime),
            "from_train":   ic.FromTrain,
            "to_train":     ic.ToTrain,
            "platform":     ic.Platform,
        }
    }
    return result
}

func getStationInfo(stationCode string) *StationInfo {
    stationCache.RLock()
    defer stationCache.RUnlock()
    
    if info, exists := stationCache.codes[stationCode]; exists {
        return info
    }
    
    return &StationInfo{
        Code: stationCode,
        Name: stationCode,
    }
}

func initializeRouteGraph(ctx context.Context) error {
    log.Println("Starting route graph initialization...")
    startTime := time.Now()
    
    collection := config.MongoDB.Collection("trains")
    
    // Use a larger batch size for efficiency
    findOptions := options.Find().SetBatchSize(500)
    
    // Add retry logic for cursor creation
    var cursor *mongo.Cursor
    var err error
    
    for retries := 0; retries < 3; retries++ {
        cursor, err = collection.Find(ctx, bson.M{}, findOptions)
        if err == nil {
            break
        }
        log.Printf("Attempt %d: Failed to create cursor: %v", retries+1, err)
        time.Sleep(2 * time.Second)
    }
    
    if err != nil {
        return fmt.Errorf("failed to create cursor after retries: %v", err)
    }
    defer cursor.Close(ctx)

    // Initialize cache with estimated capacity
    trainCache.Lock()
    trainCache.routes = make(map[string]map[string][]*TrainRoute, 5000) // Pre-allocate for ~5000 stations
    trainCache.expiry = time.Now().Add(24 * time.Hour) // Cache for 24 hours
    trainCache.Unlock()

    // Process trains in batches with progress tracking
    batchSize := 500
    processed := 0
    var total int64 = 0
    batch := make([]Train, 0, batchSize)
    
    // Count total trains for progress tracking
    if total, err = collection.CountDocuments(ctx, bson.M{}); err != nil {
        log.Printf("Warning: Failed to count total trains: %v", err)
    }

    // Create progress ticker
    progressTicker := time.NewTicker(2 * time.Second)
    defer progressTicker.Stop()

    // Start progress reporting goroutine
    go func() {
        for range progressTicker.C {
            if total > 0 {
                percentage := float64(processed) * 100 / float64(total)
                log.Printf("Route graph: processed %d/%d trains (%.1f%%)", processed, total, percentage)
            } else {
                log.Printf("Route graph: processed %d trains", processed)
            }
        }
    }()

    for {
        // Check context cancellation
        if ctx.Err() != nil {
            return fmt.Errorf("context cancelled: %v", ctx.Err())
        }

        // Add retry logic for cursor.Next()
        var hasNext bool
        for retries := 0; retries < 3; retries++ {
            if hasNext = cursor.Next(ctx); hasNext || ctx.Err() != nil {
                break
            }
            log.Printf("Attempt %d: No documents returned, retrying...", retries+1)
            time.Sleep(2 * time.Second)
        }

        if err := cursor.Err(); err != nil {
            return fmt.Errorf("cursor error: %v", err)
        }
        
        if !hasNext {
            break
        }

        var train Train
        if err := cursor.Decode(&train); err != nil {
            log.Printf("Warning: Error decoding train: %v", err)
            continue
        }
        
        // Skip trains with empty schedules
        if len(train.Schedule) == 0 {
            processed++
            continue
        }

        batch = append(batch, train)
        
        if len(batch) >= batchSize {
            if err := processBatchForGraph(batch); err != nil {
                log.Printf("Warning: Error processing batch: %v", err)
            }
            processed += len(batch)
            batch = batch[:0]
        }
    }

    // Process remaining trains
    if len(batch) > 0 {
        if err := processBatchForGraph(batch); err != nil {
            log.Printf("Warning: Error processing final batch: %v", err)
        }
        processed += len(batch)
    }

    duration := time.Since(startTime)
    trainCache.RLock()
    stationCount := len(trainCache.routes)
    var routeCount int
    for _, routes := range trainCache.routes {
        for _, trainList := range routes {
            routeCount += len(trainList)
        }
    }
    trainCache.RUnlock()

    log.Printf("Route graph initialization completed in %v: %d stations, %d routes", 
        duration, stationCount, routeCount)
    return nil
}

func processBatchForGraph(trains []Train) error {
    for _, train := range trains {
        if err := processTrainForGraph(&train); err != nil {
            return fmt.Errorf("error processing train %d: %v", train.TrainNumber, err)
        }
    }
    return nil
}

func processTrainForGraph(train *Train) error {
    if train == nil || len(train.Schedule) == 0 {
        return nil
    }

    trainCache.Lock()
    defer trainCache.Unlock()

    for i := 0; i < len(train.Schedule)-1; i++ {
        fromStation := formatStationCode(train.Schedule[i].Station)
        if fromStation == "" {
            continue
        }

        // Initialize the map for this station if it doesn't exist
        if trainCache.routes[fromStation] == nil {
            trainCache.routes[fromStation] = make(map[string][]*TrainRoute)
        }

        for j := i + 1; j < len(train.Schedule); j++ {
            toStation := formatStationCode(train.Schedule[j].Station)
            if toStation == "" || fromStation == toStation {
                continue
            }

            route := &TrainRoute{
                TrainNumber: train.TrainNumber,
                Name:       train.Title,
                Type:       train.Type,
                FromStation: fromStation,
                ToStation:  toStation,
                Departure:  train.Schedule[i].Departure,
                Arrival:   train.Schedule[j].Arrival,
                Platform:  train.Schedule[j].Platform,
                Classes:   train.Classes,
                Distance:  utils.ParseDistance(train.Schedule[j].Distance),
                Stops:     convertToStops(train.Schedule[i:j+1]),
            }

            trainCache.routes[fromStation][toStation] = append(
                trainCache.routes[fromStation][toStation], route)
        }
    }

    return nil
}

func setCache(key string, data interface{}) {
    trainCache.Lock()
    defer trainCache.Unlock()

    if trainCache.expiry.Before(time.Now()) {
        trainCache.routes = make(map[string]map[string][]*TrainRoute)
        trainCache.expiry = time.Now().Add(cacheDuration)
    }

    if routes, ok := data.([]*RouteResult); ok {
        parts := strings.Split(key, "_")
        if len(parts) == 2 {
            fromStation := parts[0]
            toStation := parts[1]
            
            if trainCache.routes[fromStation] == nil {
                trainCache.routes[fromStation] = make(map[string][]*TrainRoute)
            }
            
            // Store both direct and interchange routes
            trainRoutes := make([]*TrainRoute, 0)
            for _, route := range routes {
                trainRoutes = append(trainRoutes, route.Trains...)
            }
            
            trainCache.routes[fromStation][toStation] = trainRoutes
        }
    }
}

func cleanupCaches() {
    now := time.Now()
    
    // Cleanup train cache
    trainCache.Lock()
    if trainCache.expiry.Before(now) {
        trainCache.routes = make(map[string]map[string][]*TrainRoute)
        trainCache.expiry = now.Add(cacheDuration)
        log.Println("Train cache cleaned up")
    }
    trainCache.Unlock()

    // Cleanup station cache
    stationCache.Lock()
    if stationCache.expiry.Before(now) {
        stationCache.codes = make(map[string]*StationInfo)
        stationCache.names = make(map[string]string)
        stationCache.expiry = now.Add(cacheDuration)
        log.Println("Station cache cleaned up")
    }
    stationCache.Unlock()
}

func processTrainSchedule(train *Train) error {
    if train == nil {
        return fmt.Errorf("train is nil")
    }

    if len(train.Schedule) == 0 {
        return fmt.Errorf("train schedule is empty")
    }

    for i := 0; i < len(train.Schedule)-1; i++ {
        fromStation := formatStationCode(train.Schedule[i].Station)
        
        for j := i + 1; j < len(train.Schedule); j++ {
            toStation := formatStationCode(train.Schedule[j].Station)
            
            // Skip invalid station combinations
            if fromStation == "" || toStation == "" || fromStation == toStation {
                continue
            }

            route := &TrainRoute{
                TrainNumber: train.TrainNumber,
                Name:       train.Title,
                Type:       train.Type,
                FromStation: fromStation,
                ToStation:  toStation,
                Departure:  train.Schedule[i].Departure,
                Arrival:   train.Schedule[j].Arrival,
                Platform:  train.Schedule[j].Platform,
                Classes:   train.Classes,
                Distance:  utils.ParseDistance(train.Schedule[j].Distance),
                Stops:     convertToStops(train.Schedule[i:j+1]),
            }

            trainCache.Lock()
            // Initialize nested maps if needed
            if trainCache.routes[fromStation] == nil {
                trainCache.routes[fromStation] = make(map[string][]*TrainRoute)
            }

            // Add route to cache
            trainCache.routes[fromStation][toStation] = append(
                trainCache.routes[fromStation][toStation], route)
            trainCache.Unlock()
        }
    }

    return nil
}

func isViableInterchange(fromTrain, toTrain *TrainRoute) bool {
    waitTime := calculateWaitingTime(fromTrain.Arrival, toTrain.Departure)
    if waitTime < minWaitingTime || waitTime > maxWaitingTime {
        return false
    }

    // Check if trains run on same day or consecutive days
    fromArr, _ := time.Parse("15:04", fromTrain.Arrival)
    toDep, _ := time.Parse("15:04", toTrain.Departure)
    
    if toDep.Before(fromArr) {
        toDep = toDep.Add(24 * time.Hour)
    }
    
    timeDiff := toDep.Sub(fromArr)
    return timeDiff <= maxWaitingTime
}

func calculateRouteScore(route *TrainRoute) float64 {
    duration := calculateDuration(route.Departure, route.Arrival)
    if duration == 0 {
        return 0
    }

    durationHours := duration.Hours()
    speed := route.Distance / durationHours
    
    // Enhanced scoring components
    speedScore := (speed / 100.0) * 0.5    // Speed weight reduced to 50%
    distanceScore := (route.Distance / maxDistance) * 0.3  // Distance weight 30%
    typeScore := getTrainTypeScore(route.Type) * 0.2      // Train type weight 20%
    
    return speedScore + distanceScore + typeScore
}

func getTrainTypeScore(trainType string) float64 {
    switch strings.ToUpper(trainType) {
    case "RAJDHANI", "SHATABDI", "VANDE BHARAT":
        return 1.0
    case "DURONTO", "SUPERFAST":
        return 0.8
    case "EXPRESS":
        return 0.6
    default:
        return 0.4
    }
}

func sortRoutes(routes []*RouteResult) []*RouteResult {
    sort.Slice(routes, func(i, j int) bool {
        // Prioritize direct routes
        if routes[i].Type == "direct" && routes[j].Type != "direct" {
            return true
        }
        if routes[i].Type != "direct" && routes[j].Type == "direct" {
            return false
        }

        // For interchange routes, prefer fewer interchanges
        if strings.HasPrefix(routes[i].Type, "interchange") && 
           strings.HasPrefix(routes[j].Type, "interchange") {
            iCount := len(routes[i].Interchanges)
            jCount := len(routes[j].Interchanges)
            if iCount != jCount {
                return iCount < jCount
            }
        }

        // If same type or interchange count, sort by score
        if routes[i].Score != routes[j].Score {
            return routes[i].Score > routes[j].Score
        }

        // If scores are equal, prefer shorter duration
        return routes[i].TotalDuration < routes[j].TotalDuration
    })

    // Limit the number of interchange routes
    directCount := 0
    interchangeCount := 0
    result := make([]*RouteResult, 0)

    for _, route := range routes {
        if route.Type == "direct" {
            if directCount < maxRoutes {
                result = append(result, route)
                directCount++
            }
        } else {
            if interchangeCount < maxInterchangeRoutes {
                result = append(result, route)
                interchangeCount++
            }
        }

        if directCount >= maxRoutes && interchangeCount >= maxInterchangeRoutes {
            break
        }
    }

    return result
}

func sendErrorResponse(w http.ResponseWriter, message string, code int) {
    log.Printf("Error: %s (Code: %d)", message, code)
    
    response := map[string]interface{}{
        "error":      message,
        "code":       code,
        "status":     http.StatusText(code),
        "timestamp":  time.Now().Format(time.RFC3339),
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(response)
}

func findDirectRoutes(from, to string) []*RouteResult {
    trainCache.RLock()
    routes, exists := trainCache.routes[from][to]
    trainCache.RUnlock()

    if !exists || len(routes) == 0 {
        return nil
    }

    results := make([]*RouteResult, 0, len(routes))
    for _, route := range routes {
        results = append(results, &RouteResult{
            Type:          "direct",
            Trains:        []*TrainRoute{route},
            TotalDuration: calculateDuration(route.Departure, route.Arrival),
            TotalDistance: route.Distance,
            Score:         calculateRouteScore(route),
        })
    }

    return results
}

func calculateDuration(departure, arrival string) time.Duration {
    if departure == "" || arrival == "" {
        return 0
    }

    dept, err := time.Parse("15:04", departure)
    if err != nil {
        return 0
    }

    arr, err := time.Parse("15:04", arrival)
    if err != nil {
        return 0
    }

    if arr.Before(dept) {
        arr = arr.Add(24 * time.Hour)
    }

    return arr.Sub(dept)
}

func convertTo24Hour(timeStr string) string {
    // Remove any extra spaces
    timeStr = strings.TrimSpace(timeStr)
    
    // Parse 12-hour format
    t, err := time.Parse("03:04 PM", timeStr)
    if err != nil {
        t, err = time.Parse("03:04 AM", timeStr)
        if err != nil {
            return timeStr // Return original if parsing fails
        }
    }
    
    // Convert to 24-hour format
    return t.Format("15:04")
}

func calculateWaitingTime(arrival, departure string) time.Duration {
    if arrival == "" || departure == "" {
        return 0
    }

    // Convert times to 24-hour format
    arrival24 := convertTo24Hour(arrival)
    departure24 := convertTo24Hour(departure)

    // Parse times
    arr, err := time.Parse("15:04", arrival24)
    if err != nil {
        log.Printf("Error parsing arrival time %s (converted from %s): %v", arrival24, arrival, err)
        return 0
    }

    dept, err := time.Parse("15:04", departure24)
    if err != nil {
        log.Printf("Error parsing departure time %s (converted from %s): %v", departure24, departure, err)
        return 0
    }

    // Convert to minutes since midnight for easier comparison
    arrMins := arr.Hour()*60 + arr.Minute()
    deptMins := dept.Hour()*60 + dept.Minute()

    // Calculate waiting time considering 24-hour wraparound
    waitMins := deptMins - arrMins
    if waitMins < 0 {
        waitMins += 24 * 60 // Add 24 hours if departure is next day
    }

    waitTime := time.Duration(waitMins) * time.Minute
    log.Printf("Arrival: %s (%s), Departure: %s (%s), Waiting time: %v", 
        arrival, arrival24, departure, departure24, waitTime)
    
    return waitTime
}

func formatDuration(d time.Duration) string {
    hours := int(d.Hours())
    minutes := int(d.Minutes()) % 60
    return fmt.Sprintf("%dh %dm", hours, minutes)
}

func formatStops(stops []Stop) []map[string]interface{} {
    result := make([]map[string]interface{}, len(stops))
    for i, stop := range stops {
        result[i] = map[string]interface{}{
            "station":   stop.Station,
            "arrival":   stop.Arrival,
            "departure": stop.Departure,
            "platform": stop.Platform,
            "distance": fmt.Sprintf("%.1f KM", stop.Distance),
        }
    }
    return result
}

func processBatch(trains []Train) {
    for _, train := range trains {
        processTrainSchedule(&train)
    }
}

func formatStationCode(station string) string {
	station = strings.TrimSpace(station)
	station = strings.ToUpper(station)
	return station
}

func convertToStops(schedule []TrainSchedule) []Stop {
    stops := make([]Stop, len(schedule))
    for i, s := range schedule {
        stops[i] = Stop{
            Station:   formatStationCode(s.Station),
            Arrival:   s.Arrival,
            Departure: s.Departure,
            Platform:  s.Platform,
            Distance:  utils.ParseDistance(s.Distance),
        }
    }
    return stops
}

// GetTrainSuggestions returns train suggestions based on search term
func GetTrainSuggestionsr(w http.ResponseWriter, r *http.Request) {
    searchTerm := r.URL.Query().Get("q")
    if searchTerm == "" {
        http.Error(w, "Search term is required", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := config.MongoDB.Collection("trains")

    filter := bson.M{
        "$or": []bson.M{
            {
                "train_number": bson.M{
                    "$regex": primitive.Regex{
                        Pattern: searchTerm,
                        Options: "i",
                    },
                },
            },
            {
                "title": bson.M{
                    "$regex": primitive.Regex{
                        Pattern: searchTerm,
                        Options: "i",
                    },
                },
            },
        },
    }

    findOptions := options.Find().
        SetLimit(10).
        SetProjection(bson.M{
            "train_number": 1,
            "title": 1,
        })

    cursor, err := collection.Find(ctx, filter, findOptions)
    if err != nil {
        log.Printf("Error querying trains: %v", err)
        http.Error(w, "Error fetching suggestions", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    var suggestions []TrainSuggestion
    for cursor.Next(ctx) {
        var train Train
        if err := cursor.Decode(&train); err != nil {
            continue
        }
        suggestions = append(suggestions, TrainSuggestion{
            TrainNumber: train.TrainNumber,
            TrainName:   train.Title,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "suggestions": suggestions,
    })
}

func GetTrainDetails(w http.ResponseWriter, r *http.Request) {
    // Get train number from URL path parameters using gorilla/mux
    vars := mux.Vars(r)
    trainNumber := vars["train_number"]
    
    if trainNumber == "" {
        sendErrorResponse(w, "Train number is required", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(r.Context(), searchTimeout)
    defer cancel()

    collection := config.MongoDB.Collection("trains")
    
    // Convert trainNumber string to int
    trainNum, err := strconv.Atoi(trainNumber)
    if err != nil {
        sendErrorResponse(w, "Invalid train number format", http.StatusBadRequest)
        return
    }

    var train Train
    err = collection.FindOne(ctx, bson.M{"train_number": trainNum}).Decode(&train)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            sendErrorResponse(w, "Train not found", http.StatusNotFound)
            return
        }
        sendErrorResponse(w, "Error finding train", http.StatusInternalServerError)
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
        sendErrorResponse(w, "Invalid request format", http.StatusBadRequest)
        return
    }

    if req.Station == "" {
        sendErrorResponse(w, "Station code is required", http.StatusBadRequest)
        return
    }

    stationCode := formatStationCode(req.Station)
    trainCache.RLock()
    routes := trainCache.routes[stationCode]
    trainCache.RUnlock()

    trains := make([]*TrainRoute, 0)
    for _, routeList := range routes {
        trains = append(trains, routeList...)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "trains": trains,
        "count":  len(trains),
    })
}

func GetTrainsBetweenStations(w http.ResponseWriter, r *http.Request) {
    startTime := time.Now() 
    var req struct {
        FromStation string `json:"from_station"`
        ToStation   string `json:"to_station"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendErrorResponse(w, "Invalid request format", http.StatusBadRequest)
        return
    }

    // Clean and validate station codes
    fromCode := resolveStationCode(req.FromStation)
    toCode := resolveStationCode(req.ToStation)

    if fromCode == "" || toCode == "" {
        sendErrorResponse(w, "Invalid station code(s)", http.StatusBadRequest)
        return
    }

    // Check cache
    cacheKey := fmt.Sprintf("%s_%s", fromCode, toCode)
    if cachedResponse, found := checkCache(cacheKey); found {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(cachedResponse)
        return
    }

    // Find routes
    routes := findRoutes(fromCode, toCode)

    // Format response
    response := formatRouteResponse(routes, fromCode, toCode, startTime)

    // Cache response
    setCache(cacheKey, response)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func resolveStationCode(input string) string {
    if input == "" {
        return ""
    }

    input = strings.TrimSpace(strings.ToUpper(input))

    // Check if it's already a code
    if len(input) <= 5 && !strings.Contains(input, " ") {
        return input
    }

    // Check if it's in "CODE - Name" format
    if parts := strings.Split(input, " - "); len(parts) > 1 {
        return strings.TrimSpace(parts[0])
    }

    // Look up in station cache
    stationCache.RLock()
    defer stationCache.RUnlock()

    // Try exact match
    if code, exists := stationCache.names[input]; exists {
        return code
    }

    // Try partial match
    for name, code := range stationCache.names {
        if strings.Contains(strings.ToUpper(name), input) {
            return code
        }
    }

    return input
}

func checkCache(key string) (interface{}, bool) {
    trainCache.RLock()
    defer trainCache.RUnlock()

    if trainCache.expiry.Before(time.Now()) {
        return nil, false
    }

    if routes, exists := trainCache.routes[key]; exists {
        return routes, true
    }
    return nil, false
}

// Add this to your train_handler.go file if it's not already there
func GetStationSuggestionsr(w http.ResponseWriter, r *http.Request) {
    searchTerm := r.URL.Query().Get("q")
    if searchTerm == "" {
        http.Error(w, "Search term is required", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := config.MongoDB.Collection("trains")

    pipeline := []bson.M{
        {
            "$unwind": "$schedule_table",
        },
        {
            "$match": bson.M{
                "schedule_table.station": bson.M{
                    "$regex": primitive.Regex{
                        Pattern: searchTerm,
                        Options: "i",
                    },
                },
            },
        },
        {
            "$group": bson.M{
                "_id": "$schedule_table.station",
            },
        },
        {
            "$limit": 10,
        },
    }

    cursor, err := collection.Aggregate(ctx, pipeline)
    if err != nil {
        log.Printf("Error querying stations: %v", err)
        http.Error(w, "Error fetching suggestions", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(ctx)

    var suggestions []StationSuggestion
    for cursor.Next(ctx) {
        var result struct {
            ID string `bson:"_id"`
        }
        if err := cursor.Decode(&result); err != nil {
            continue
        }
        suggestions = append(suggestions, StationSuggestion{
            StationName: result.ID,
            StationCode: extractStationCode(result.ID),
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "suggestions": suggestions,
    })
}