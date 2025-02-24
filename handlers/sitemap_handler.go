package handlers

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
	"village_site/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	countCache = make(map[string]struct {
		count     int
		expiresAt time.Time
	})
	countMutex sync.RWMutex

	// Replace the existing sitemapCache with sync.Map
	sitemapCache sync.Map
)

type sitemapCacheEntry struct {
	content   []byte
	expiresAt time.Time
}

// getCachedCount gets a count from cache or calls the provider function
func getCachedCount(key string, provider func() int) int {
	countMutex.RLock()
	if cached, ok := countCache[key]; ok && time.Now().Before(cached.expiresAt) {
		countMutex.RUnlock()
		return cached.count
	}
	countMutex.RUnlock()

	// Cache miss or expired, get new count
	count := provider()
	
	countMutex.Lock()
	countCache[key] = struct {
		count     int
		expiresAt time.Time
	}{
		count:     count,
		expiresAt: time.Now().Add(cacheDuration),
	}
	countMutex.Unlock()

	return count
}

// getSitemapCacheKey generates a unique cache key for a sitemap
func getSitemapCacheKey(section string, page int) string {
	return fmt.Sprintf("%s_page_%d", section, page)
}

// getCachedSitemap retrieves a cached sitemap if it exists and is fresh
func getCachedSitemap(section string, page int) ([]byte, bool) {
	key := getSitemapCacheKey(section, page)
	
	if value, ok := sitemapCache.Load(key); ok {
		entry := value.(sitemapCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.content, true
		}
		// Remove expired entry
		sitemapCache.Delete(key)
	}
	
	return nil, false
}

// cacheSitemap stores a generated sitemap in the cache
func cacheSitemap(section string, page int, content []byte) {
	key := getSitemapCacheKey(section, page)
	
	entry := sitemapCacheEntry{
		content:   content,
		expiresAt: time.Now().Add(sitemapCacheDuration),
	}
	
	sitemapCache.Store(key, entry)
}

// sitemapExists checks if a sitemap for the given section and page already exists and is still fresh
func sitemapExists(section string, page int) bool {
	cacheKey := getSitemapCacheKey(section, page)
	_, exists := sitemapCache.Load(cacheKey)
	return exists
}

