package handlers

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sync"
	"time"
	"village_site/config"
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

const baseURL = "https://villagedirectory.in"

var (
	countCache = make(map[string]struct {
		count     int
		expiresAt time.Time
	})
	countMutex sync.RWMutex
)

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

// GetSitemapIndex handles the main sitemap index request
func GetSitemapIndex(w http.ResponseWriter, r *http.Request) {
	sitemapIndex := SitemapIndex{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		Sitemaps: []Sitemap{
			{
				Loc:     fmt.Sprintf("%s/api/v1/sitemaps/villages", baseURL),
				LastMod: time.Now().Format("2006-01-02"),
			},
			{
				Loc:     fmt.Sprintf("%s/api/v1/sitemaps/banks", baseURL),
				LastMod: time.Now().Format("2006-01-02"),
			},
			{
				Loc:     fmt.Sprintf("%s/api/v1/sitemaps/pincodes", baseURL),
				LastMod: time.Now().Format("2006-01-02"),
			},
		},
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(sitemapIndex)
}

// GetVillagesSitemap generates sitemap for villages
func GetVillagesSitemap(w http.ResponseWriter, r *http.Request) {
	urls := []URL{}

	// Query for states
	rows, err := config.DB.Query(`
		SELECT DISTINCT state_name 
		FROM villages 
		ORDER BY state_name
	`)
	if err != nil {
		http.Error(w, "Error fetching states", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var state string
		if err := rows.Scan(&state); err != nil {
			continue
		}
		urls = append(urls, URL{
			Loc:     fmt.Sprintf("%s/village/%s", baseURL, state),
			LastMod: time.Now().Format("2006-01-02"),
		})
	}

	// Query for districts
	rows, err = config.DB.Query(`
		SELECT DISTINCT state_name, district_name 
		FROM villages 
		ORDER BY state_name, district_name
	`)
	if err != nil {
		http.Error(w, "Error fetching districts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var state, district string
		if err := rows.Scan(&state, &district); err != nil {
			continue
		}
		urls = append(urls, URL{
			Loc:     fmt.Sprintf("%s/village/%s/%s", baseURL, state, district),
			LastMod: time.Now().Format("2006-01-02"),
		})
	}

	urlset := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(urlset)
}

// GetBanksSitemap generates sitemap for banks
func GetBanksSitemap(w http.ResponseWriter, r *http.Request) {
	urls := []URL{}

	// Query for banks
	rows, err := config.DB.Query(`
		SELECT DISTINCT bank_name 
		FROM ifsc_details 
		ORDER BY bank_name
	`)
	if err != nil {
		http.Error(w, "Error fetching banks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var bank string
		if err := rows.Scan(&bank); err != nil {
			continue
		}
		urls = append(urls, URL{
			Loc:     fmt.Sprintf("%s/bank/%s", baseURL, bank),
			LastMod: time.Now().Format("2006-01-02"),
		})
	}

	// Query for IFSC codes
	rows, err = config.DB.Query(`
		SELECT DISTINCT ifsc 
		FROM ifsc_details 
		ORDER BY ifsc
	`)
	if err != nil {
		http.Error(w, "Error fetching IFSC codes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var ifsc string
		if err := rows.Scan(&ifsc); err != nil {
			continue
		}
		urls = append(urls, URL{
			Loc:     fmt.Sprintf("%s/bank/ifsc/%s", baseURL, ifsc),
			LastMod: time.Now().Format("2006-01-02"),
		})
	}

	urlset := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(urlset)
}

// GetPincodesSitemap generates sitemap for pincodes
func GetPincodesSitemap(w http.ResponseWriter, r *http.Request) {
	urls := []URL{}

	// Query for PIN codes
	rows, err := config.DB.Query(`
		SELECT DISTINCT pincode 
		FROM pin_details 
		ORDER BY pincode
	`)
	if err != nil {
		http.Error(w, "Error fetching PIN codes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var pincode string
		if err := rows.Scan(&pincode); err != nil {
			continue
		}
		urls = append(urls, URL{
			Loc:     fmt.Sprintf("%s/pincode/%s", baseURL, pincode),
			LastMod: time.Now().Format("2006-01-02"),
		})
	}

	// Query for post offices
	rows, err = config.DB.Query(`
		SELECT DISTINCT office_name, pincode 
		FROM pin_details 
		ORDER BY office_name
	`)
	if err != nil {
		http.Error(w, "Error fetching post offices", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var officeName, pincode string
		if err := rows.Scan(&officeName, &pincode); err != nil {
			continue
		}
		urls = append(urls, URL{
			Loc:     fmt.Sprintf("%s/pincode/%s/post-office/%s", baseURL, pincode, officeName),
			LastMod: time.Now().Format("2006-01-02"),
		})
	}

	urlset := URLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(urlset)
}

// Count functions for caching
func getVillagesCount() int {
	return getCachedCount("villages", func() int {
		var count int
		_ = config.DB.QueryRow("SELECT COUNT(*) FROM villages").Scan(&count)
		return count
	})
}

func getBanksCount() int {
	return getCachedCount("banks", func() int {
		var count int
		_ = config.DB.QueryRow("SELECT COUNT(*) FROM ifsc_details").Scan(&count)
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