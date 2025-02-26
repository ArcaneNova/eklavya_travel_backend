package handlers

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
	"village_site/config"
	"compress/gzip"
	"bytes"
	"strings"
)

type URL struct {
	XMLName    xml.Name `xml:"url"`
	Loc        string   `xml:"loc"`
	LastMod    string   `xml:"lastmod,omitempty"`
	ChangeFreq string   `xml:"changefreq,omitempty"`
	Priority   float64  `xml:"priority,omitempty"`
}

type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	XMLNS   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

type SitemapIndex struct {
	XMLName  xml.Name  `xml:"sitemapindex"`
	XMLNS    string    `xml:"xmlns,attr"`
	Sitemaps []Sitemap `xml:"sitemap"`
}

type Sitemap struct {
	XMLName xml.Name `xml:"sitemap"`
	Loc     string   `xml:"loc"`
	LastMod string   `xml:"lastmod,omitempty"`
}

type SitemapURL struct {
	URL string `json:"url"`
}

const (
	maxURLsPerSitemap = 10000
	baseURL = "https://eklavyatravel.com"
	XMLHeader = `<?xml version="1.0" encoding="UTF-8"?>`
	sitemapCacheDuration = 24 * time.Hour // Cache sitemaps for 24 hours
)

var (
	sitemapCache = make(map[string]sitemapCacheEntry)
	cacheMutex   sync.RWMutex
)

type sitemapCacheEntry struct {
	content   []byte
	expiresAt time.Time
}

func getCachedCount(key string, provider func() int) int {
	cacheMutex.RLock()
	entry, exists := sitemapCache[key]
	cacheMutex.RUnlock()

	if exists && time.Now().Before(entry.expiresAt) {
		return provider()
	}

	count := provider()

	cacheMutex.Lock()
	sitemapCache[key] = sitemapCacheEntry{
		content:   nil,
		expiresAt: time.Now().Add(24 * time.Hour),
	}
	cacheMutex.Unlock()

	return count
}

func getSitemapCacheKey(section string, page int) string {
	return fmt.Sprintf("%s_page_%d", section, page)
}

func getCachedSitemap(section string, page int) ([]byte, bool) {
	key := getSitemapCacheKey(section, page)
	
	cacheMutex.RLock()
	entry, exists := sitemapCache[key]
	cacheMutex.RUnlock()

	if exists && time.Now().Before(entry.expiresAt) {
		return entry.content, true
	}
	return nil, false
}

func cacheSitemap(section string, page int, content []byte) {
	key := getSitemapCacheKey(section, page)
	
	cacheMutex.Lock()
	sitemapCache[key] = sitemapCacheEntry{
		content:   content,
		expiresAt: time.Now().Add(24 * time.Hour),
	}
	cacheMutex.Unlock()
}

func sitemapExists(section string, page int) bool {
	_, exists := getCachedSitemap(section, page)
	return exists
}

func GetSitemapIndex(w http.ResponseWriter, r *http.Request) {
	baseURL := "https://eklavyatravel.com/api/v1/sitemaps"
	lastmod := time.Now().Format("2006-01-02")

	sitemaps := []Sitemap{
		{Loc: fmt.Sprintf("%s/villages", baseURL), LastMod: lastmod},
		{Loc: fmt.Sprintf("%s/mandals", baseURL), LastMod: lastmod},
		{Loc: fmt.Sprintf("%s/pincodes", baseURL), LastMod: lastmod},
		{Loc: fmt.Sprintf("%s/distances", baseURL), LastMod: lastmod},
	}

	index := SitemapIndex{
		XMLNS:    "http://www.sitemaps.org/schemas/sitemap/0.9",
		Sitemaps: sitemaps,
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(index)
}

func getPaginationParams(r *http.Request) (page, limit int) {
	page = 1
	limit = 50000 // Maximum URLs per sitemap as per protocol

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	return page, limit
}

func writeXMLResponse(w http.ResponseWriter, urls []URL, section string, page int) {
	urlset := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	output, err := xml.MarshalIndent(urlset, "", "  ")
	if err != nil {
		http.Error(w, "Error generating sitemap", http.StatusInternalServerError)
		return
	}

	// Cache the output
	cacheSitemap(section, page, output)

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml.Header))
	w.Write(output)
}

