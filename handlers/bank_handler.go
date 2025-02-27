package handlers

import (
    "database/sql"
    "encoding/json"
    "log"
    "net/http"
    "strings"
    "village_site/config"
)

// Struct definitions
type IFSCDetails struct {
    Bank       string `json:"bank"`
    IFSC       string `json:"ifsc"`
    Branch     string `json:"branch"`
    Address    string `json:"address"`
    BranchCity string `json:"branch_city"`
    District   string `json:"district"`
    State      string `json:"state"`
    Phone      string `json:"phone"`
    MICR       string `json:"micr,omitempty"`
    Website    string `json:"website,omitempty"`
    Email      string `json:"email,omitempty"`
}

type PinCodeDetails struct {
    OfficeName        string `json:"office_name"`
    Pincode          string `json:"pincode"`
    PostType         string `json:"post_type"`
    DeliveryStatus   string `json:"delivery_status"`
    DivisionName     string `json:"division_name"`
    RegionName       string `json:"region_name"`
    CircleName       string `json:"circle_name"`
    Taluk            string `json:"taluk"`
    District         string `json:"district"`
    State            string `json:"state"`
    Telephone        string `json:"telephone"`
    RelatedSuboffice string `json:"related_suboffice"`
    RelatedHeadoffice string `json:"related_headoffice"`
    StateCode        string `json:"state_code,omitempty"`
}

type PinStateResponse struct {
    States []string `json:"states"`
}

type PinDistrictResponse struct {
    Districts []string `json:"districts"`
}

type StateResponse struct {
    States []string `json:"states"`
}

type DistrictResponse struct {
    Districts []string `json:"districts"`
}

type BranchCityResponse struct {
    Cities []string `json:"cities"`
}

type BranchInfo struct {
    BranchName string `json:"branch_name"`
    IFSC       string `json:"ifsc"`
}

type BranchResponse struct {
    Branches []BranchInfo `json:"branches"`
}

type PostOfficeResponse struct {
    PostOffices []string `json:"post_offices"`
}

type BankStats struct {
    TotalBanks            int                `json:"total_banks"`
    TotalBranches        int                `json:"total_branches"`
    BanksWithWebsites    int                `json:"banks_with_websites"`
    BranchesWithMICR     int                `json:"branches_with_micr"`
    StateWiseCounts      map[string]int     `json:"state_wise_counts"`
}

// Get list of banks
func GetBankList(w http.ResponseWriter, r *http.Request) {
    log.Printf("GetBankList: Starting to fetch bank list")

    // Set response headers
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    w.Header().Set("Pragma", "no-cache")
    w.Header().Set("Expires", "0")

    // Check if DB is nil
    if config.DB == nil {
        log.Printf("GetBankList: Database connection is nil")
        http.Error(w, `{"error": "Database connection not initialized"}`, http.StatusInternalServerError)
        return
    }

    // Check DB connection
    if err := config.DB.Ping(); err != nil {
        log.Printf("GetBankList: Database ping failed: %v", err)
        http.Error(w, `{"error": "Database connection error"}`, http.StatusInternalServerError)
        return
    }

    // Check if table exists
    var tableExists bool
    err := config.DB.QueryRow(`
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_name = 'ifsc_details'
        )`).Scan(&tableExists)
    
    if err != nil {
        log.Printf("GetBankList: Error checking table existence: %v", err)
        http.Error(w, `{"error": "Error checking database structure"}`, http.StatusInternalServerError)
        return
    }

    if !tableExists {
        log.Printf("GetBankList: ifsc_details table does not exist")
        http.Error(w, `{"error": "Required table not found"}`, http.StatusInternalServerError)
        return
    }
    
    query := `
        SELECT DISTINCT bank 
        FROM ifsc_details 
        WHERE bank IS NOT NULL AND bank != ''
        ORDER BY bank`

    log.Printf("GetBankList: Executing query: %s", query)
    rows, err := config.DB.Query(query)
    if err != nil {
        log.Printf("GetBankList: Database error fetching banks: %v", err)
        http.Error(w, `{"error": "Error fetching banks"}`, http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var banks []string
    for rows.Next() {
        var bank string
        if err := rows.Scan(&bank); err != nil {
            log.Printf("GetBankList: Error scanning bank: %v", err)
            continue
        }
        if bank != "" {
            banks = append(banks, strings.TrimSpace(bank))
            log.Printf("GetBankList: Found bank: %s", bank)
        }
    }

    if err = rows.Err(); err != nil {
        log.Printf("GetBankList: Error iterating bank rows: %v", err)
        http.Error(w, `{"error": "Error processing banks"}`, http.StatusInternalServerError)
        return
    }

    if len(banks) == 0 {
        log.Printf("GetBankList: No banks found in database")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "banks": []string{},
            "message": "No banks found",
        })
        return
    }

    log.Printf("GetBankList: Found %d banks, sending response", len(banks))
    response := map[string]interface{}{
        "banks": banks,
        "total": len(banks),
    }
    
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("GetBankList: Error encoding response: %v", err)
        http.Error(w, `{"error": "Error encoding response"}`, http.StatusInternalServerError)
        return
    }
    log.Printf("GetBankList: Response sent successfully")
}

