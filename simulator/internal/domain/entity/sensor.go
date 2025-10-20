package entity

import (
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrSensorNotFound = errors.New("sensor not found")
)

type Sensor struct {
	Id        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Latitude  float64            `bson:"latitude" json:"latitude"`
	Longitude float64            `bson:"longitude" json:"longitude"`
	Receiver  string             `bson:"receiver" json:"receiver"`
	Amount    string             `bson:"amount" json:"amount"`
	Params    map[string]Param   `bson:"params" json:"params"`
}

type Param struct {
	Min    int     `json:"min"`
	Max    int     `json:"max"`
	Factor float64 `json:"z"`
}

func NewSensor(name string, latitude float64, longitude float64, receiver string, amount string, params map[string]Param) *Sensor {
	sensor := &Sensor{
		Name:      name,
		Latitude:  latitude,
		Longitude: longitude,
		Receiver:  receiver,
		Amount:    amount,
		Params:    params,
	}
	if err := sensor.Validate(); err != nil {
		return nil
	}
	return sensor
}

func (s *Sensor) Validate() error {
	if s.Name == "" {
		return errors.New("name is required")
	}
	if s.Latitude == 0 {
		return errors.New("latitude is required")
	}
	if s.Longitude == 0 {
		return errors.New("longitude is required")
	}
	if s.Receiver == "" {
		return errors.New("receiver is required")
	}
	if s.Amount == "" {
		return errors.New("amount is required and must be positive")
	}
	if len(s.Params) == 0 {
		return errors.New("params is required")
	}
	return nil
}