func GetVillagesSitemap(w http.ResponseWriter, r *http.Request) {
	page, limit := getPaginationParams(r)
	offset := (page - 1) * limit

	// Get total count first
	var total int
	err := config.DB.QueryRow("SELECT COUNT(*) FROM villages").Scan(&total)
	if err != nil {
		http.Error(w, "Error counting villages", http.StatusInternalServerError)
		return
	}

	// Calculate max pages
	maxPages := (total + limit - 1) / limit
	if page > maxPages {
		http.Error(w, "Page number exceeds maximum pages", http.StatusBadRequest)
		return
	}

	// Set up streaming response
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Total-Pages", strconv.Itoa(maxPages))
	w.Header().Set("X-Current-Page", strconv.Itoa(page))

	gz := gzip.NewWriter(w)
	defer gz.Close()

	// Start XML document
	fmt.Fprintf(gz, xml.Header)
	fmt.Fprintf(gz, "<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")

	// Stream results in batches
	rows, err := config.DB.Query(`
		SELECT state, district, subdistrict, locality 
		FROM villages 
		ORDER BY state, district, subdistrict, locality
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		http.Error(w, "Error generating sitemap", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Use buffer for better performance
	var buf bytes.Buffer
	now := time.Now().Format("2006-01-02")

	for rows.Next() {
		var state, district, subdistrict, locality string
		if err := rows.Scan(&state, &district, &subdistrict, &locality); err != nil {
			continue
		}

		buf.Reset()
		url := fmt.Sprintf("%s/village/%s/%s/%s/%s",
			baseURL,
			url.PathEscape(state),
			url.PathEscape(district),
			url.PathEscape(subdistrict),
			url.PathEscape(locality))

		fmt.Fprintf(&buf, "  <url>\n    <loc>%s</loc>\n    <lastmod>%s</lastmod>\n    <changefreq>monthly</changefreq>\n    <priority>0.7</priority>\n  </url>\n",
			url, now)

		if _, err := gz.Write(buf.Bytes()); err != nil {
			http.Error(w, "Error writing response", http.StatusInternalServerError)
			return
		}
	}

	// Close XML document
	fmt.Fprintf(gz, "</urlset>")
}

func GetMandalsSitemap(w http.ResponseWriter, r *http.Request) {
	page, limit := getPaginationParams(r)
	offset := (page - 1) * limit

	// Get total count first
	var total int
	err := config.DB.QueryRow(`
		SELECT COUNT(DISTINCT district || '/' || subdistrict) 
		FROM mandals
	`).Scan(&total)
	if err != nil {
		http.Error(w, "Error counting mandals", http.StatusInternalServerError)
		return
	}

	// Get paginated results
	rows, err := config.DB.Query(`
		SELECT DISTINCT district, subdistrict 
		FROM mandals 
		ORDER BY district, subdistrict
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		http.Error(w, "Error generating sitemap", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	urlSet := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	for rows.Next() {
		var district, subdistrict string
		if err := rows.Scan(&district, &subdistrict); err != nil {
			continue
		}

		urlSet.URLs = append(urlSet.URLs, URL{
			Loc: fmt.Sprintf("%s/mandal/%s/%s",
				baseURL,
				url.PathEscape(district),
				url.PathEscape(subdistrict)),
			ChangeFreq: "monthly",
			Priority:   0.6,
			LastMod:   time.Now().Format("2006-01-02"),
		})
	}

	writeXMLResponse(w, urlSet.URLs, "mandals", page)
}

func GetPincodesSitemap(w http.ResponseWriter, r *http.Request) {
	page, limit := getPaginationParams(r)
	offset := (page - 1) * limit

	// Get total count first
	var total int
	err := config.DB.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT DISTINCT pincode, officename, district, state 
			FROM pin_details
		) t
	`).Scan(&total)
	if err != nil {
		http.Error(w, "Error counting pincodes", http.StatusInternalServerError)
		return
	}

	// Calculate max pages
	maxPages := (total + limit - 1) / limit
	if page > maxPages {
		http.Error(w, "Invalid page number", http.StatusBadRequest)
		return
	}

	// Get paginated results with optimized query
	rows, err := config.DB.Query(`
		WITH RECURSIVE pin_hierarchy AS (
			SELECT DISTINCT 
				pincode,
				officename,
				district,
				state,
				ROW_NUMBER() OVER (
					ORDER BY state, district, pincode, officename
				) as rn
			FROM pin_details
		)
		SELECT 
			pincode,
			officename,
			district,
			state
		FROM pin_hierarchy
		WHERE rn > $1 AND rn <= $2
	`, offset, offset+limit)
	if err != nil {
		http.Error(w, "Error generating sitemap", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	urlSet := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	now := time.Now().Format("2006-01-02")
	for rows.Next() {
		var pincode, officeName, district, state string
		if err := rows.Scan(&pincode, &officeName, &district, &state); err != nil {
			continue
		}

		// Pincode page
		urlSet.URLs = append(urlSet.URLs, URL{
			Loc: fmt.Sprintf("%s/pincode/%s",
				baseURL,
				url.PathEscape(pincode)),
			LastMod:    now,
			ChangeFreq: "monthly",
			Priority:   0.7,
		})

		// Post office page
		urlSet.URLs = append(urlSet.URLs, URL{
			Loc: fmt.Sprintf("%s/post-office/%s/%s",
				baseURL,
				url.PathEscape(state),
				url.PathEscape(officeName)),
			LastMod:    now,
			ChangeFreq: "monthly",
			Priority:   0.6,
		})

		// District post offices page
		urlSet.URLs = append(urlSet.URLs, URL{
			Loc: fmt.Sprintf("%s/post-offices/%s/%s",
				baseURL,
				url.PathEscape(state),
				url.PathEscape(district)),
			LastMod:    now,
			ChangeFreq: "monthly",
			Priority:   0.6,
		})

		// State post offices page
		urlSet.URLs = append(urlSet.URLs, URL{
			Loc: fmt.Sprintf("%s/post-offices/%s",
				baseURL,
				url.PathEscape(state)),
			LastMod:    now,
			ChangeFreq: "monthly",
			Priority:   0.7,
		})
	}

	w.Header().Set("X-Total-Pages", strconv.Itoa(maxPages))
	w.Header().Set("X-Current-Page", strconv.Itoa(page))
	writeXMLResponse(w, urlSet.URLs, "pincodes", page)
}

func GetDistancesSitemap(w http.ResponseWriter, r *http.Request) {
	page, limit := getPaginationParams(r)
	
	// Check cache first
	if cached, ok := getCachedSitemap("distances", page); ok {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("X-Robots-Tag", "noindex")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Cache", "HIT")
		w.Write(cached)
		return
	}

	// First get total count to validate pagination
	var total int
	err := config.DB.QueryRow(`
		SELECT COUNT(DISTINCT subdistrict) 
		FROM villages 
		WHERE subdistrict != ''`).Scan(&total)
	if err != nil {
		http.Error(w, "Error counting locations: " + err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate max pages
	maxPages := (total + limit - 1) / limit
	if page > maxPages {
		http.Error(w, "Page number exceeds maximum pages", http.StatusBadRequest)
		return
	}

	// Query to get combinations of locations for distance pages
	query := `
		WITH locations AS (
			SELECT DISTINCT state, district, subdistrict
			FROM villages
			WHERE subdistrict != ''
			ORDER BY state, district, subdistrict
			OFFSET $1 LIMIT $2
		)
		SELECT 
			l1.state as from_state,
			l1.district as from_district,
			l1.subdistrict as from_subdistrict,
			l2.state as to_state,
			l2.district as to_district,
			l2.subdistrict as to_subdistrict
		FROM locations l1
		CROSS JOIN locations l2
		WHERE l1.state = l2.state
		AND (l1.district < l2.district 
			OR (l1.district = l2.district 
				AND l1.subdistrict < l2.subdistrict))
		ORDER BY 
			l1.state, l1.district, l1.subdistrict,
			l2.state, l2.district, l2.subdistrict`

	rows, err := config.DB.Query(query, (page-1)*limit, limit)
	if err != nil {
		http.Error(w, "Error fetching distances: " + err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	urlSet := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	now := time.Now().Format("2006-01-02")
	processedDistricts := make(map[string]bool)

	for rows.Next() {
		var fromState, fromDistrict, fromSubdistrict string
		var toState, toDistrict, toSubdistrict string

		err := rows.Scan(&fromState, &fromDistrict, &fromSubdistrict,
			&toState, &toDistrict, &toSubdistrict)
		if err != nil {
			continue
		}

		// Skip empty values
		if fromSubdistrict == "" || toSubdistrict == "" {
			continue
		}

		// Clean and encode location names
		fromState = url.PathEscape(cleanName(fromState))
		fromDistrict = url.PathEscape(cleanName(fromDistrict))
		fromSubdistrict = url.PathEscape(cleanName(fromSubdistrict))
		toState = url.PathEscape(cleanName(toState))
		toDistrict = url.PathEscape(cleanName(toDistrict))
		toSubdistrict = url.PathEscape(cleanName(toSubdistrict))

		// Main distance page
		urlSet.URLs = append(urlSet.URLs, URL{
			Loc: fmt.Sprintf("%s/distance-between/%s/%s/%s/%s",
				baseURL, fromDistrict, fromSubdistrict, toDistrict, toSubdistrict),
			LastMod:    now,
			ChangeFreq: "weekly",
			Priority:   0.7,
		})

		// Add district-level pages if not already processed
		districtKey := fromDistrict + "-" + toDistrict
		if !processedDistricts[districtKey] {
			processedDistricts[districtKey] = true
			urlSet.URLs = append(urlSet.URLs, URL{
				Loc: fmt.Sprintf("%s/distance-between/%s/to/%s",
					baseURL, fromDistrict, toDistrict),
				LastMod:    now,
				ChangeFreq: "weekly",
				Priority:   0.6,
			})
		}
	}

	// Set pagination headers
	w.Header().Set("X-Total-Pages", strconv.Itoa(maxPages))
	w.Header().Set("X-Current-Page", strconv.Itoa(page))

	// Write response with caching
	writeXMLResponse(w, urlSet.URLs, "distances", page)
}

func getVillagesCount() int {
	return getCachedCount("villages", func() int {
		var count int
		_ = config.DB.QueryRow("SELECT COUNT(*) FROM villages").Scan(&count)
		return count
	})
}

func getMandalsCount() int {
	return getCachedCount("mandals", func() int {
		var count int
		_ = config.DB.QueryRow("SELECT COUNT(DISTINCT district || '/' || subdistrict) FROM mandals").Scan(&count)
		return count
	})
}

func getPincodesCount() int {
	return getCachedCount("pincodes", func() int {
		var count int
		_ = config.DB.QueryRow("SELECT COUNT(DISTINCT pincode) FROM pin_details").Scan(&count)
		return count
	})
}

func getDistancesCount() int {
	return getCachedCount("distances", func() int {
		var count int
		_ = config.DB.QueryRow(`
			SELECT COUNT(DISTINCT subdistrict) 
			FROM villages 
			WHERE subdistrict != ''
		`).Scan(&count)
		return count
	})
}

func cleanName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return s
}