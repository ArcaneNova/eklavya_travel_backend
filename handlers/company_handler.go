package handlers

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "sort"
    "strconv"
    "strings"
    "unicode"
    "village_site/config"
)

// Struct definitions
type Company struct {
    ID                              string  `json:"id"`
    CIN                            string  `json:"cin"`
    URLTitle                       string  `json:"url_title"`
    CompanyName                    string  `json:"company_name"`
    CompanyROCCode                 string  `json:"company_roc_code"`
    CompanyCategory                string  `json:"company_category"`
    CompanySubcategory            string  `json:"company_subcategory"`
    CompanyClass                   string  `json:"company_class"`
    AuthorizedCapital             float64 `json:"authorized_capital"`
    PaidupCapital                 float64 `json:"paidup_capital"`
    CompanyRegDate                string  `json:"company_reg_date"`
    RegOfficeAddress              string  `json:"reg_office_address"`
    ListingStatus                 string  `json:"listing_status"`
    CompanyStatus                 string  `json:"company_status"`
    CompanyStateCode              string  `json:"company_state_code"`
    CompanyCountry                string  `json:"company_country"`
    NICCode                       string  `json:"nic_code"`
    CompanyIndustrialClassification string `json:"company_industrial_classification"`
}

type CompanyListItem struct {
    CompanyName      string `json:"company_name"`
    CIN             string `json:"cin"`
    CompanyROCCode  string `json:"company_roc_code"`
    CompanyStateCode string `json:"company_state_code"`
    URLTitle        string `json:"url_title"`
}

type CompanyListResponse struct {
    Companies []CompanyListItem `json:"companies"`
    Total     int              `json:"total"`
    Pages     int              `json:"total_pages"`
    Current   int              `json:"current_page"`
}

type NearbyCompany struct {
    CompanyName      string  `json:"company_name"`
    CIN             string  `json:"cin"`
    URLTitle        string  `json:"url_title"`
    Address         string  `json:"address"`
    SimilarityScore float64 `json:"similarity_score"`
}

type CompanyResponse struct {
    Company         Company         `json:"company"`
    NearbyCompanies []NearbyCompany `json:"nearby_companies"`
}

// Helper functions
func cleanText(s string) string {
    cleaned := strings.Map(func(r rune) rune {
        if unicode.IsPrint(r) || unicode.IsSpace(r) {
            return r
        }
        return -1
    }, s)
    return strings.TrimSpace(strings.Join(strings.Fields(cleaned), " "))
}

