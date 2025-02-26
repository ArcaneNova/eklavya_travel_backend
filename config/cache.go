// Add to your config/cache.go
package config

import (
    "github.com/patrickmn/go-cache"
    "time"
    "fmt"
)

var (
    // Cache instances for different data types
    VillageCache *cache.Cache
    BankCache    *cache.Cache
    PinCodeCache *cache.Cache
)

const (
    // Cache durations
    villageCacheDuration = 24 * time.Hour
    bankCacheDuration    = 12 * time.Hour
    pinCodeCacheDuration = 24 * time.Hour
    
    // Cleanup intervals
    villageCleanupInterval = 48 * time.Hour
    bankCleanupInterval    = 24 * time.Hour
    pinCodeCleanupInterval = 48 * time.Hour
)

func InitCache() {
    // Initialize separate caches for different data types
    VillageCache = cache.New(villageCacheDuration, villageCleanupInterval)
    BankCache = cache.New(bankCacheDuration, bankCleanupInterval)
    PinCodeCache = cache.New(pinCodeCacheDuration, pinCodeCleanupInterval)
}

func ClearAllCaches() {
    VillageCache.Flush()
    BankCache.Flush()
    PinCodeCache.Flush()
}

func GetCacheKey(prefix string, params ...interface{}) string {
    key := prefix
    for _, param := range params {
        key += ":" + fmt.Sprintf("%v", param)
    }
    return key
}