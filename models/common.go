package models

type River struct {
    Name         string   `json:"name"`
    Translations []string `json:"translations,omitempty"`
    Length       string   `json:"length,omitempty"`
}