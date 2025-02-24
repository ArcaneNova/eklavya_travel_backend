package handlers

import (
    "encoding/json"
    "net/http"
    "village_site/config"
    "log"
)

type LocationRequest struct {
    State      string `json:"state,omitempty"`
    District   string `json:"district,omitempty"`
    Subdistrict string `json:"subdistrict,omitempty"`
}

type LocationResponse struct {
    States       []string `json:"states,omitempty"`
    Districts    []string `json:"districts,omitempty"`
    Subdistricts []string `json:"subdistricts,omitempty"`
    Villages     []string `json:"villages,omitempty"`
}

func GetLocations(w http.ResponseWriter, r *http.Request) {
    log.Printf("GetLocations: Starting request handling")

    // Check if DB is nil
    if config.DB == nil {
        log.Printf("GetLocations: Database connection is nil")
        http.Error(w, "Database connection not initialized", http.StatusInternalServerError)
        return
    }

    // Check DB connection
    if err := config.DB.Ping(); err != nil {
        log.Printf("GetLocations: Database ping failed: %v", err)
        http.Error(w, "Database connection error", http.StatusInternalServerError)
        return
    }

    // Check if villages table exists
    var tableExists bool
    err := config.DB.QueryRow(`
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_name = 'villages'
        )`).Scan(&tableExists)
    
    if err != nil {
        log.Printf("GetLocations: Error checking table existence: %v", err)
        http.Error(w, "Error checking database structure", http.StatusInternalServerError)
        return
    }

    if !tableExists {
        log.Printf("GetLocations: villages table does not exist")
        http.Error(w, "Required table not found", http.StatusInternalServerError)
        return
    }

    var req LocationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        log.Printf("GetLocations: Error decoding request body: %v", err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    log.Printf("GetLocations: Received request with State=%s, District=%s, Subdistrict=%s", 
        req.State, req.District, req.Subdistrict)

    var response LocationResponse

    // If no state provided, return list of states
    if req.State == "" {
        log.Printf("GetLocations: Fetching list of states")
        rows, err := config.DB.Query(`
            SELECT DISTINCT state 
            FROM villages 
            WHERE state IS NOT NULL
            ORDER BY state`)
        if err != nil {
            log.Printf("GetLocations: Error querying states: %v", err)
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var states []string
        for rows.Next() {
            var state string
            if err := rows.Scan(&state); err != nil {
                log.Printf("GetLocations: Error scanning state: %v", err)
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            if state != "" {
                states = append(states, state)
            }
        }
        response.States = states
        log.Printf("GetLocations: Found %d states", len(states))

    } else if req.District == "" {
        // Get districts for state
        log.Printf("GetLocations: Fetching districts for state: %s", req.State)
        rows, err := config.DB.Query(`
            SELECT DISTINCT district 
            FROM villages 
            WHERE state = $1 
            AND district IS NOT NULL
            ORDER BY district`, req.State)
        if err != nil {
            log.Printf("GetLocations: Error querying districts: %v", err)
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var districts []string
        for rows.Next() {
            var district string
            if err := rows.Scan(&district); err != nil {
                log.Printf("GetLocations: Error scanning district: %v", err)
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            if district != "" {
                districts = append(districts, district)
            }
        }
        response.Districts = districts
        log.Printf("GetLocations: Found %d districts for state %s", len(districts), req.State)

    } else if req.Subdistrict == "" {
        // Get subdistricts for district
        log.Printf("GetLocations: Fetching subdistricts for state: %s, district: %s", req.State, req.District)
        rows, err := config.DB.Query(`
            SELECT DISTINCT subdistrict 
            FROM villages 
            WHERE state = $1 
            AND district = $2 
            AND subdistrict IS NOT NULL
            ORDER BY subdistrict`, req.State, req.District)
        if err != nil {
            log.Printf("GetLocations: Error querying subdistricts: %v", err)
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var subdistricts []string
        for rows.Next() {
            var subdistrict string
            if err := rows.Scan(&subdistrict); err != nil {
                log.Printf("GetLocations: Error scanning subdistrict: %v", err)
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            if subdistrict != "" {
                subdistricts = append(subdistricts, subdistrict)
            }
        }
        response.Subdistricts = subdistricts
        log.Printf("GetLocations: Found %d subdistricts for state %s, district %s", 
            len(subdistricts), req.State, req.District)

    } else {
        // Get villages/localities for subdistrict
        log.Printf("GetLocations: Fetching villages for state: %s, district: %s, subdistrict: %s", 
            req.State, req.District, req.Subdistrict)
        rows, err := config.DB.Query(`
            SELECT DISTINCT locality 
            FROM villages 
            WHERE state = $1 
            AND district = $2 
            AND subdistrict = $3 
            AND locality IS NOT NULL
            ORDER BY locality`, req.State, req.District, req.Subdistrict)
        if err != nil {
            log.Printf("GetLocations: Error querying villages: %v", err)
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var villages []string
        for rows.Next() {
            var village string
            if err := rows.Scan(&village); err != nil {
                log.Printf("GetLocations: Error scanning village: %v", err)
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            if village != "" {
                villages = append(villages, village)
            }
        }
        response.Villages = villages
        log.Printf("GetLocations: Found %d villages for state %s, district %s, subdistrict %s", 
            len(villages), req.State, req.District, req.Subdistrict)
    }

    // Add cache control headers
    w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
    w.Header().Set("Content-Type", "application/json")
    
    // Return response
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("GetLocations: Error encoding response: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    log.Printf("GetLocations: Successfully sent response")
}

// Additional helper function to check if location exists
func ValidateLocation(state, district, subdistrict, locality string) bool {
    var exists bool
    query := `
        SELECT EXISTS(
            SELECT 1 
            FROM villages 
            WHERE state = $1 
            AND district = $2 
            AND subdistrict = $3 
            AND locality = $4
        )`
    
    err := config.DB.QueryRow(query, state, district, subdistrict, locality).Scan(&exists)
    if err != nil {
        return false
    }
    return exists
}

// Function to get location details
func GetLocationDetails(state, district, subdistrict, locality string) (map[string]interface{}, error) {
    var details map[string]interface{}
    
    query := `
        SELECT row_to_json(t)
        FROM (
            SELECT 
                state,
                district,
                subdistrict,
                locality,
                latitude,
                longitude
            FROM villages
            WHERE state = $1 
            AND district = $2 
            AND subdistrict = $3 
            AND locality = $4
        ) t`

    err := config.DB.QueryRow(query, state, district, subdistrict, locality).Scan(&details)
    if err != nil {
        return nil, err
    }

    return details, nil
}

// Function to search locations
func SearchLocations(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Query string `json:"query"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    rows, err := config.DB.Query(`
        SELECT DISTINCT 
            state, 
            district, 
            subdistrict, 
            locality
        FROM villages
        WHERE 
            state ILIKE $1 OR 
            district ILIKE $1 OR 
            subdistrict ILIKE $1 OR 
            locality ILIKE $1
        LIMIT 10`,
        "%"+req.Query+"%")
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var results []map[string]string
    for rows.Next() {
        var state, district, subdistrict, locality string
        if err := rows.Scan(&state, &district, &subdistrict, &locality); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        results = append(results, map[string]string{
            "state": state,
            "district": district,
            "subdistrict": subdistrict,
            "locality": locality,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "results": results,
    })
}