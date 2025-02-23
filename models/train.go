package models

import (
    // "time" 
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type Train struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
    TrainNumber int               `bson:"train_number" json:"train_number"`
    Name        string            `bson:"name" json:"name"`
    Type        string            `bson:"type" json:"type"`
    FromStation string            `bson:"from_station" json:"from_station"`
    ToStation   string            `bson:"to_station" json:"to_station"`
    Duration    string            `bson:"duration" json:"duration"`
    Schedule    []TrainStop       `bson:"schedule" json:"schedule"`
    Classes     []string          `bson:"classes" json:"classes"`
}

type TrainStop struct {
    Station    string  `bson:"station" json:"station"`
    Arrival    string  `bson:"arrival" json:"arrival"`
    Departure  string  `bson:"departure" json:"departure"`
    Day        int     `bson:"day" json:"day"`
    Distance   float64 `bson:"distance" json:"distance"`
    Platform   string  `bson:"platform" json:"platform"`
    Halt       string  `bson:"halt" json:"halt"`
    Latitude   float64 `bson:"latitude" json:"latitude"`
    Longitude  float64 `bson:"longitude" json:"longitude"`
}