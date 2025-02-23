package utils

import (
    "math"
    "strconv"
    "strings"
)

func ParseDistance(distance string) float64 {
    distance = strings.TrimSpace(strings.ToUpper(distance))
    distance = strings.TrimSuffix(distance, "KM")
    distance = strings.TrimSpace(distance)

    val, err := strconv.ParseFloat(distance, 64)
    if err != nil {
        return 0
    }
    return val
}

func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
    const earthRadius = 6371.0 // Earth's radius in kilometers

    // Convert coordinates to radians
    lat1Rad := lat1 * math.Pi / 180
    lon1Rad := lon1 * math.Pi / 180
    lat2Rad := lat2 * math.Pi / 180
    lon2Rad := lon2 * math.Pi / 180

    // Calculate differences
    dLat := lat2Rad - lat1Rad
    dLon := lon2Rad - lon1Rad

    // Haversine formula
    a := math.Sin(dLat/2)*math.Sin(dLat/2) +
        math.Cos(lat1Rad)*math.Cos(lat2Rad)*
            math.Sin(dLon/2)*math.Sin(dLon/2)

    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
    distance := earthRadius * c

    return distance
}