// Get states by bank
func GetBankStates(w http.ResponseWriter, r *http.Request) {
    bank := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("bank")))
    if bank == "" {
        http.Error(w, "Bank name is required", http.StatusBadRequest)
        return
    }

    query := `
        SELECT DISTINCT state 
        FROM ifsc_details 
        WHERE UPPER(bank) = $1 
        ORDER BY state`

    rows, err := config.DB.Query(query, bank)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching states", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var states []string
    for rows.Next() {
        var state string
        if err := rows.Scan(&state); err != nil {
            continue
        }
        states = append(states, state)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(StateResponse{States: states})
}

// Get districts by bank and state
func GetBankDistricts(w http.ResponseWriter, r *http.Request) {
    bank := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("bank")))
    state := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("state")))

    if bank == "" || state == "" {
        http.Error(w, "Bank name and state are required", http.StatusBadRequest)
        return
    }

    query := `
        SELECT DISTINCT district 
        FROM ifsc_details 
        WHERE UPPER(bank) = $1 
        AND UPPER(state) = $2 
        ORDER BY district`

    rows, err := config.DB.Query(query, bank, state)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching districts", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var districts []string
    for rows.Next() {
        var district string
        if err := rows.Scan(&district); err != nil {
            continue
        }
        districts = append(districts, district)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(DistrictResponse{Districts: districts})
}

// Get branch cities
func GetBankBranchCities(w http.ResponseWriter, r *http.Request) {
    bank := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("bank")))
    state := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("state")))
    district := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("district")))

    if bank == "" || state == "" || district == "" {
        http.Error(w, "Bank name, state, and district are required", http.StatusBadRequest)
        return
    }

    query := `
        SELECT DISTINCT branch_city 
        FROM ifsc_details 
        WHERE UPPER(bank) = $1 
        AND UPPER(state) = $2 
        AND UPPER(district) = $3 
        ORDER BY branch_city`

    rows, err := config.DB.Query(query, bank, state, district)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching cities", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var cities []string
    for rows.Next() {
        var city string
        if err := rows.Scan(&city); err != nil {
            continue
        }
        cities = append(cities, city)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(BranchCityResponse{Cities: cities})
}

// Get bank branches with IFSC codes
func GetBankBranches(w http.ResponseWriter, r *http.Request) {
    bank := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("bank")))
    state := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("state")))
    district := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("district")))
    city := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("city")))

    if bank == "" || state == "" || district == "" || city == "" {
        http.Error(w, "Bank name, state, district, and city are required", http.StatusBadRequest)
        return
    }

    query := `
        SELECT branch, ifsc 
        FROM ifsc_details 
        WHERE UPPER(bank) = $1 
        AND UPPER(state) = $2 
        AND UPPER(district) = $3 
        AND UPPER(branch_city) = $4 
        ORDER BY branch`

    rows, err := config.DB.Query(query, bank, state, district, city)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching branches", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var branches []BranchInfo
    for rows.Next() {
        var branch BranchInfo
        err := rows.Scan(&branch.BranchName, &branch.IFSC)
        if err != nil {
            log.Printf("Error scanning branch: %v", err)
            continue
        }
        branches = append(branches, branch)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(BranchResponse{Branches: branches})
}