// GetSitemapIndex handles the main sitemap index request with pagination info
func GetSitemapIndex(w http.ResponseWriter, r *http.Request) {
	sections := []struct {
		name     string
		getCount func() int
		priority float64
	}{
		{name: "bus-routes", getCount: getBusRoutesCount, priority: 0.8},
		{name: "train-routes", getCount: getTrainRoutesCount, priority: 0.8},
		{name: "distances", getCount: getDistancesCount, priority: 0.7},
		{name: "banks", getCount: getBanksCount, priority: 0.7},
		{name: "companies", getCount: getCompaniesCount, priority: 0.7},
		{name: "villages", getCount: getVillagesCount, priority: 0.8},
		{name: "mandals", getCount: getMandalsCount, priority: 0.8},
		{name: "pincodes", getCount: getPincodesCount, priority: 0.7},
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(cacheDuration.Seconds())))

	// Create response structure
	index := SitemapIndex{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	now := time.Now().Format(time.RFC3339)

	// Process each section
	for _, section := range sections {
		count := getCachedCount(section.name, section.getCount)
		pageCount := (count + maxURLsPerSitemap - 1) / maxURLsPerSitemap

		for page := 1; page <= pageCount; page++ {
			// Check if sitemap exists and is fresh
			exists := sitemapExists(section.name, page)
			
			// Add sitemap entry
			sitemap := Sitemap{
				Loc:     fmt.Sprintf("%s/api/v1/sitemaps/%s?page=%d", baseURL, section.name, page),
				LastMod: now,
			}
			
			// Add cache status as a comment in the XML
			if exists {
				sitemap.Loc += "<!-- cached -->"
			}
			
			index.Sitemaps = append(index.Sitemaps, sitemap)
		}
	}

	// Marshal and write response
	output, err := xml.MarshalIndent(index, "", "  ")
	if err != nil {
		http.Error(w, "Error generating sitemap index", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%s%s", xml.Header, output)
}

// GetBusRoutesSitemap generates sitemap for bus routes
func GetBusRoutesSitemap(w http.ResponseWriter, r *http.Request) {
	page, limit := getPaginationParams(r)
	
	// Check cache first
	if cached, ok := getCachedSitemap("bus-routes", page); ok {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("X-Robots-Tag", "noindex")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Cache", "HIT")
		w.Write(cached)
		return
	}

	offset := (page - 1) * limit

	collection := config.MongoDB.Collection("bus_routes")
	ctx := r.Context()

	// First, get all routes grouped by city with their stops
	pipeline := []bson.M{
		{
			"$project": bson.M{
				"city":          1,
				"route_name":    1,
				"stops":         1,
			},
		},
		{
			"$sort": bson.M{
				"city": 1,
				"route_name": 1,
			},
		},
		{
			"$skip": offset,
		},
		{
			"$limit": limit,
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	urlSet := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	now := time.Now().Format("2006-01-02")
	for cursor.Next(ctx) {
		var route struct {
			City      string   `bson:"city"`
			RouteName string   `bson:"route_name"`
			Stops     []string `bson:"stops"`
		}
		if err := cursor.Decode(&route); err != nil {
			continue
		}

		// Main route page
		urlSet.URLs = append(urlSet.URLs, URL{
			Loc:        fmt.Sprintf("%s/bus/%s/%s", baseURL, url.PathEscape(route.City), url.PathEscape(route.RouteName)),
			LastMod:    now,
			ChangeFreq: "weekly",
			Priority:   0.8,
		})

		// Generate URLs for all possible stop combinations
		for i := 0; i < len(route.Stops); i++ {
			fromStop := route.Stops[i]
			if fromStop == "" {
				continue
			}

			// Generate URLs to all subsequent stops
			for j := i + 1; j < len(route.Stops); j++ {
				toStop := route.Stops[j]
				if toStop == "" || toStop == fromStop {
					continue
				}

				urlSet.URLs = append(urlSet.URLs, URL{
					Loc: fmt.Sprintf("%s/bus/%s/%s/from/%s/to/%s",
						baseURL,
						url.PathEscape(route.City),
						url.PathEscape(route.RouteName),
						url.PathEscape(fromStop),
						url.PathEscape(toStop)),
					LastMod:    now,
					ChangeFreq: "weekly",
					Priority:   0.7,
				})

				// Also add reverse direction
				urlSet.URLs = append(urlSet.URLs, URL{
					Loc: fmt.Sprintf("%s/bus/%s/%s/from/%s/to/%s",
						baseURL,
						url.PathEscape(route.City),
						url.PathEscape(route.RouteName),
						url.PathEscape(toStop),
						url.PathEscape(fromStop)),
					LastMod:    now,
					ChangeFreq: "weekly",
					Priority:   0.7,
				})
			}

			// Add city-level stop combinations
			urlSet.URLs = append(urlSet.URLs, URL{
				Loc: fmt.Sprintf("%s/bus/%s/from/%s",
					baseURL,
					url.PathEscape(route.City),
					url.PathEscape(fromStop)),
				LastMod:    now,
				ChangeFreq: "weekly",
				Priority:   0.6,
			})
		}
	}

	// Update the writeXMLResponse call to include section and page
	writeXMLResponse(w, urlSet.URLs, "bus-routes", page)
}

// GetTrainRoutesSitemap generates sitemap for train routes
func GetTrainRoutesSitemap(w http.ResponseWriter, r *http.Request) {
	page, limit := getPaginationParams(r)
	
	// Check cache first
	if cached, ok := getCachedSitemap("train-routes", page); ok {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("X-Robots-Tag", "noindex")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Cache", "HIT")
		w.Write(cached)
		return
	}

	offset := (page - 1) * limit

	collection := config.MongoDB.Collection("trains")
	ctx := r.Context()

	// First get total count
	total, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		http.Error(w, "Error counting trains: " + err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate max pages
	maxPages := (int(total) + limit - 1) / limit
	if page > maxPages {
		http.Error(w, "Page number exceeds maximum pages", http.StatusBadRequest)
		return
	}

	// Optimized aggregation pipeline
	pipeline := []bson.M{
		{
			"$skip": offset,
		},
		{
			"$limit": limit,
		},
		{
			"$project": bson.M{
				"train_number": 1,
				"schedule_table": 1,
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		http.Error(w, "Error fetching trains: " + err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	urlSet := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	now := time.Now().Format("2006-01-02")
	
	// Process each train
	for cursor.Next(ctx) {
		var train struct {
			TrainNumber   int    `bson:"train_number"`
			ScheduleTable []struct {
				Station string `bson:"station"`
			} `bson:"schedule_table"`
		}
		
		if err := cursor.Decode(&train); err != nil {
			continue
		}

		// Main train page
		urlSet.URLs = append(urlSet.URLs, URL{
			Loc:        fmt.Sprintf("%s/train/%d", baseURL, train.TrainNumber),
			LastMod:    now,
			ChangeFreq: "weekly",
			Priority:   0.8,
		})

		// Process each station in schedule
		processedStations := make(map[string]bool)
		for _, stop := range train.ScheduleTable {
			station := cleanStationCode(stop.Station)
			if station == "" || processedStations[station] {
				continue
			}
			processedStations[station] = true

			// Train-station specific page
			urlSet.URLs = append(urlSet.URLs, URL{
				Loc:        fmt.Sprintf("%s/train/%d/station/%s", baseURL, train.TrainNumber, url.PathEscape(station)),
				LastMod:    now,
				ChangeFreq: "weekly",
				Priority:   0.7,
			})

			// Station info page
			urlSet.URLs = append(urlSet.URLs, URL{
				Loc:        fmt.Sprintf("%s/station/%s", baseURL, url.PathEscape(station)),
				LastMod:    now,
				ChangeFreq: "weekly",
				Priority:   0.7,
			})
		}
	}

	if err = cursor.Err(); err != nil {
		http.Error(w, "Error processing trains: " + err.Error(), http.StatusInternalServerError)
		return
	}

	// Set pagination headers
	w.Header().Set("X-Total-Pages", strconv.Itoa(maxPages))
	w.Header().Set("X-Current-Page", strconv.Itoa(page))

	// Write response with caching
	writeXMLResponse(w, urlSet.URLs, "train-routes", page)
}

// Add helper function for pagination
func getPaginationParams(r *http.Request) (page, limit int) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page = 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	limit = maxURLsPerSitemap
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= maxURLsPerSitemap {
		limit = l
	}

	return page, limit
}

// writeXMLResponse modified to support caching and compression
func writeXMLResponse(w http.ResponseWriter, urls []URL, section string, page int) {
	// Check if sitemap exists in cache
	if content, exists := getCachedSitemap(section, page); exists {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("X-Cache", "HIT")
		w.Write(content)
		return
	}

	// Generate new sitemap
	urlSet := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	output, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		http.Error(w, "Error generating sitemap: " + err.Error(), http.StatusInternalServerError)
		return
	}

	// Add XML header
	content := []byte(xml.Header + string(output))

	// Compress the content
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(content); err != nil {
		http.Error(w, "Error compressing sitemap: " + err.Error(), http.StatusInternalServerError)
		return
	}
	if err := gz.Close(); err != nil {
		http.Error(w, "Error finalizing compression: " + err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache the compressed sitemap
	cacheSitemap(section, page, buf.Bytes())

	// Set response headers
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("X-Cache", "MISS")
	w.Write(buf.Bytes())
}

// GetBanksSitemap generates sitemap for banks
func GetBanksSitemap(w http.ResponseWriter, r *http.Request) {
	page, limit := getPaginationParams(r)
	offset := (page - 1) * limit

	// Get total count first
	var total int
	err := config.DB.QueryRow(`
		SELECT COUNT(DISTINCT CONCAT(bank, '/', state, '/', district, '/', branch_city, '/', ifsc)) 
		FROM ifsc_details
	`).Scan(&total)
	if err != nil {
		http.Error(w, "Error counting banks", http.StatusInternalServerError)
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
		WITH RECURSIVE bank_hierarchy AS (
			SELECT DISTINCT 
				bank,
				state,
				district,
				branch_city,
				ifsc,
				ROW_NUMBER() OVER (
					ORDER BY bank, state, district, branch_city, ifsc
				) as rn
			FROM ifsc_details
		)
		SELECT 
			bank,
			state,
			district,
			branch_city,
			ifsc
		FROM bank_hierarchy
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
		var bank, state, district, city, ifsc string
		if err := rows.Scan(&bank, &state, &district, &city, &ifsc); err != nil {
			continue
		}

		// Bank branch page
		urlSet.URLs = append(urlSet.URLs, URL{
			Loc: fmt.Sprintf("%s/bank/%s/%s/%s/%s/%s",
				baseURL,
				url.PathEscape(bank),
				url.PathEscape(state),
				url.PathEscape(district),
				url.PathEscape(city),
				url.PathEscape(ifsc)),
			LastMod:    now,
			ChangeFreq: "monthly",
			Priority:   0.6,
		})

		// Bank state page
		urlSet.URLs = append(urlSet.URLs, URL{
			Loc: fmt.Sprintf("%s/bank/%s/%s",
				baseURL,
				url.PathEscape(bank),
				url.PathEscape(state)),
			LastMod:    now,
			ChangeFreq: "monthly",
			Priority:   0.7,
		})

		// Bank district page
		urlSet.URLs = append(urlSet.URLs, URL{
			Loc: fmt.Sprintf("%s/bank/%s/%s/%s",
				baseURL,
				url.PathEscape(bank),
				url.PathEscape(state),
				url.PathEscape(district)),
			LastMod:    now,
			ChangeFreq: "monthly",
			Priority:   0.7,
		})
	}

	w.Header().Set("X-Total-Pages", strconv.Itoa(maxPages))
	w.Header().Set("X-Current-Page", strconv.Itoa(page))
	writeXMLResponse(w, urlSet.URLs, "banks", page)
}

// GetCompaniesSitemap generates sitemap for company directory
func GetCompaniesSitemap(w http.ResponseWriter, r *http.Request) {
	page, limit := getPaginationParams(r)
	offset := (page - 1) * limit

	// Get total count first
	collection := config.MongoDB.Collection("companies")
	ctx := r.Context()
	
	total, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		http.Error(w, "Error counting companies", http.StatusInternalServerError)
		return
	}

	// Use total to calculate max pages
	maxPages := (int(total) / limit) + 1
	if page > maxPages {
		page = maxPages
		offset = (page - 1) * limit
	}

	// Get paginated results
	cursor, err := collection.Find(ctx, bson.M{},
		options.Find().
			SetSkip(int64(offset)).
			SetLimit(int64(limit)))
	if err != nil {
		http.Error(w, "Error fetching companies", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	urlSet := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	for cursor.Next(ctx) {
		var company Company
		if err := cursor.Decode(&company); err != nil {
			continue
		}

		urlSet.URLs = append(urlSet.URLs, URL{
			Loc: fmt.Sprintf("%s/company/%s", baseURL, url.PathEscape(company.URLTitle)),
			ChangeFreq: "monthly",
			Priority:   0.7,
			LastMod:   time.Now().Format("2006-01-02"),
		})
	}

	writeXMLResponse(w, urlSet.URLs, "companies", page)
}

// GetVillagesSitemap generates sitemap for villages with optimized streaming
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

// GetMandalsSitemap generates sitemap for mandals
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

// GetPincodesSitemap generates sitemap for pincodes
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

// GetDistancesSitemap generates sitemap for distance calculator pages
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

// Optimize count queries
func getBusRoutesCount() int {
	return getCachedCount("bus_routes", func() int {
		count, _ := config.MongoDB.Collection("bus_routes").CountDocuments(context.Background(), bson.M{})
		return int(count)
	})
}

func getTrainRoutesCount() int {
	return getCachedCount("trains", func() int {
		count, _ := config.MongoDB.Collection("trains").CountDocuments(context.Background(), bson.M{})
		return int(count)
	})
}

func getBanksCount() int {
	return getCachedCount("banks", func() int {
		var count int
		_ = config.DB.QueryRow("SELECT COUNT(*) FROM ifsc_details").Scan(&count)
		return count
	})
}

func getCompaniesCount() int {
	return getCachedCount("companies", func() int {
		count, _ := config.MongoDB.Collection("companies").CountDocuments(context.Background(), bson.M{})
		return int(count)
	})
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

// cleanName removes special characters and trims spaces from a string
func cleanName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return s
}

func cleanStationCode(station string) string {
	station = strings.TrimSpace(station)
	station = strings.ToUpper(station)
	return station
}