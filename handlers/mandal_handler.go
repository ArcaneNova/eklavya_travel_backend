package handlers

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "sync" // Add this import
    "village_site/config"
    "village_site/utils"
    "database/sql"
    // Remove models import since it's not being used
)

type MandalRequest struct {
    District    string `json:"district"`
    Subdistrict string `json:"subdistrict"`
}

type MandalDistanceRequest struct {
    FromDistrict    string `json:"from_district"`
    FromSubdistrict string `json:"from_subdistrict"`
    ToDistrict      string `json:"to_district"`
    ToSubdistrict   string `json:"to_subdistrict"`
}

type FormattedTime struct {
    Hours   int    `json:"hours"`
    Minutes int    `json:"minutes"`
    Text    string `json:"text"`
}

type Facility struct {
    Title       string  `json:"title"`
    Address     string  `json:"address"`
    State       string  `json:"state"`
    District    string  `json:"district"`
    Subdistrict string  `json:"subdistrict"`
    Village     string  `json:"village"`
    Latitude    float64 `json:"latitude"`
    Longitude   float64 `json:"longitude"`
    Distance    float64 `json:"distance,omitempty"`
}

type Village struct {
    Name            string   `json:"name"`
    Population      int      `json:"population"`
    Area            float64  `json:"area"`
    Panchayat       string   `json:"panchayat"`
    PinCode         string   `json:"pincode"`
    Distance        float64  `json:"distance_from_mandal"`
    Facilities      []string `json:"facilities"`
    Schools         int      `json:"schools_count"`
    Hospitals       int      `json:"hospitals_count"`
    Banks           int      `json:"banks_count"`
    PostOffices     int      `json:"post_offices_count"`
    WaterSource     string   `json:"water_source"`
    PowerSupply     string   `json:"power_supply"`
    RoadConnectivity string  `json:"road_connectivity"`
}

type NearbyFacilities struct {
    ATMs            []Facility `json:"atms"`
    BusStops        []Facility `json:"bus_stops"`
    Cinemas         []Facility `json:"cinemas"`
    Colleges        []Facility `json:"colleges"`
    Electronics     []Facility `json:"electronics"`
    Government      []Facility `json:"government"`
    Hospitals       []Facility `json:"hospitals"`
    Hotels          []Facility `json:"hotels"`
    Mosques         []Facility `json:"mosques"`
    Parks           []Facility `json:"parks"`
    PetrolPumps     []Facility `json:"petrol_pumps"`
    PoliceStations  []Facility `json:"police_stations"`
    Restaurants     []Facility `json:"restaurants"`
    Schools         []Facility `json:"schools"`
    Supermarkets    []Facility `json:"supermarkets"`
    Temples         []Facility `json:"temples"`
}

type MandalDetails struct {
    BasicInfo struct {
        Latitude           float64 `json:"latitude"`
        Longitude         float64 `json:"longitude"`
        Language          string  `json:"language"`
        ElevationAltitude float64 `json:"elevation_altitude"`
        TelephoneCode    string  `json:"telephone_std_code"`
        VehicleReg       string  `json:"vehicle_registration"`
        RTOOffice        string  `json:"rto_office"`
        AssemblyConst    string  `json:"assembly_constituency"`
        AssemblyMLA      string  `json:"assembly_mla"`
        LokSabha         string  `json:"lok_sabha_constituency"`
        ParliamentMP     string  `json:"parliament_mp"`
        AlternateMandal  string  `json:"alternate_mandal_name"`
        AlternateCity    string  `json:"alternate_city_name"`
        AlternateTehsil  string  `json:"alternate_tehsil_name"`
        AlternateBlock   string  `json:"alternate_block_name"`
        AlternateTaluk   string  `json:"alternate_taluk_name"`
        AlternateTaluka  string  `json:"alternate_taluka_name"`
        AdminType        string  `json:"administrative_type"`
        Headquarters     string  `json:"headquarters"`
        Region           string  `json:"region"`
        NearbyCities     string  `json:"nearby_cities"`
        VillagesCount    int     `json:"villages_count"`
        PanchayatsCount  int     `json:"panchayats_count"`
        Elevation        float64 `json:"elevation"`
        Languages        string  `json:"languages"`
        PoliticalParties string  `json:"political_parties"`
        CurrentMLA       string  `json:"current_mla"`
        MLAParty         string  `json:"mla_party"`
        ParliamentConst  string  `json:"parliament_constituency"`
        SmallestVillage  string  `json:"smallest_village"`
        BiggestVillage   string  `json:"biggest_village"`
        PopulationTotal  int     `json:"population_total"`
        PopulationMales  int     `json:"population_males"`
        PopulationFemales int    `json:"population_females"`
        HousesCount      int     `json:"houses_count"`
        District         string  `json:"district"`
        Subdistrict      string  `json:"subdistrict"`
    } `json:"basic_info"`

    Villages         []Village         `json:"villages"`
    Facilities       NearbyFacilities  `json:"facilities"`
}

