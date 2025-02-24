package handlers

import (
	"math"
)

type RouteMetrics struct {
	TotalDistance float64
	StopCount    int
	Duration     int
	Comfort      float64
}

func calculateRouteMetrics(route BusRoute, fromIdx, toIdx int) RouteMetrics {
	// Calculate actual segment distance based on the stops being used
	segmentDistance := 0.0
	if fromIdx < toIdx && fromIdx >= 0 && toIdx < len(route.Route) {
		// Calculate direct distance between stops
		for i := fromIdx; i < toIdx; i++ {
			lat1, lng1 := route.Route[i].Lat, route.Route[i].Lng
			lat2, lng2 := route.Route[i+1].Lat, route.Route[i+1].Lng
			segmentDistance += calculateHaversineDistance(lat1, lng1, lat2, lng2)
		}
	} else {
		segmentDistance = parseDistance(route.Distance)
	}
	
	stopCount := toIdx - fromIdx + 1
	duration := int((segmentDistance / 20.0) * 60) // Average bus speed 20 km/h
	
	// Comfort score calculation:
	// - Penalize more for number of stops (0.15 per stop)
	// - Penalize less for duration (0.005 per minute)
	// - Base comfort of 1.0
	comfort := 1.0 - (float64(stopCount) * 0.15) - (float64(duration) * 0.005)
	
	return RouteMetrics{
		TotalDistance: segmentDistance,
		StopCount:    stopCount,
		Duration:     duration,
		Comfort:      comfort,
	}
}

func calculateInterchangeViability(firstRoute, secondRoute BusRoute, interchangeStop string) float64 {
	var pos1, pos2 int
	for i, stop := range firstRoute.Route {
		if stopsMatch(stop.StopName, interchangeStop) {
			pos1 = i
			break
		}
	}
	for i, stop := range secondRoute.Route {
		if stopsMatch(stop.StopName, interchangeStop) {
			pos2 = i
			break
		}
	}

	// Calculate actual distances for the segments being used
	distance1 := calculateRouteSegmentDistance(firstRoute.Route, 0, pos1)
	distance2 := calculateRouteSegmentDistance(secondRoute.Route, pos2, len(secondRoute.Route)-1)
	
	totalDistance := distance1 + distance2
	timeFactor := float64((totalDistance / 20.0) * 60) // minutes
	stopsFactor := float64(pos1 + (len(secondRoute.Route) - pos2))
	
	// Position factor - prefer interchanges that are at logical points
	// (not too early in first route and not too late in second route)
	positionQuality := math.Abs(float64(pos1)/float64(len(firstRoute.Route)) - 0.7) +
					  math.Abs(float64(pos2)/float64(len(secondRoute.Route)) - 0.3)
	
	// Weighted scoring:
	// - Distance: 40% weight
	// - Time: 30% weight
	// - Number of stops: 20% weight
	// - Interchange position quality: 10% weight
	return (totalDistance * 0.4) +
		   (timeFactor * 0.3) +
		   (stopsFactor * 0.2) +
		   (positionQuality * 100 * 0.1)
}

func calculateRouteSegmentDistance(stops []BusStop, fromIdx, toIdx int) float64 {
	if fromIdx >= toIdx || fromIdx < 0 || toIdx >= len(stops) {
		return 0
	}
	
	var distance float64
	for i := fromIdx; i < toIdx; i++ {
		lat1, lng1 := stops[i].Lat, stops[i].Lng
		lat2, lng2 := stops[i+1].Lat, stops[i+1].Lng
		distance += calculateHaversineDistance(lat1, lng1, lat2, lng2)
	}
	return distance
}

func calculateHaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		 math.Cos(lat1Rad)*math.Cos(lat2Rad)*
		 math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return R * c
}

func optimizeRoutePath(routes []BusRoute, interchangePoints []string) float64 {
	if len(routes) < 2 {
		return 0
	}

	var totalDistance, totalDuration float64
	var stopCount int
	
	// Calculate metrics for each route segment
	for i, route := range routes {
		var fromIdx, toIdx int
		
		// Find the correct segment of this route to use
		if i == 0 { // First route
			fromIdx = findStopIndex(route, routes[0].Route[0].StopName)
			toIdx = findStopIndex(route, interchangePoints[0])
		} else if i == len(routes)-1 { // Last route
			fromIdx = findStopIndex(route, interchangePoints[i-1])
			toIdx = len(route.Route) - 1
		} else { // Middle routes
			fromIdx = findStopIndex(route, interchangePoints[i-1])
			toIdx = findStopIndex(route, interchangePoints[i])
		}
		
		if fromIdx >= 0 && toIdx >= 0 {
			distance := calculateRouteSegmentDistance(route.Route, fromIdx, toIdx)
			totalDistance += distance
			totalDuration += (distance / 20.0) * 60 // minutes
			stopCount += toIdx - fromIdx + 1
		}
	}

	// Calculate interchange quality
	var interchangeQuality float64
	for i := 0; i < len(routes)-1; i++ {
		interchangeQuality += calculateInterchangeViability(routes[i], routes[i+1], interchangePoints[i])
	}
	
	// Final score calculation:
	// - Distance: 35% weight
	// - Duration: 25% weight
	// - Stop count: 20% weight
	// - Interchange quality: 20% weight
	return (totalDistance * 0.35) +
		   (totalDuration * 0.25) +
		   (float64(stopCount) * 0.20) +
		   (interchangeQuality * 0.20)
}

func findStopIndex(route BusRoute, stopName string) int {
	for i, stop := range route.Route {
		if stopsMatch(stop.StopName, stopName) {
			return i
		}
	}
	return -1
} 