// Get IFSC details
func GetIFSCDetails(w http.ResponseWriter, r *http.Request) {
    ifsc := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("ifsc")))

    if ifsc == "" {
        http.Error(w, "IFSC code is required", http.StatusBadRequest)
        return
    }

    log.Printf("Searching for IFSC: %s", ifsc)

    query := `
        SELECT i.bank, i.ifsc, i.branch, i.address, i.branch_city, 
               i.district, i.state, i.phone, 
               COALESCE(m.micr, '') as micr,
               COALESCE(b.website, '') as website,
               COALESCE(b.email, '') as email
        FROM ifsc_details i
        LEFT JOIN micr_details m ON i.ifsc = m.ifsc
        LEFT JOIN bank_details b ON i.bank = b.bank
        WHERE i.ifsc = $1`

    var details IFSCDetails
    err := config.DB.QueryRow(query, ifsc).Scan(
        &details.Bank,
        &details.IFSC,
        &details.Branch,
        &details.Address,
        &details.BranchCity,
        &details.District,
        &details.State,
        &details.Phone,
        &details.MICR,
        &details.Website,
        &details.Email,
    )

    if err == sql.ErrNoRows {
        log.Printf("No IFSC details found for: %s", ifsc)
        http.Error(w, "IFSC details not found", http.StatusNotFound)
        return
    } else if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching IFSC details", http.StatusInternalServerError)
        return
    }

    log.Printf("Found IFSC details: %+v", details)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(details)
}

// Debug IFSC data
func DebugIFSCData(w http.ResponseWriter, r *http.Request) {
    ifsc := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("ifsc")))
    
    if ifsc == "" {
        http.Error(w, "IFSC code is required", http.StatusBadRequest)
        return
    }

    var debug struct {
        IFSCDetails  []map[string]interface{} `json:"ifsc_details"`
        MICRDetails  []map[string]interface{} `json:"micr_details"`
        BankDetails  []map[string]interface{} `json:"bank_details"`
    }

    rows, err := config.DB.Query(`
        SELECT * FROM ifsc_details WHERE UPPER(ifsc) = $1`, ifsc)
    if err != nil {
        log.Printf("Error querying ifsc_details: %v", err)
    } else {
        debug.IFSCDetails = scanRowsToMap(rows)
        rows.Close()
    }

    rows, err = config.DB.Query(`
        SELECT * FROM micr_details WHERE UPPER(ifsc) = $1`, ifsc)
    if err != nil {
        log.Printf("Error querying micr_details: %v", err)
    } else {
        debug.MICRDetails = scanRowsToMap(rows)
        rows.Close()
    }

    if len(debug.IFSCDetails) > 0 {
        bank, ok := debug.IFSCDetails[0]["bank"].(string)
        if ok {
            rows, err = config.DB.Query(`
                SELECT * FROM bank_details WHERE UPPER(bank) = UPPER($1)`, bank)
            if err != nil {
                log.Printf("Error querying bank_details: %v", err)
            } else {
                debug.BankDetails = scanRowsToMap(rows)
                rows.Close()
            }
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(debug)
}

// Helper function to scan rows into map
func scanRowsToMap(rows *sql.Rows) []map[string]interface{} {
    var result []map[string]interface{}
    
    columns, err := rows.Columns()
    if err != nil {
        return result
    }

    for rows.Next() {
        values := make([]interface{}, len(columns))
        valuePointers := make([]interface{}, len(columns))
        for i := range values {
            valuePointers[i] = &values[i]
        }

        err := rows.Scan(valuePointers...)
        if err != nil {
            continue
        }

        entry := make(map[string]interface{})
        for i, col := range columns {
            val := values[i]
            if b, ok := val.([]byte); ok {
                entry[col] = string(b)
            } else {
                entry[col] = val
            }
        }
        
        result = append(result, entry)
    }
    
    return result
}

// Get all states for PIN codes
func GetPinStates(w http.ResponseWriter, r *http.Request) {
    query := `
        SELECT DISTINCT state 
        FROM pin_details 
        ORDER BY state`

    rows, err := config.DB.Query(query)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching states", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var states []string
    for rows.Next() {
        var state string
        if err := rows.Scan(&state); err != nil {
            continue
        }
        states = append(states, state)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(PinStateResponse{States: states})
}

// Get districts by state for PIN codes
func GetPinDistricts(w http.ResponseWriter, r *http.Request) {
    state := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("state")))
    if state == "" {
        http.Error(w, "State is required", http.StatusBadRequest)
        return
    }

    query := `
        SELECT DISTINCT district 
        FROM pin_details 
        WHERE UPPER(state) = $1 
        ORDER BY district`

    rows, err := config.DB.Query(query, state)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching districts", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var districts []string
    for rows.Next() {
        var district string
        if err := rows.Scan(&district); err != nil {
            continue
        }
        districts = append(districts, district)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(PinDistrictResponse{Districts: districts})
}

// Get post offices by state and district
func GetPostOffices(w http.ResponseWriter, r *http.Request) {
    state := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("state")))
    district := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("district")))

    if state == "" || district == "" {
        http.Error(w, "State and district are required", http.StatusBadRequest)
        return
    }

    query := `
        SELECT DISTINCT officename 
        FROM pin_details 
        WHERE UPPER(state) = $1 
        AND UPPER(district) = $2 
        ORDER BY officename`

    rows, err := config.DB.Query(query, state, district)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching post offices", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var offices []string
    for rows.Next() {
        var office string
        if err := rows.Scan(&office); err != nil {
            continue
        }
        offices = append(offices, office)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(PostOfficeResponse{PostOffices: offices})
}