type TravelInfo struct {
    Distance     float64                  `json:"distance"`
    TravelTimes  map[string]FormattedTime `json:"travel_times"`
    RouteDetails struct {
        MainRoads    []string `json:"main_roads"`
        Landmarks    []string `json:"landmarks"`
        Interchanges []string `json:"interchanges"`
    } `json:"route_details"`
}

type MandalDistanceResponse struct {
    TravelInfo        TravelInfo    `json:"travel_info"`
    FromMandalDetails MandalDetails `json:"from_mandal_details"`
    ToMandalDetails   MandalDetails `json:"to_mandal_details"`
}

type DistrictSuggestion struct {
    District string `json:"district"`
}

type SubdistrictSuggestion struct {
    Subdistrict string `json:"subdistrict"`
}

func formatTime(decimalHours float64) FormattedTime {
    totalMinutes := int(decimalHours * 60)
    hours := totalMinutes / 60
    minutes := totalMinutes % 60
    
    text := ""
    if hours > 0 {
        text = fmt.Sprintf("%d hour", hours)
        if hours > 1 {
            text += "s"
        }
        if minutes > 0 {
            text += fmt.Sprintf(" %d minute", minutes)
            if minutes > 1 {
                text += "s"
            }
        }
    } else {
        text = fmt.Sprintf("%d minute", minutes)
        if minutes > 1 {
            text += "s"
        }
    }

    return FormattedTime{
        Hours:   hours,
        Minutes: minutes,
        Text:    text,
    }
}

func initializeMandalDetails() MandalDetails {
    return MandalDetails{
        Villages: []Village{},
        Facilities: NearbyFacilities{
            ATMs:           []Facility{},
            BusStops:       []Facility{},
            Cinemas:        []Facility{},
            Colleges:       []Facility{},
            Electronics:    []Facility{},
            Government:     []Facility{},
            Hospitals:      []Facility{},
            Hotels:         []Facility{},
            Mosques:        []Facility{},
            Parks:          []Facility{},
            PetrolPumps:    []Facility{},
            PoliceStations: []Facility{},
            Restaurants:    []Facility{},
            Schools:        []Facility{},
            Supermarkets:   []Facility{},
            Temples:        []Facility{},
        },
    }
}

func calculateTravelTimes(distance float64) map[string]FormattedTime {
    return map[string]FormattedTime{
        "bus":   formatTime(distance / 40),  // 40 km/h average speed
        "train": formatTime(distance / 60),  // 60 km/h average speed
        "car":   formatTime(distance / 50),  // 50 km/h average speed
        "bike":  formatTime(distance / 45),  // 45 km/h average speed
        "auto":  formatTime(distance / 35),  // 35 km/h average speed
    }
}

func GetMandalDetails(w http.ResponseWriter, r *http.Request) {
    var req MandalRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("Error decoding request: %v", err)
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }

    if req.District == "" || req.Subdistrict == "" {
        http.Error(w, "District and Subdistrict are required", http.StatusBadRequest)
        return
    }

    log.Printf("Fetching details for district: %s, subdistrict: %s", req.District, req.Subdistrict)

    mandalDetails := initializeMandalDetails()

    if err := getMandaBasicInfo(&mandalDetails, req.District, req.Subdistrict); err != nil {
        log.Printf("Error fetching basic mandal info: %v", err)
        http.Error(w, "Error fetching mandal details", http.StatusInternalServerError)
        return
    }

    if err := getFacilities(&mandalDetails, req.District, req.Subdistrict); err != nil {
        log.Printf("Error fetching facilities: %v", err)
        // Continue with partial data
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(mandalDetails)
}

