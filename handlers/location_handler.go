package handlers

import (
    "encoding/json"
    "net/http"
    "village_site/config"
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
    var req LocationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    var response LocationResponse

    // If no state provided, return list of states
    if req.State == "" {
        rows, err := config.DB.Query(`
            SELECT DISTINCT state 
            FROM villages 
            WHERE state IS NOT NULL
            ORDER BY state`)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var states []string
        for rows.Next() {
            var state string
            if err := rows.Scan(&state); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            if state != "" {
                states = append(states, state)
            }
        }
        response.States = states

    } else if req.District == "" {
        // Get districts for state
        rows, err := config.DB.Query(`
            SELECT DISTINCT district 
            FROM villages 
            WHERE state = $1 
            AND district IS NOT NULL
            ORDER BY district`, req.State)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var districts []string
        for rows.Next() {
            var district string
            if err := rows.Scan(&district); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            if district != "" {
                districts = append(districts, district)
            }
        }
        response.Districts = districts

    } else if req.Subdistrict == "" {
        // Get subdistricts for district
        rows, err := config.DB.Query(`
            SELECT DISTINCT subdistrict 
            FROM villages 
            WHERE state = $1 
            AND district = $2 
            AND subdistrict IS NOT NULL
            ORDER BY subdistrict`, req.State, req.District)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var subdistricts []string
        for rows.Next() {
            var subdistrict string
            if err := rows.Scan(&subdistrict); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            if subdistrict != "" {
                subdistricts = append(subdistricts, subdistrict)
            }
        }
        response.Subdistricts = subdistricts

    } else {
        // Get villages/localities for subdistrict
        rows, err := config.DB.Query(`
            SELECT DISTINCT locality 
            FROM villages 
            WHERE state = $1 
            AND district = $2 
            AND subdistrict = $3 
            AND locality IS NOT NULL
            ORDER BY locality`, req.State, req.District, req.Subdistrict)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var villages []string
        for rows.Next() {
            var village string
            if err := rows.Scan(&village); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            if village != "" {
                villages = append(villages, village)
            }
        }
        response.Villages = villages
    }

    // Add cache control headers
    w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
    w.Header().Set("Content-Type", "application/json")
    
    // Return response
    if err := json.NewEncoder(w).Encode(response); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
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