// Get PIN code details
func GetPinCodeDetails(w http.ResponseWriter, r *http.Request) {
    state := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("state")))
    district := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("district")))
    officeName := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("office")))

    if state == "" || district == "" || officeName == "" {
        http.Error(w, "State, district, and office name are required", http.StatusBadRequest)
        return
    }

    query := `
        SELECT p.officename, p.pincode, p.post_type, p.delivery_status,
               p.divionname, p.regionname, p.circlename, p.taluk,
               p.district, p.state, p.telephone, p.related_suboffice,
               p.related_headoffice, s.state_code
        FROM pin_details p
        LEFT JOIN state_details s ON UPPER(p.state) = UPPER(s.state) 
            AND UPPER(p.district) = UPPER(s.district)
        WHERE UPPER(p.state) = $1
        AND UPPER(p.district) = $2
        AND UPPER(p.officename) = $3`

    var details PinCodeDetails
    err := config.DB.QueryRow(query, state, district, officeName).Scan(
        &details.OfficeName,
        &details.Pincode,
        &details.PostType,
        &details.DeliveryStatus,
        &details.DivisionName,
        &details.RegionName,
        &details.CircleName,
        &details.Taluk,
        &details.District,
        &details.State,
        &details.Telephone,
        &details.RelatedSuboffice,
        &details.RelatedHeadoffice,
        &details.StateCode,
    )

    if err == sql.ErrNoRows {
        http.Error(w, "PIN code details not found", http.StatusNotFound)
        return
    } else if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching PIN code details", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(details)
}

