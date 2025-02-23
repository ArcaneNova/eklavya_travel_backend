// Add to your config/cache.go
package config

import (
    "github.com/patrickmn/go-cache"
    "time"
)

var Cache *cache.Cache

func InitCache() {
    // Create a cache with 5 minutes default expiration and 10 minutes cleanup interval
    Cache = cache.New(5*time.Minute, 10*time.Minute)
}