func calculateAddressSimilarity(addr1, addr2 string) float64 {
    addr1 = cleanText(strings.ToLower(addr1))
    addr2 = cleanText(strings.ToLower(addr2))

    words1 := strings.Fields(addr1)
    words2 := strings.Fields(addr2)

    freq1 := make(map[string]int)
    freq2 := make(map[string]int)

    for _, word := range words1 {
        freq1[word]++
    }
    for _, word := range words2 {
        freq2[word]++
    }

    matchCount := 0
    for word, count1 := range freq1 {
        if count2, exists := freq2[word]; exists {
            matchCount += min(count1, count2)
        }
    }

    totalWords := len(words1) + len(words2) - matchCount
    if totalWords == 0 {
        return 0
    }
    return float64(matchCount) / float64(totalWords)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

// Handler functions
func GetCompany(w http.ResponseWriter, r *http.Request) {
    urlTitle := r.URL.Query().Get("url_title")
    cin := strings.ToUpper(r.URL.Query().Get("cin"))

    if urlTitle == "" && cin == "" {
        http.Error(w, "URL title or CIN is required", http.StatusBadRequest)
        return
    }

    query := `
        SELECT id, cin, url_title, 
               COALESCE(NULLIF(regexp_replace(company_name, '[^\x20-\x7E]', '', 'g'), ''), company_name) as company_name,
               company_roc_code, company_category,
               company_subcategory, company_class, authorized_capital,
               paidup_capital, company_reg_date, 
               COALESCE(NULLIF(regexp_replace(reg_office_address, '[^\x20-\x7E]', '', 'g'), ''), reg_office_address) as reg_office_address,
               listing_status, company_status, company_state_code,
               company_country, nic_code, company_industrial_classification
        FROM companies_data
        WHERE url_title = $1 OR cin = $2`

    var company Company
    err := config.DB.QueryRow(query, urlTitle, cin).Scan(
        &company.ID, &company.CIN, &company.URLTitle, &company.CompanyName,
        &company.CompanyROCCode, &company.CompanyCategory,
        &company.CompanySubcategory, &company.CompanyClass,
        &company.AuthorizedCapital, &company.PaidupCapital,
        &company.CompanyRegDate, &company.RegOfficeAddress,
        &company.ListingStatus, &company.CompanyStatus,
        &company.CompanyStateCode, &company.CompanyCountry,
        &company.NICCode, &company.CompanyIndustrialClassification,
    )

    if err == sql.ErrNoRows {
        http.Error(w, "Company not found", http.StatusNotFound)
        return
    } else if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching company data", http.StatusInternalServerError)
        return
    }

    company.CompanyName = cleanText(company.CompanyName)
    company.RegOfficeAddress = cleanText(company.RegOfficeAddress)

    // Get nearby companies
    nearbyQuery := `
        SELECT 
            COALESCE(NULLIF(regexp_replace(company_name, '[^\x20-\x7E]', '', 'g'), ''), company_name) as company_name,
            cin,
            url_title,
            COALESCE(NULLIF(regexp_replace(reg_office_address, '[^\x20-\x7E]', '', 'g'), ''), reg_office_address) as reg_office_address
        FROM companies_data
        WHERE company_state_code = $1
          AND cin != $2
        LIMIT 100`

    rows, err := config.DB.Query(nearbyQuery, company.CompanyStateCode, company.CIN)
    if err != nil {
        log.Printf("Error fetching nearby companies: %v", err)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(CompanyResponse{Company: company})
        return
    }
    defer rows.Close()

    var nearbyCompanies []NearbyCompany
    for rows.Next() {
        var nearby NearbyCompany
        err := rows.Scan(
            &nearby.CompanyName,
            &nearby.CIN,
            &nearby.URLTitle,
            &nearby.Address,
        )
        if err != nil {
            continue
        }

        nearby.SimilarityScore = calculateAddressSimilarity(
            company.RegOfficeAddress,
            nearby.Address,
        )

        if nearby.SimilarityScore > 0.3 {
            nearbyCompanies = append(nearbyCompanies, nearby)
        }
    }

    sort.Slice(nearbyCompanies, func(i, j int) bool {
        return nearbyCompanies[i].SimilarityScore > nearbyCompanies[j].SimilarityScore
    })

    if len(nearbyCompanies) > 30 {
        nearbyCompanies = nearbyCompanies[:30]
    }

    response := CompanyResponse{
        Company:         company,
        NearbyCompanies: nearbyCompanies,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func ListCompanies(w http.ResponseWriter, r *http.Request) {
    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    if page < 1 {
        page = 1
    }
    
    limit := 50
    offset := (page - 1) * limit

    var total int
    err := config.DB.QueryRow("SELECT COUNT(*) FROM companies_data").Scan(&total)
    if err != nil {
        log.Printf("Error counting companies: %v", err)
        http.Error(w, "Error fetching companies", http.StatusInternalServerError)
        return
    }

    query := `
        SELECT 
            COALESCE(NULLIF(regexp_replace(company_name, '[^\x20-\x7E]', '', 'g'), ''), company_name) as company_name,
            cin, 
            company_roc_code, 
            company_state_code, 
            url_title
        FROM companies_data
        ORDER BY company_name
        LIMIT $1 OFFSET $2`

    rows, err := config.DB.Query(query, limit, offset)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error fetching companies", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var companies []CompanyListItem
    for rows.Next() {
        var company CompanyListItem
        err := rows.Scan(
            &company.CompanyName,
            &company.CIN,
            &company.CompanyROCCode,
            &company.CompanyStateCode,
            &company.URLTitle,
        )
        if err != nil {
            log.Printf("Error scanning company: %v", err)
            continue
        }
        company.CompanyName = cleanText(company.CompanyName)
        companies = append(companies, company)
    }

    response := CompanyListResponse{
        Companies: companies,
        Total:    total,
        Pages:    (total + limit - 1) / limit,
        Current:  page,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func SearchCompanies(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" {
        http.Error(w, "Search query is required", http.StatusBadRequest)
        return
    }

    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    if page < 1 {
        page = 1
    }
    
    limit := 50
    offset := (page - 1) * limit

    searchQuery := cleanText(query)
    searchTerms := strings.Split(searchQuery, " ")
    
    conditions := []string{}
    args := []interface{}{}
    argCount := 1

    cinTerm := strings.ToUpper(searchQuery)
    conditions = append(conditions, fmt.Sprintf("cin LIKE $%d", argCount))
    args = append(args, cinTerm+"%")
    argCount++

    nameConditions := []string{}
    for _, term := range searchTerms {
        if len(term) >= 3 {
            nameConditions = append(nameConditions, 
                fmt.Sprintf("company_name ILIKE $%d", argCount))
            args = append(args, "%"+term+"%")
            argCount++
        }
    }
    if len(nameConditions) > 0 {
        conditions = append(conditions, "("+strings.Join(nameConditions, " AND ")+")")
    }

    whereClause := strings.Join(conditions, " OR ")
    countQuery := fmt.Sprintf(`
        SELECT COUNT(*) 
        FROM companies_data 
        WHERE %s`, whereClause)

    var total int
    err := config.DB.QueryRow(countQuery, args...).Scan(&total)
    if err != nil {
        log.Printf("Error counting search results: %v", err)
        http.Error(w, "Error searching companies", http.StatusInternalServerError)
        return
    }

    args = append(args, limit, offset)
    searchSQL := fmt.Sprintf(`
        SELECT 
            COALESCE(NULLIF(regexp_replace(company_name, '[^\x20-\x7E]', '', 'g'), ''), company_name) as company_name,
            cin, 
            company_roc_code, 
            company_state_code, 
            url_title
        FROM companies_data
        WHERE %s
        ORDER BY company_name
        LIMIT $%d OFFSET $%d`,
        whereClause, argCount, argCount+1)

    rows, err := config.DB.Query(searchSQL, args...)
    if err != nil {
        log.Printf("Database error: %v", err)
        http.Error(w, "Error searching companies", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var companies []CompanyListItem
    for rows.Next() {
        var company CompanyListItem
        err := rows.Scan(
            &company.CompanyName,
            &company.CIN,
            &company.CompanyROCCode,
            &company.CompanyStateCode,
            &company.URLTitle,
        )
        if err != nil {
            log.Printf("Error scanning company: %v", err)
            continue
        }
        company.CompanyName = cleanText(company.CompanyName)
        companies = append(companies, company)
    }

    response := CompanyListResponse{
        Companies: companies,
        Total:    total,
        Pages:    (total + limit - 1) / limit,
        Current:  page,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}