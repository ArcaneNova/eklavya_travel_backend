package models

type Station struct {
    Code        string     `bson:"code" json:"code"`
    Name        string     `bson:"name" json:"name"`
    City        string     `bson:"city" json:"city"`
    State       string     `bson:"state" json:"state"`
    Location    GeoPoint   `bson:"location" json:"location"`
    Connections []Connection `bson:"connections" json:"connections"`
}

type GeoPoint struct {
    Latitude  float64 `bson:"latitude" json:"latitude"`
    Longitude float64 `bson:"longitude" json:"longitude"`
}

type Connection struct {
    ToStation string    `bson:"to_station" json:"to_station"`
    Trains    []TrainConnection `bson:"trains" json:"trains"`
}

type TrainConnection struct {
    TrainNumber int     `bson:"train_number" json:"train_number"`
    Departure   string  `bson:"departure" json:"departure"`
    Arrival     string  `bson:"arrival" json:"arrival"`
    Distance    float64 `bson:"distance" json:"distance"`
    Day         int     `bson:"day" json:"day"`
}