// GetBankStats returns statistics about banks and branches
func GetBankStats(w http.ResponseWriter, r *http.Request) {
    log.Printf("GetBankStats: Starting to fetch bank statistics")

    // Set response headers
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
    w.Header().Set("Pragma", "no-cache")
    w.Header().Set("Expires", "0")

    if config.DB == nil {
        log.Printf("GetBankStats: Database connection is nil")
        http.Error(w, `{"error": "Database connection not initialized"}`, http.StatusInternalServerError)
        return
    }

    // Check DB connection
    if err := config.DB.Ping(); err != nil {
        log.Printf("GetBankStats: Database ping failed: %v", err)
        http.Error(w, `{"error": "Database connection error"}`, http.StatusInternalServerError)
        return
    }

    stats := BankStats{
        StateWiseCounts: make(map[string]int),
    }

    // Get total number of unique banks
    err := config.DB.QueryRow(`
        SELECT COUNT(DISTINCT bank) 
        FROM ifsc_details 
        WHERE bank IS NOT NULL AND bank != ''`).Scan(&stats.TotalBanks)
    if err != nil {
        log.Printf("GetBankStats: Error getting total banks: %v", err)
        http.Error(w, `{"error": "Error fetching bank statistics"}`, http.StatusInternalServerError)
        return
    }

    // Get total number of branches
    err = config.DB.QueryRow(`
        SELECT COUNT(*) 
        FROM ifsc_details`).Scan(&stats.TotalBranches)
    if err != nil {
        log.Printf("GetBankStats: Error getting total branches: %v", err)
        http.Error(w, `{"error": "Error fetching bank statistics"}`, http.StatusInternalServerError)
        return
    }

    // Get number of banks with websites
    err = config.DB.QueryRow(`
        SELECT COUNT(*) 
        FROM bank_details 
        WHERE website IS NOT NULL AND website != ''`).Scan(&stats.BanksWithWebsites)
    if err != nil {
        log.Printf("GetBankStats: Error getting banks with websites: %v", err)
        http.Error(w, `{"error": "Error fetching bank statistics"}`, http.StatusInternalServerError)
        return
    }

    // Get number of branches with MICR codes
    err = config.DB.QueryRow(`
        SELECT COUNT(*) 
        FROM micr_details`).Scan(&stats.BranchesWithMICR)
    if err != nil {
        log.Printf("GetBankStats: Error getting branches with MICR: %v", err)
        http.Error(w, `{"error": "Error fetching bank statistics"}`, http.StatusInternalServerError)
        return
    }

    // Get state-wise branch counts with normalized state names
    rows, err := config.DB.Query(`
        SELECT UPPER(TRIM(state)) as state, COUNT(*) as count 
        FROM ifsc_details 
        WHERE state IS NOT NULL AND TRIM(state) != '' 
        GROUP BY UPPER(TRIM(state)) 
        ORDER BY count DESC`)
    if err != nil {
        log.Printf("GetBankStats: Error getting state-wise counts: %v", err)
        http.Error(w, `{"error": "Error fetching bank statistics"}`, http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    // Map for state name normalization
    stateNormalization := map[string]string{
        "UTTARANCHAL":     "UTTARAKHAND",
        "UttaraKhand":     "UTTARAKHAND",
        "Uttarakhand":     "UTTARAKHAND",
        "Gujarat":         "GUJARAT",
        "Madhya Pradesh":  "MADHYA PRADESH",
        "Tamil Nadu":      "TAMIL NADU",
        "Telangana":       "TELANGANA",
        "Tripura":         "TRIPURA",
    }

    for rows.Next() {
        var state string
        var count int
        if err := rows.Scan(&state, &count); err != nil {
            log.Printf("GetBankStats: Error scanning state count: %v", err)
            continue
        }

        // Normalize state name if a mapping exists
        if normalizedState, exists := stateNormalization[state]; exists {
            state = normalizedState
        }

        // Add count to existing state or create new entry
        stats.StateWiseCounts[state] += count
    }

    if err = rows.Err(); err != nil {
        log.Printf("GetBankStats: Error iterating state counts: %v", err)
        http.Error(w, `{"error": "Error processing bank statistics"}`, http.StatusInternalServerError)
        return
    }

    log.Printf("GetBankStats: Successfully fetched statistics")
    if err := json.NewEncoder(w).Encode(stats); err != nil {
        log.Printf("GetBankStats: Error encoding response: %v", err)
        http.Error(w, `{"error": "Error encoding response"}`, http.StatusInternalServerError)
        return
    }
}