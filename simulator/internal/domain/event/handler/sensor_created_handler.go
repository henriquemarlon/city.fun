package handler

import (
	"encoding/json"
	"log/slog"
	"sync"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/henriquemarlon/city.fun/simulator/internal/domain/entity"
	"github.com/henriquemarlon/city.fun/simulator/pkg/events"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SensorCreatedHandler struct {
	Client        MQTT.Client
	SensorChannel chan *entity.Sensor
}

func NewSensorCreatedHandler(client MQTT.Client, sensorChan chan *entity.Sensor) *SensorCreatedHandler {
	return &SensorCreatedHandler{
		Client:        client,
		SensorChannel: sensorChan,
	}
}

func (h *SensorCreatedHandler) Handle(event events.EventInterface, wg *sync.WaitGroup) {
	defer wg.Done()
	rawPayload := event.GetPayload()

	bytesPayload, err := json.Marshal(rawPayload)
	if err != nil {
		slog.Error("Error serializing the raw payload", "error", err)
	}

	token := h.Client.Publish("sensors/created", 1, false, bytesPayload)
	if token.Error() != nil {
		slog.Error("Failed to publish the message", "error", token.Error())
	}
	token.Wait()

	var payload struct {
		Id        primitive.ObjectID      `json:"id"`
		Name      string                  `json:"name"`
		Latitude  float64                 `json:"latitude"`
		Longitude float64                 `json:"longitude"`
		Params    map[string]entity.Param `json:"params"`
	}
	err = json.Unmarshal(bytesPayload, &payload)
	if err != nil {
		slog.Error("Error deserializing the payload", "error", err)
	}

	slog.Debug(event.GetName(), "id", payload.Id.Hex(), "name", payload.Name, "latitude", payload.Latitude, "longitude", payload.Longitude)

	h.SensorChannel <- &entity.Sensor{
		Id:        payload.Id,
		Name:      payload.Name,
		Latitude:  payload.Latitude,
		Longitude: payload.Longitude,
		Params:    payload.Params,
	}
}