func GetMandalDistance(w http.ResponseWriter, r *http.Request) {
    var req MandalDistanceRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("Error decoding request: %v", err)
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }

    if req.FromDistrict == "" || req.FromSubdistrict == "" || 
       req.ToDistrict == "" || req.ToSubdistrict == "" {
        http.Error(w, "All district and subdistrict fields are required", http.StatusBadRequest)
        return
    }

    var response MandalDistanceResponse

    // Initialize mandal details
    fromMandal := initializeMandalDetails()
    toMandal := initializeMandalDetails()

    // Get source mandal details
    if err := getMandaBasicInfo(&fromMandal, req.FromDistrict, req.FromSubdistrict); err != nil {
        log.Printf("Error fetching source mandal: %v", err)
        http.Error(w, "Source mandal not found", http.StatusNotFound)
        return
    }

    // Get destination mandal details
    if err := getMandaBasicInfo(&toMandal, req.ToDistrict, req.ToSubdistrict); err != nil {
        log.Printf("Error fetching destination mandal: %v", err)
        http.Error(w, "Destination mandal not found", http.StatusNotFound)
        return
    }

    // Calculate distance using mandal coordinates
    distance := utils.CalculateDistance(
        fromMandal.BasicInfo.Latitude,
        fromMandal.BasicInfo.Longitude,
        toMandal.BasicInfo.Latitude,
        toMandal.BasicInfo.Longitude,
    )

    response.TravelInfo.Distance = distance
    response.TravelInfo.TravelTimes = calculateTravelTimes(distance)

    // Get facilities for both mandals
    log.Printf("Fetching facilities for source mandal: %s, %s", req.FromDistrict, req.FromSubdistrict)
    if err := getFacilities(&fromMandal, req.FromDistrict, req.FromSubdistrict); err != nil {
        log.Printf("Error fetching source mandal facilities: %v", err)
    }

    log.Printf("Fetching facilities for destination mandal: %s, %s", req.ToDistrict, req.ToSubdistrict)
    if err := getFacilities(&toMandal, req.ToDistrict, req.ToSubdistrict); err != nil {
        log.Printf("Error fetching destination mandal facilities: %v", err)
    }

    response.FromMandalDetails = fromMandal
    response.ToMandalDetails = toMandal

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getMandaBasicInfo(mandalDetails *MandalDetails, district, subdistrict string) error {
    query := `
        SELECT 
            COALESCE(latitude::float8, 0.0), 
            COALESCE(longitude::float8, 0.0), 
            COALESCE(language, ''),
            COALESCE(elevation_altitude::float8, 0.0),
            COALESCE(telephone_std_code, ''),
            COALESCE(vehicle_registration, ''),
            COALESCE(rto_office, ''),
            COALESCE(assembly_constituency, ''),
            COALESCE(assembly_mla, ''),
            COALESCE(lok_sabha_constituency, ''),
            COALESCE(parliament_mp, ''),
            COALESCE(alternate_mandal_name, ''),
            COALESCE(alternate_city_name, ''),
            COALESCE(alternate_tehsil_name, ''),
            COALESCE(alternate_block_name, ''),
            COALESCE(alternate_taluk_name, ''),
            COALESCE(alternate_taluka_name, ''),
            COALESCE(administrative_type, ''),
            COALESCE(headquarters, ''),
            COALESCE(region, ''),
            COALESCE(nearby_cities, ''),
            COALESCE(villages_count::integer, 0),
            COALESCE(panchayats_count::integer, 0),
            COALESCE(elevation::float8, 0.0),
            COALESCE(languages, ''),
            COALESCE(political_parties, ''),
            COALESCE(current_mla, ''),
            COALESCE(mla_party, ''),
            COALESCE(parliament_constituency, ''),
            COALESCE(smallest_village, ''),
            COALESCE(biggest_village, ''),
            COALESCE(population_total::integer, 0),
            COALESCE(population_males::integer, 0),
            COALESCE(population_females::integer, 0),
            COALESCE(houses_count::integer, 0)
        FROM mandals
        WHERE LOWER(district) = LOWER($1) AND LOWER(subdistrict) = LOWER($2)`

    err := config.DB.QueryRow(query, district, subdistrict).Scan(
        &mandalDetails.BasicInfo.Latitude,
        &mandalDetails.BasicInfo.Longitude,
        &mandalDetails.BasicInfo.Language,
        &mandalDetails.BasicInfo.ElevationAltitude,
        &mandalDetails.BasicInfo.TelephoneCode,
        &mandalDetails.BasicInfo.VehicleReg,
        &mandalDetails.BasicInfo.RTOOffice,
        &mandalDetails.BasicInfo.AssemblyConst,
        &mandalDetails.BasicInfo.AssemblyMLA,
        &mandalDetails.BasicInfo.LokSabha,
        &mandalDetails.BasicInfo.ParliamentMP,
        &mandalDetails.BasicInfo.AlternateMandal,
        &mandalDetails.BasicInfo.AlternateCity,
        &mandalDetails.BasicInfo.AlternateTehsil,
        &mandalDetails.BasicInfo.AlternateBlock,
        &mandalDetails.BasicInfo.AlternateTaluk,
        &mandalDetails.BasicInfo.AlternateTaluka,
        &mandalDetails.BasicInfo.AdminType,
        &mandalDetails.BasicInfo.Headquarters,
        &mandalDetails.BasicInfo.Region,
        &mandalDetails.BasicInfo.NearbyCities,
        &mandalDetails.BasicInfo.VillagesCount,
        &mandalDetails.BasicInfo.PanchayatsCount,
        &mandalDetails.BasicInfo.Elevation,
        &mandalDetails.BasicInfo.Languages,
        &mandalDetails.BasicInfo.PoliticalParties,
        &mandalDetails.BasicInfo.CurrentMLA,
        &mandalDetails.BasicInfo.MLAParty,
        &mandalDetails.BasicInfo.ParliamentConst,
        &mandalDetails.BasicInfo.SmallestVillage,
        &mandalDetails.BasicInfo.BiggestVillage,
        &mandalDetails.BasicInfo.PopulationTotal,
        &mandalDetails.BasicInfo.PopulationMales,
        &mandalDetails.BasicInfo.PopulationFemales,
        &mandalDetails.BasicInfo.HousesCount,
    )

    if err != nil {
        return fmt.Errorf("database query error: %v", err)
    }

    mandalDetails.BasicInfo.District = district
    mandalDetails.BasicInfo.Subdistrict = subdistrict

    return nil
}

