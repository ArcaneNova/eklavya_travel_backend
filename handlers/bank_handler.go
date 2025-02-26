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

// Get list of banks
func GetBankList(w http.ResponseWriter, r *http.Request) {
    log.Printf("GetBankList: Starting to fetch bank list")

    // Check if DB is nil
    if config.DB == nil {
        log.Printf("GetBankList: Database connection is nil")
        http.Error(w, "Database connection not initialized", http.StatusInternalServerError)
        return
    }

    // Check DB connection
    if err := config.DB.Ping(); err != nil {
        log.Printf("GetBankList: Database ping failed: %v", err)
        http.Error(w, "Database connection error", http.StatusInternalServerError)
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
        http.Error(w, "Error checking database structure", http.StatusInternalServerError)
        return
    }

    if !tableExists {
        log.Printf("GetBankList: ifsc_details table does not exist")
        http.Error(w, "Required table not found", http.StatusInternalServerError)
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
        http.Error(w, "Error fetching banks", http.StatusInternalServerError)
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
            banks = append(banks, bank)
            log.Printf("GetBankList: Found bank: %s", bank)
        }
    }

    if err = rows.Err(); err != nil {
        log.Printf("GetBankList: Error iterating bank rows: %v", err)
        http.Error(w, "Error processing banks", http.StatusInternalServerError)
        return
    }

    if len(banks) == 0 {
        log.Printf("GetBankList: No banks found in database")
        http.Error(w, "No banks found", http.StatusNotFound)
        return
    }

    log.Printf("GetBankList: Found %d banks, sending response", len(banks))
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(map[string]interface{}{
        "banks": banks,
    }); err != nil {
        log.Printf("GetBankList: Error encoding response: %v", err)
        http.Error(w, "Error encoding response", http.StatusInternalServerError)
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

// GetBankByIFSC handles retrieving bank details by IFSC code
func GetBankByIFSC(w http.ResponseWriter, r *http.Request) {
    ifsc := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("ifsc")))
    if ifsc == "" {
        http.Error(w, "IFSC code is required", http.StatusBadRequest)
        return
    }

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
        http.Error(w, "IFSC details not found", http.StatusNotFound)
        return
    } else if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching IFSC details", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
    json.NewEncoder(w).Encode(details)
}

