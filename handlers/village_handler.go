package handlers

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "sync"
    "village_site/config"
    "village_site/models"
)

type VillageRequest struct {
    State       string `json:"state"`
    District    string `json:"district"`
    Subdistrict string `json:"subdistrict"`
    Locality    string `json:"locality"`
}

type NearbyFacility struct {
    Title     string  `json:"title"`
    Address   string  `json:"address"`
    Distance  float64 `json:"distance"`
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
}

type CollegeNear struct {
    Name    string `json:"name"`
    Address string `json:"address"`
}

type SchoolNear struct {
    Name    string `json:"name"`
    Address string `json:"address"`
}

type Highway struct {
    Highway string `json:"highway"`
}

type VillageDetails struct {
    BasicInfo struct {
        LocalityName    string         `json:"locality_name"`
        State          string         `json:"state"`
        District       string         `json:"district"`
        Subdistrict    string         `json:"subdistrict"`
        Latitude       float64        `json:"latitude"`
        Longitude      float64        `json:"longitude"`
        CollegesNear   []CollegeNear  `json:"colleges_near"`
        SchoolsNear    []SchoolNear   `json:"schools_near"`
        Highways       []Highway      `json:"national_highways"`
        Rivers         []models.River  `json:"rivers"`
        PinCode        string         `json:"pin_code"`
        ParliamentMP   string         `json:"parliament_mp"`
        AssemblyMLA    string         `json:"assembly_mla"`
        Language       string         `json:"language"`
        Elevation      float64        `json:"elevation"`
        PostOffice     string         `json:"post_office"`
        Block          string         `json:"block"`
        Tehsil         string         `json:"tehsil"`
        Division       string         `json:"division"`
        VillageName    string         `json:"village_name,omitempty"`
        MainVillage    string         `json:"main_village,omitempty"`
    } `json:"basic_info"`

    NearbyFacilities struct {
        ATMs           []NearbyFacility `json:"atms"`
        BusStops       []NearbyFacility `json:"bus_stops"`
        Cinemas        []NearbyFacility `json:"cinemas"`
        Colleges       []NearbyFacility `json:"colleges"`
        Electronics    []NearbyFacility `json:"electronics"`
        Governments    []NearbyFacility `json:"governments"`
        Hospitals      []NearbyFacility `json:"hospitals"`
        Hotels         []NearbyFacility `json:"hotels"`
        Mosques        []NearbyFacility `json:"mosques"`
        Parks          []NearbyFacility `json:"parks"`
        PetrolPumps    []NearbyFacility `json:"petrol_pumps"`
        PoliceStations []NearbyFacility `json:"police_stations"`
        Restaurants    []NearbyFacility `json:"restaurants"`
        Schools        []NearbyFacility `json:"schools"`
        Supermarkets   []NearbyFacility `json:"supermarkets"`
        Temples        []NearbyFacility `json:"temples"`
    } `json:"nearby_facilities"`

    CensusData map[string]interface{} `json:"census_data,omitempty"`
}

