package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/henriquemarlon/city.fun/simulator/internal/domain/entity"
	"github.com/henriquemarlon/city.fun/simulator/internal/infra/repository"
	"github.com/henriquemarlon/city.fun/simulator/internal/usecase"
	"github.com/henriquemarlon/city.fun/simulator/pkg/events"
)

type SensorHandlers struct {
	SensorCreated    events.EventInterface
	SensorRepository repository.SensorRepository
	EventDispatcher  events.EventDispatcherInterface
	SensorChannel    chan<- *entity.Sensor
}

func NewSensorHandlers(
	sensorCreated events.EventInterface,
	sensorRepository repository.SensorRepository,
	eventDispatcher events.EventDispatcherInterface,
	sensorChannel chan<- *entity.Sensor,
) *SensorHandlers {
	return &SensorHandlers{
		SensorCreated:    sensorCreated,
		SensorRepository: sensorRepository,
		EventDispatcher:  eventDispatcher,
		SensorChannel:    sensorChannel,
	}
}

func (s *SensorHandlers) CreateSensor(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var input usecase.CreateSensorInputDTO
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	createSensor := usecase.NewCreateSensorUseCase(s.SensorCreated, s.SensorRepository, s.EventDispatcher)
	output, err := createSensor.Execute(ctx, &input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.SensorChannel <- &entity.Sensor{
		Id:        output.Id,
		Name:      output.Name,
		Latitude:  output.Latitude,
		Longitude: output.Longitude,
		Params:    output.Params,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
}
