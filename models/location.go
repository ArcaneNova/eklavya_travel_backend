package models

type Location struct {
    State       string `json:"state"`
    District    string `json:"district"`
    Subdistrict string `json:"subdistrict"`
    Village     string `json:"village"`
    Latitude    float64 `json:"latitude"`
    Longitude   float64 `json:"longitude"`
}

type LocationHierarchy struct {
    States       []string `json:"states,omitempty"`
    Districts    []string `json:"districts,omitempty"`
    Subdistricts []string `json:"subdistricts,omitempty"`
    Villages     []string `json:"villages,omitempty"`
}