func GetVillageDetails(w http.ResponseWriter, r *http.Request) {
    var req VillageRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    log.Printf("Received request for village: State=%s, District=%s, Subdistrict=%s, Locality=%s",
        req.State, req.District, req.Subdistrict, req.Locality)

    var response VillageDetails

    // Initialize empty slices for all facility types
    response.NearbyFacilities = struct {
        ATMs           []NearbyFacility `json:"atms"`
        BusStops       []NearbyFacility `json:"bus_stops"`
        Cinemas        []NearbyFacility `json:"cinemas"`
        Colleges       []NearbyFacility `json:"colleges"`
        Electronics    []NearbyFacility `json:"electronics"`
        Governments    []NearbyFacility `json:"governments"`
        Hospitals      []NearbyFacility `json:"hospitals"`
        Hotels         []NearbyFacility `json:"hotels"`
        Mosques        []NearbyFacility `json:"mosques"`
        Parks          []NearbyFacility `json:"parks"`
        PetrolPumps    []NearbyFacility `json:"petrol_pumps"`
        PoliceStations []NearbyFacility `json:"police_stations"`
        Restaurants    []NearbyFacility `json:"restaurants"`
        Schools        []NearbyFacility `json:"schools"`
        Supermarkets   []NearbyFacility `json:"supermarkets"`
        Temples        []NearbyFacility `json:"temples"`
    }{
        ATMs:           make([]NearbyFacility, 0),
        BusStops:       make([]NearbyFacility, 0),
        Cinemas:        make([]NearbyFacility, 0),
        Colleges:       make([]NearbyFacility, 0),
        Electronics:    make([]NearbyFacility, 0),
        Governments:    make([]NearbyFacility, 0),
        Hospitals:      make([]NearbyFacility, 0),
        Hotels:         make([]NearbyFacility, 0),
        Mosques:        make([]NearbyFacility, 0),
        Parks:          make([]NearbyFacility, 0),
        PetrolPumps:    make([]NearbyFacility, 0),
        PoliceStations: make([]NearbyFacility, 0),
        Restaurants:    make([]NearbyFacility, 0),
        Schools:        make([]NearbyFacility, 0),
        Supermarkets:   make([]NearbyFacility, 0),
        Temples:        make([]NearbyFacility, 0),
    }

    // Get basic village information
    var collegesNearJSON, schoolsNearJSON, highwaysJSON, riversJSON string
    err := config.DB.QueryRow(`
        SELECT 
            COALESCE(locality, village_name) as village_name,
            state,
            district,
            subdistrict,
            COALESCE(NULLIF(trim(latitude::text), '')::float8, 0) as latitude,
            COALESCE(NULLIF(trim(longitude::text), '')::float8, 0) as longitude,
            COALESCE(NULLIF(colleges_near::text, ''), '[]') as colleges_near,
            COALESCE(NULLIF(schools_near::text, ''), '[]') as schools_near,
            COALESCE(NULLIF(national_highways::text, ''), '[]') as national_highways,
            COALESCE(NULLIF(rivers::text, ''), '[]') as rivers,
            COALESCE(pin_code::text, '') as pin_code,
            COALESCE(parliament_mp, '') as parliament_mp,
            COALESCE(assembly_mla, '') as assembly_mla,
            COALESCE(language, '') as language,
            COALESCE(NULLIF(trim(elevation::text), '')::float8, 0) as elevation,
            COALESCE(post_office, '') as post_office,
            COALESCE(block, '') as block,
            COALESCE(tehsil, '') as tehsil,
            COALESCE(division, '') as division,
            COALESCE(village_name, '') as original_village_name,
            COALESCE(main_village, '') as main_village
        FROM villages 
        WHERE state = $1 
        AND district = $2 
        AND subdistrict = $3 
        AND (LOWER(locality) = LOWER($4) OR LOWER(village_name) = LOWER($4))
        LIMIT 1`,
        req.State, req.District, req.Subdistrict, req.Locality).Scan(
            &response.BasicInfo.LocalityName,
            &response.BasicInfo.State,
            &response.BasicInfo.District,
            &response.BasicInfo.Subdistrict,
            &response.BasicInfo.Latitude,
            &response.BasicInfo.Longitude,
            &collegesNearJSON,
            &schoolsNearJSON,
            &highwaysJSON,
            &riversJSON,
            &response.BasicInfo.PinCode,
            &response.BasicInfo.ParliamentMP,
            &response.BasicInfo.AssemblyMLA,
            &response.BasicInfo.Language,
            &response.BasicInfo.Elevation,
            &response.BasicInfo.PostOffice,
            &response.BasicInfo.Block,
            &response.BasicInfo.Tehsil,
            &response.BasicInfo.Division,
            &response.BasicInfo.VillageName,
            &response.BasicInfo.MainVillage,
    )

    if err != nil {
        log.Printf("Error fetching village details: %v", err)
        http.Error(w, "Village/Locality not found", http.StatusNotFound)
        return
    }

    log.Printf("Found village: %s with coordinates: %f, %f", 
        response.BasicInfo.LocalityName, 
        response.BasicInfo.Latitude, 
        response.BasicInfo.Longitude)

    // Parse JSON fields with error handling
    if err := json.Unmarshal([]byte(collegesNearJSON), &response.BasicInfo.CollegesNear); err != nil {
        log.Printf("Error parsing colleges_near: %v", err)
        response.BasicInfo.CollegesNear = []CollegeNear{}
    }
    if err := json.Unmarshal([]byte(schoolsNearJSON), &response.BasicInfo.SchoolsNear); err != nil {
        log.Printf("Error parsing schools_near: %v", err)
        response.BasicInfo.SchoolsNear = []SchoolNear{}
    }
    if err := json.Unmarshal([]byte(highwaysJSON), &response.BasicInfo.Highways); err != nil {
        log.Printf("Error parsing highways: %v", err)
        response.BasicInfo.Highways = []Highway{}
    }
    if err := json.Unmarshal([]byte(riversJSON), &response.BasicInfo.Rivers); err != nil {
        log.Printf("Error parsing rivers: %v", err)
        response.BasicInfo.Rivers = []models.River{}
    }

    // Only proceed with nearby facilities if we have valid coordinates
    if response.BasicInfo.Latitude != 0 && response.BasicInfo.Longitude != 0 {
        var wg sync.WaitGroup
        facilityMap := make(map[string][]NearbyFacility)
        var mutex sync.Mutex

        // List of facility tables and their corresponding fields
        facilityTables := map[string]string{
            "atm": "atms",
            "bus_stop": "bus_stops",
            "cinema": "cinemas",
            "college": "colleges",
            "electronic": "electronics",
            "government": "governments",
            "hospitals": "hospitals",
            "hotel": "hotels",
            "mosque": "mosques",
            "park": "parks",
            "petrol_pump": "petrol_pumps",
            "police_station": "police_stations",
            "restaurant": "restaurants",
            "school": "schools",
            "supermarket": "supermarkets",
            "temples": "temples",
        }

        // Fetch facilities concurrently
        for tableName := range facilityTables {
            wg.Add(1)
            go func(table string) {
                defer wg.Done()

                query := fmt.Sprintf(`
                    SELECT 
                        COALESCE(title, '') as title,
                        COALESCE(address, '') as address,
                        COALESCE(NULLIF(trim(latitude::text), '')::float8, 0) as latitude,
                        COALESCE(NULLIF(trim(longitude::text), '')::float8, 0) as longitude,
                        ROUND(
                            (6371 * acos(
                                cos(radians($1)) * 
                                cos(radians(NULLIF(trim(latitude::text), '')::float8)) * 
                                cos(radians(NULLIF(trim(longitude::text), '')::float8) - radians($2)) + 
                                sin(radians($1)) * 
                                sin(radians(NULLIF(trim(latitude::text), '')::float8))
                            ))::numeric, 2
                        ) as distance
                    FROM %s
                    WHERE 
                        NULLIF(trim(latitude::text), '') IS NOT NULL
                        AND NULLIF(trim(longitude::text), '') IS NOT NULL
                        AND NULLIF(trim(latitude::text), '')::float8 BETWEEN $1 - 0.5 AND $1 + 0.5
                        AND NULLIF(trim(longitude::text), '')::float8 BETWEEN $2 - 0.5 AND $2 + 0.5
                        AND title IS NOT NULL
                    ORDER BY (
                        6371 * acos(
                            cos(radians($1)) * 
                            cos(radians(NULLIF(trim(latitude::text), '')::float8)) * 
                            cos(radians(NULLIF(trim(longitude::text), '')::float8) - radians($2)) + 
                            sin(radians($1)) * 
                            sin(radians(NULLIF(trim(latitude::text), '')::float8))
                        )
                    )
                    LIMIT 10`, table)

                rows, err := config.DB.Query(query, 
                    response.BasicInfo.Latitude, 
                    response.BasicInfo.Longitude)
                
                if err != nil {
                    log.Printf("Error querying %s: %v", table, err)
                    return
                }
                defer rows.Close()

                var facilities []NearbyFacility
                for rows.Next() {
                    var f NearbyFacility
                    if err := rows.Scan(&f.Title, &f.Address, &f.Latitude, &f.Longitude, &f.Distance); err != nil {
                        log.Printf("Error scanning %s row: %v", table, err)
                        continue
                    }
                    if f.Latitude != 0 && f.Longitude != 0 {
                        facilities = append(facilities, f)
                    }
                }

                mutex.Lock()
                facilityMap[table] = facilities
                mutex.Unlock()
            }(tableName)
        }

        wg.Wait()

        // Assign results to response
        for tableName, responseField := range facilityTables {
            facilities := facilityMap[tableName]
            if facilities == nil {
                facilities = make([]NearbyFacility, 0)
            }
            
            switch responseField {
            case "atms":
                response.NearbyFacilities.ATMs = facilities
            case "bus_stops":
                response.NearbyFacilities.BusStops = facilities
            case "cinemas":
                response.NearbyFacilities.Cinemas = facilities
            case "colleges":
                response.NearbyFacilities.Colleges = facilities
            case "electronics":
                response.NearbyFacilities.Electronics = facilities
            case "governments":
                response.NearbyFacilities.Governments = facilities
            case "hospitals":
                response.NearbyFacilities.Hospitals = facilities
            case "hotels":
                response.NearbyFacilities.Hotels = facilities
            case "mosques":
                response.NearbyFacilities.Mosques = facilities
            case "parks":
                response.NearbyFacilities.Parks = facilities
            case "petrol_pumps":
                response.NearbyFacilities.PetrolPumps = facilities
            case "police_stations":
                response.NearbyFacilities.PoliceStations = facilities
            case "restaurants":
                response.NearbyFacilities.Restaurants = facilities
            case "schools":
                response.NearbyFacilities.Schools = facilities
            case "supermarkets":
                response.NearbyFacilities.Supermarkets = facilities
            case "temples":
                response.NearbyFacilities.Temples = facilities
            }
        }
    }

    // Get census data if available
    var censusDataStr string
    err = config.DB.QueryRow(`
        SELECT 
            CASE 
                WHEN EXISTS (
                    SELECT 1 
                    FROM village_census 
                    WHERE LOWER(district) = LOWER($1) 
                    AND LOWER(subdistrict) = LOWER($2) 
                    AND LOWER(village) = LOWER($3)
                ) 
                THEN (
                    SELECT jsonb_build_object(
                        'demographics', jsonb_build_object(
                            'total_population', COALESCE(total_population, 0),
                            'female_population', COALESCE(female_population, 0),
                            'total_literacy', COALESCE(total_literacy, 0),
                            'female_literacy', COALESCE(female_literacy, 0),
                            'st_population', COALESCE(st_population, 0),
                            'working_population', COALESCE(working_population, 0)
                        ),
                        'location', jsonb_build_object(
                            'gram_panchayat', COALESCE(gram_panchayat, ''),
                            'distance_from_subdistrict', COALESCE(distance_from_subdistrict, 0),
                            'distance_from_district', COALESCE(distance_from_district, 0),
                            'nearest_town', COALESCE(nearest_town, ''),
                            'nearest_town_distance', COALESCE(nearest_town_distance, 0)
                        ),
                        'education', jsonb_build_object(
                            'govt_primary_school', COALESCE(govt_primary_school, 0) = 1,
                            'govt_disabled_school', COALESCE(govt_disabled_school, 0) = 1,
                            'govt_engineering_college', COALESCE(govt_engineering_college, 0) = 1,
                            'govt_medical_college', COALESCE(govt_medical_college, 0) = 1,
                            'govt_polytechnic', COALESCE(govt_polytechnic, 0) = 1,
                            'govt_secondary_school', COALESCE(govt_secondary_school, 0) = 1,
                            'govt_senior_secondary', COALESCE(govt_senior_secondary, 0) = 1,
                            'nearest_pre_primary', COALESCE(nearest_pre_primary, ''),
                            'nearest_polytechnic', COALESCE(nearest_polytechnic, ''),
                            'nearest_secondary', COALESCE(nearest_secondary, '')
                        ),
                        'health', jsonb_build_object(
                            'primary_health_center', COALESCE(primary_health_center, 0) = 1,
                            'community_health_center', COALESCE(community_health_center, 0) = 1,
                            'family_welfare_center', COALESCE(family_welfare_center, 0) = 1,
                            'maternity_child_center', COALESCE(maternity_child_center, 0) = 1,
                            'tb_clinic', COALESCE(tb_clinic, 0) = 1,
                            'veterinary_hospital', COALESCE(veterinary_hospital, 0) = 1,
                            'mobile_health_clinic', COALESCE(mobile_health_clinic, 0) = 1,
                            'medical_shop', COALESCE(medical_shop, 0) = 1
                        ),
                        'infrastructure', jsonb_build_object(
                            'treated_tap_water', COALESCE(treated_tap_water, 0) = 1,
                            'untreated_water', COALESCE(untreated_water, 0) = 1,
                            'covered_well', COALESCE(covered_well, 0) = 1,
                            'uncovered_well', COALESCE(uncovered_well, 0) = 1,
                            'handpump', COALESCE(handpump, 0) = 1,
                            'drainage_system', COALESCE(drainage_system, 0) = 1,
                            'garbage_collection', COALESCE(garbage_collection, 0) = 1,
                            'direct_drain_discharge', COALESCE(direct_drain_discharge, 0) = 1
                        ),
                        'connectivity', jsonb_build_object(
                            'mobile_coverage', COALESCE(mobile_coverage, 0) = 1,
                            'internet_cafe', COALESCE(internet_cafe, 0) = 1,
                            'private_courier', COALESCE(private_courier, 0) = 1,
                            'bus_service', COALESCE(bus_service, 0) = 1,
                            'railway_station', COALESCE(railway_station, 0) = 1,
                            'animal_cart', COALESCE(animal_cart, 0) = 1
                        ),
                        'transport', jsonb_build_object(
                            'national_highway', COALESCE(national_highway, 0) = 1,
                            'state_highway', COALESCE(state_highway, 0) = 1,
                            'district_road', COALESCE(district_road, 0) = 1
                        ),
                        'financial', jsonb_build_object(
                            'atm', COALESCE(atm, 0) = 1,
                            'commercial_bank', COALESCE(commercial_bank, 0) = 1,
                            'cooperative_bank', COALESCE(cooperative_bank, 0) = 1
                        ),
                        'other_amenities', jsonb_build_object(
                            'power_supply', COALESCE(power_supply, 0) = 1,
                            'anganwadi', COALESCE(anganwadi, 0) = 1,
                            'birth_death_registration', COALESCE(birth_death_registration, 0) = 1,
                            'newspaper', COALESCE(newspaper, 0) = 1
                        ),
                        'area', jsonb_build_object(
                            'total_area', COALESCE(total_area, 0),
                            'irrigated_area', COALESCE(irrigated_area, 0)
                        )
                    )::text
                    FROM village_census
                    WHERE LOWER(district) = LOWER($1) 
                    AND LOWER(subdistrict) = LOWER($2) 
                    AND LOWER(village) = LOWER($3)
                )
                ELSE NULL
            END`,
        req.District, req.Subdistrict, req.Locality).Scan(&censusDataStr)

    if err == nil && censusDataStr != "" {
        if err := json.Unmarshal([]byte(censusDataStr), &response.CensusData); err != nil {
            log.Printf("Error parsing census data JSON: %v", err)
            response.CensusData = nil
        }
    } else {
        log.Printf("No census data found or error: %v", err)
        response.CensusData = nil
    }

    // Set response headers
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

    // Return response
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("Error encoding response: %v", err)
        http.Error(w, "Error encoding response", http.StatusInternalServerError)
        return
    }
}