func getFacilities(mandalDetails *MandalDetails, district, subdistrict string) error {
    queryFacilities := func(tableName string) ([]Facility, error) {
        query := fmt.Sprintf(`
            WITH mandal_center AS (
                SELECT latitude, longitude 
                FROM mandals 
                WHERE LOWER(district) = LOWER($1) 
                AND LOWER(subdistrict) = LOWER($2)
                LIMIT 1
            )
            SELECT 
                f.title, 
                f.address, 
                f.state, 
                f.district, 
                f.subdistrict, 
                f.village, 
                f.latitude, 
                f.longitude,
                ROUND(
                    CAST(
                        111.111 *
                        DEGREES(ACOS(LEAST(1.0, COS(RADIANS(f.latitude))
                        * COS(RADIANS(m.latitude))
                        * COS(RADIANS(f.longitude - m.longitude))
                        + SIN(RADIANS(f.latitude))
                        * SIN(RADIANS(m.latitude)))))
                        AS NUMERIC
                    ), 2
                ) AS distance
            FROM %s f
            CROSS JOIN mandal_center m
            WHERE LOWER(f.district) = LOWER($1) 
            AND LOWER(f.subdistrict) = LOWER($2)
            ORDER BY distance
            LIMIT 10`, tableName)

        rows, err := config.DB.Query(query, district, subdistrict)
        if err != nil {
            return nil, fmt.Errorf("query error for %s: %v", tableName, err)
        }
        defer rows.Close()

        var facilities []Facility
        for rows.Next() {
            var f Facility
            if err := rows.Scan(
                &f.Title,
                &f.Address,
                &f.State,
                &f.District,
                &f.Subdistrict,
                &f.Village,
                &f.Latitude,
                &f.Longitude,
                &f.Distance,
            ); err != nil {
                return nil, fmt.Errorf("scan error for %s: %v", tableName, err)
            }
            facilities = append(facilities, f)
        }

        if err := rows.Err(); err != nil {
            return nil, fmt.Errorf("rows error for %s: %v", tableName, err)
        }

        return facilities, nil
    }

    // Map of facility types to their corresponding slice pointers
    facilityTypes := map[string]*[]Facility{
        "atm":            &mandalDetails.Facilities.ATMs,
        "bus_stop":       &mandalDetails.Facilities.BusStops,
        "cinema":         &mandalDetails.Facilities.Cinemas,
        "college":        &mandalDetails.Facilities.Colleges,
        "electronic":     &mandalDetails.Facilities.Electronics,
        "government":     &mandalDetails.Facilities.Government,
        "hospitals":      &mandalDetails.Facilities.Hospitals,
        "hotel":         &mandalDetails.Facilities.Hotels,
        "mosque":        &mandalDetails.Facilities.Mosques,
        "park":          &mandalDetails.Facilities.Parks,
        "petrol_pump":   &mandalDetails.Facilities.PetrolPumps,
        "police_station": &mandalDetails.Facilities.PoliceStations,
        "restaurant":    &mandalDetails.Facilities.Restaurants,
        "school":        &mandalDetails.Facilities.Schools,
        "supermarket":   &mandalDetails.Facilities.Supermarkets,
        "temples":       &mandalDetails.Facilities.Temples,
    }

    // Query each facility type concurrently using goroutines
    var wg sync.WaitGroup
    errChan := make(chan error, len(facilityTypes))

    for tableName, facilitySlice := range facilityTypes {
        wg.Add(1)
        go func(table string, slice *[]Facility) {
            defer wg.Done()
            facilities, err := queryFacilities(table)
            if err != nil {
                log.Printf("Error fetching %s: %v", table, err)
                *slice = []Facility{}
                errChan <- err
                return
            }
            *slice = facilities
        }(tableName, facilitySlice)
    }

    // Wait for all goroutines to complete
    wg.Wait()
    close(errChan)

    // Check for any errors
    var errs []error
    for err := range errChan {
        errs = append(errs, err)
    }

    if len(errs) > 0 {
        return fmt.Errorf("errors occurred while fetching facilities: %v", errs)
    }

    return nil
}

