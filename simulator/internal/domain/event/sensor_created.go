package event

import (
	"time"
)

type SensorCreated struct {
	Name    string
	Payload interface{}
}

func NewSensorCreated() *SensorCreated {
	return &SensorCreated{
		Name: "sensor_created",
	}
}

func (e *SensorCreated) GetName() string {
	return e.Name
}

func (e *SensorCreated) GetPayload() interface{} {
	return e.Payload
}

func (e *SensorCreated) SetPayload(payload interface{}) {
	e.Payload = payload
}

func (e *SensorCreated) GetDateTime() time.Time {
	return time.Now()
}
