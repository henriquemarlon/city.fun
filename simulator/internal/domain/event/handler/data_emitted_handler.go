package handler

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/henriquemarlon/city.fun/simulator/pkg/events"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DataEmittedHandler struct {
	Client    MQTT.Client
	MqttTopic string
}

func NewDataEmittedHandler(client MQTT.Client, mqttTopic string) *DataEmittedHandler {
	return &DataEmittedHandler{
		Client:    client,
		MqttTopic: mqttTopic,
	}
}

func (h *DataEmittedHandler) Handle(event events.EventInterface, wg *sync.WaitGroup) {
	defer wg.Done()
	rawPayload := event.GetPayload()

	bytesPayload, err := json.Marshal(rawPayload)
	if err != nil {
		slog.Error("Error serializing the payload", "error", err)
	}

	token := h.Client.Publish(h.MqttTopic, 1, false, bytesPayload)
	token.WaitTimeout(2 * time.Second)
	if token.Error() != nil {
		slog.Error("Failed to publish the message", "error", token.Error())
	}

	var payload struct {
		Id        primitive.ObjectID `json:"id"`
		Name      string             `json:"name"`
		Latitude  float64            `json:"latitude"`
		Longitude float64            `json:"longitude"`
		Data      string             `json:"data"`
	}
	err = json.Unmarshal(bytesPayload, &payload)
	if err != nil {
		slog.Error("Error deserializing the payload", "error", err)
	}
	slog.Debug(event.GetName(), "id", payload.Id.Hex(), "name", payload.Name, "latitude", payload.Latitude, "longitude", payload.Longitude, "data", payload.Data)
}
