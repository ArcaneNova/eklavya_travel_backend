package models

type BusRoute struct {
    City          string    `json:"city"`
    RouteName     string    `json:"route_name"`
    StartingStage string    `json:"starting_stage"`
    EndingStage   string    `json:"ending_stage"`
    Distance      float64   `json:"distance"`
    Route         []BusStop `json:"route"`
}

type BusStop struct {
    StopName string  `json:"stop_name"`
    Lat      float64 `json:"lat"`
    Lng      float64 `json:"lng"`
}

type RouteSegment struct {
    RouteName     string    `json:"route_name"`
    Stops         []BusStop `json:"stops"`
    InterchangeAt string    `json:"interchange_at,omitempty"`
}

type BusRouteSearch struct {
    City     string `json:"city"`
    FromStop string `json:"from_stop"`
    ToStop   string `json:"to_stop"`
}

type BusRouteResponse struct {
    DirectRoutes   []BusRoute     `json:"direct_routes"`
    Interchanges   []RouteSegment `json:"interchanges,omitempty"`
    TotalDistance  float64        `json:"total_distance"`
    EstimatedTime  float64        `json:"estimated_time"`
}