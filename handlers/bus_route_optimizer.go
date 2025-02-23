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
	distance := parseDistance(route.Distance)
	stopCount := toIdx - fromIdx + 1
	duration := getDurationMinutes(route.Distance)
	
	comfort := 1.0 - (float64(stopCount) * 0.1) - (float64(duration) * 0.01)
	
	return RouteMetrics{
		TotalDistance: distance,
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

	distanceFactor := parseDistance(firstRoute.Distance) + parseDistance(secondRoute.Distance)
	timeFactor := float64(getDurationMinutes(firstRoute.Distance) + getDurationMinutes(secondRoute.Distance))
	stopsFactor := float64(len(firstRoute.Route) + len(secondRoute.Route))
	positionFactor := math.Abs(float64(pos1)/float64(len(firstRoute.Route)) - float64(pos2)/float64(len(secondRoute.Route)))

	return distanceFactor*0.4 + timeFactor*0.3 + stopsFactor*0.2 + positionFactor*0.1
}

func optimizeRoutePath(routes []BusRoute, interchangePoints []string) float64 {
	if len(routes) < 2 {
		return 0
	}

	var totalDistance, totalDuration float64
	for _, route := range routes {
		totalDistance += parseDistance(route.Distance)
		totalDuration += float64(getDurationMinutes(route.Distance))
	}

	var interchangeQuality float64
	for i := 0; i < len(routes)-1; i++ {
		interchangeQuality += calculateInterchangeViability(routes[i], routes[i+1], interchangePoints[i])
	}

	return totalDistance*0.3 + totalDuration*0.3 + interchangeQuality*0.4
} 