// SearchBanks handles searching banks by name or location
func SearchBanks(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" {
        http.Error(w, "Search query is required", http.StatusBadRequest)
        return
    }

    sqlQuery := `
        SELECT DISTINCT i.bank, i.branch, i.ifsc, i.branch_city, i.state
        FROM ifsc_details i
        WHERE 
            LOWER(i.bank) LIKE LOWER($1) OR
            LOWER(i.branch) LIKE LOWER($1) OR
            LOWER(i.ifsc) LIKE LOWER($1) OR
            LOWER(i.branch_city) LIKE LOWER($1)
        ORDER BY 
            CASE 
                WHEN LOWER(i.ifsc) = LOWER($2) THEN 1
                WHEN LOWER(i.bank) = LOWER($2) THEN 2
                WHEN LOWER(i.branch) = LOWER($2) THEN 3
                ELSE 4
            END,
            i.bank, i.branch
        LIMIT 50`

    rows, err := config.DB.Query(sqlQuery, "%"+query+"%", query)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error searching banks", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var results []map[string]string
    for rows.Next() {
        var bank, branch, ifsc, city, state string
        if err := rows.Scan(&bank, &branch, &ifsc, &city, &state); err != nil {
            continue
        }
        results = append(results, map[string]string{
            "bank": bank,
            "branch": branch,
            "ifsc": ifsc,
            "city": city,
            "state": state,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes
    json.NewEncoder(w).Encode(map[string]interface{}{
        "results": results,
        "query": query,
    })
}

// GetBankStats handles retrieving statistics about banks
func GetBankStats(w http.ResponseWriter, r *http.Request) {
    var stats struct {
        TotalBanks     int            `json:"total_banks"`
        TotalBranches  int            `json:"total_branches"`
        StateWise      map[string]int `json:"state_wise"`
        BankWise       map[string]int `json:"bank_wise"`
        WithMICR       int            `json:"with_micr"`
        WithWebsite    int            `json:"with_website"`
    }

    // Get total banks count
    err := config.DB.QueryRow(`
        SELECT COUNT(DISTINCT bank) 
        FROM ifsc_details`).Scan(&stats.TotalBanks)
    if err != nil {
        log.Printf("Error getting total banks: %v", err)
        http.Error(w, "Error fetching bank stats", http.StatusInternalServerError)
        return
    }

    // Get total branches count
    err = config.DB.QueryRow(`
        SELECT COUNT(*) 
        FROM ifsc_details`).Scan(&stats.TotalBranches)
    if err != nil {
        log.Printf("Error getting total branches: %v", err)
        http.Error(w, "Error fetching bank stats", http.StatusInternalServerError)
        return
    }

    // Get state-wise branch counts
    rows, err := config.DB.Query(`
        SELECT state, COUNT(*) 
        FROM ifsc_details 
        GROUP BY state 
        ORDER BY state`)
    if err != nil {
        log.Printf("Error getting state-wise counts: %v", err)
        http.Error(w, "Error fetching state-wise stats", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    stats.StateWise = make(map[string]int)
    for rows.Next() {
        var state string
        var count int
        if err := rows.Scan(&state, &count); err != nil {
            continue
        }
        stats.StateWise[state] = count
    }

    // Get bank-wise branch counts
    rows, err = config.DB.Query(`
        SELECT bank, COUNT(*) 
        FROM ifsc_details 
        GROUP BY bank 
        ORDER BY COUNT(*) DESC`)
    if err != nil {
        log.Printf("Error getting bank-wise counts: %v", err)
        http.Error(w, "Error fetching bank-wise stats", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    stats.BankWise = make(map[string]int)
    for rows.Next() {
        var bank string
        var count int
        if err := rows.Scan(&bank, &count); err != nil {
            continue
        }
        stats.BankWise[bank] = count
    }

    // Get count of branches with MICR codes
    err = config.DB.QueryRow(`
        SELECT COUNT(*) 
        FROM ifsc_details i 
        INNER JOIN micr_details m 
        ON i.ifsc = m.ifsc`).Scan(&stats.WithMICR)
    if err != nil {
        stats.WithMICR = 0
    }

    // Get count of banks with websites
    err = config.DB.QueryRow(`
        SELECT COUNT(*) 
        FROM ifsc_details i 
        INNER JOIN bank_details b 
        ON i.bank = b.bank 
        WHERE b.website IS NOT NULL 
        AND b.website != ''`).Scan(&stats.WithWebsite)
    if err != nil {
        stats.WithWebsite = 0
    }

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
    json.NewEncoder(w).Encode(stats)
}

// SearchPinCodes handles searching PIN codes and post offices
func SearchPinCodes(w http.ResponseWriter, r *http.Request) {
    query := strings.TrimSpace(r.URL.Query().Get("q"))
    if query == "" {
        http.Error(w, "Search query is required", http.StatusBadRequest)
        return
    }

    sqlQuery := `
        SELECT p.officename, p.pincode, p.district, p.state, p.post_type
        FROM pin_details p
        WHERE 
            p.pincode LIKE $1 OR
            LOWER(p.officename) LIKE LOWER($2) OR
            LOWER(p.district) LIKE LOWER($2) OR
            LOWER(p.state) LIKE LOWER($2)
        ORDER BY 
            CASE 
                WHEN p.pincode = $3 THEN 1
                WHEN LOWER(p.officename) = LOWER($3) THEN 2
                ELSE 3
            END,
            p.state, p.district, p.officename
        LIMIT 50`

    rows, err := config.DB.Query(sqlQuery, query+"%", "%"+query+"%", query)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error searching PIN codes", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var results []map[string]string
    for rows.Next() {
        var officeName, pincode, district, state, postType string
        if err := rows.Scan(&officeName, &pincode, &district, &state, &postType); err != nil {
            continue
        }
        results = append(results, map[string]string{
            "office_name": officeName,
            "pincode": pincode,
            "district": district,
            "state": state,
            "post_type": postType,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes
    json.NewEncoder(w).Encode(map[string]interface{}{
        "results": results,
        "query": query,
    })
}

// GetPinCodeStats handles retrieving statistics about PIN codes and post offices
func GetPinCodeStats(w http.ResponseWriter, r *http.Request) {
    var stats struct {
        TotalPinCodes    int            `json:"total_pincodes"`
        TotalPostOffices int            `json:"total_post_offices"`
        StateWise        map[string]int `json:"state_wise"`
        PostTypeWise     map[string]int `json:"post_type_wise"`
        DeliveryWise     map[string]int `json:"delivery_wise"`
        WithPhone        int            `json:"with_phone"`
        WithSuboffice    int            `json:"with_suboffice"`
        WithHeadoffice   int            `json:"with_headoffice"`
    }

    // Get total PIN codes count
    err := config.DB.QueryRow(`
        SELECT COUNT(DISTINCT pincode) 
        FROM pin_details`).Scan(&stats.TotalPinCodes)
    if err != nil {
        log.Printf("Error getting total PIN codes: %v", err)
        http.Error(w, "Error fetching PIN code stats", http.StatusInternalServerError)
        return
    }

    // Get total post offices count
    err = config.DB.QueryRow(`
        SELECT COUNT(*) 
        FROM pin_details`).Scan(&stats.TotalPostOffices)
    if err != nil {
        log.Printf("Error getting total post offices: %v", err)
        http.Error(w, "Error fetching post office stats", http.StatusInternalServerError)
        return
    }

    // Get state-wise post office counts
    rows, err := config.DB.Query(`
        SELECT state, COUNT(*) 
        FROM pin_details 
        GROUP BY state 
        ORDER BY state`)
    if err != nil {
        log.Printf("Error getting state-wise counts: %v", err)
        http.Error(w, "Error fetching state-wise stats", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    stats.StateWise = make(map[string]int)
    for rows.Next() {
        var state string
        var count int
        if err := rows.Scan(&state, &count); err != nil {
            continue
        }
        stats.StateWise[state] = count
    }

    // Get post type-wise counts
    rows, err = config.DB.Query(`
        SELECT post_type, COUNT(*) 
        FROM pin_details 
        GROUP BY post_type 
        ORDER BY COUNT(*) DESC`)
    if err != nil {
        log.Printf("Error getting post type-wise counts: %v", err)
        http.Error(w, "Error fetching post type stats", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    stats.PostTypeWise = make(map[string]int)
    for rows.Next() {
        var postType string
        var count int
        if err := rows.Scan(&postType, &count); err != nil {
            continue
        }
        stats.PostTypeWise[postType] = count
    }

    // Get delivery status-wise counts
    rows, err = config.DB.Query(`
        SELECT delivery_status, COUNT(*) 
        FROM pin_details 
        GROUP BY delivery_status 
        ORDER BY COUNT(*) DESC`)
    if err != nil {
        log.Printf("Error getting delivery status counts: %v", err)
        http.Error(w, "Error fetching delivery status stats", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    stats.DeliveryWise = make(map[string]int)
    for rows.Next() {
        var status string
        var count int
        if err := rows.Scan(&status, &count); err != nil {
            continue
        }
        stats.DeliveryWise[status] = count
    }

    // Get count of post offices with phone numbers
    err = config.DB.QueryRow(`
        SELECT COUNT(*) 
        FROM pin_details 
        WHERE telephone IS NOT NULL 
        AND telephone != ''`).Scan(&stats.WithPhone)
    if err != nil {
        stats.WithPhone = 0
    }

    // Get count of post offices with sub offices
    err = config.DB.QueryRow(`
        SELECT COUNT(*) 
        FROM pin_details 
        WHERE related_suboffice IS NOT NULL 
        AND related_suboffice != ''`).Scan(&stats.WithSuboffice)
    if err != nil {
        stats.WithSuboffice = 0
    }

    // Get count of post offices with head offices
    err = config.DB.QueryRow(`
        SELECT COUNT(*) 
        FROM pin_details 
        WHERE related_headoffice IS NOT NULL 
        AND related_headoffice != ''`).Scan(&stats.WithHeadoffice)
    if err != nil {
        stats.WithHeadoffice = 0
    }

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
    json.NewEncoder(w).Encode(stats)
}