// Add these new handler functions
func GetDistrictSuggestions(w http.ResponseWriter, r *http.Request) {
    searchTerm := r.URL.Query().Get("q")
    if searchTerm == "" {
        http.Error(w, "Search term is required", http.StatusBadRequest)
        return
    }

    query := `
        SELECT DISTINCT district 
        FROM mandals 
        WHERE LOWER(district) LIKE LOWER($1 || '%')
        ORDER BY district
        LIMIT 10`

    rows, err := config.DB.Query(query, searchTerm)
    if err != nil {
        log.Printf("Error querying districts: %v", err)
        http.Error(w, "Error fetching suggestions", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var suggestions []DistrictSuggestion
    for rows.Next() {
        var district string
        if err := rows.Scan(&district); err != nil {
            log.Printf("Error scanning district: %v", err)
            continue
        }
        suggestions = append(suggestions, DistrictSuggestion{District: district})
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "suggestions": suggestions,
    })
}

func GetSubdistrictSuggestions(w http.ResponseWriter, r *http.Request) {
    searchTerm := r.URL.Query().Get("q")
    district := r.URL.Query().Get("district")
    
    if district == "" {
        http.Error(w, "District is required", http.StatusBadRequest)
        return
    }

    var query string
    var rows *sql.Rows
    var err error

    if searchTerm != "" {
        query = `
            SELECT DISTINCT subdistrict 
            FROM mandals 
            WHERE LOWER(district) = LOWER($1)
            AND LOWER(subdistrict) LIKE LOWER($2 || '%')
            ORDER BY subdistrict
            LIMIT 10`
        rows, err = config.DB.Query(query, district, searchTerm)
    } else {
        query = `
            SELECT DISTINCT subdistrict 
            FROM mandals 
            WHERE LOWER(district) = LOWER($1)
            ORDER BY subdistrict
            LIMIT 10`
        rows, err = config.DB.Query(query, district)
    }

    if err != nil {
        log.Printf("Error querying subdistricts: %v", err)
        http.Error(w, "Error fetching suggestions", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var suggestions []SubdistrictSuggestion
    for rows.Next() {
        var subdistrict string
        if err := rows.Scan(&subdistrict); err != nil {
            log.Printf("Error scanning subdistrict: %v", err)
            continue
        }
        suggestions = append(suggestions, SubdistrictSuggestion{Subdistrict: subdistrict})
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "suggestions": suggestions,
    })
}