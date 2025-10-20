package usecase

import (
	"context"
	"encoding/json"

	"github.com/henriquemarlon/city.fun/simulator/internal/infra/repository"
	"github.com/henriquemarlon/city.fun/simulator/pkg/events"
	"github.com/henriquemarlon/city.fun/simulator/pkg/sampling"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmitDataUseCase struct {
	DataEmitted      events.EventInterface
	SensorRepository repository.SensorRepository
	EventDispatcher  events.EventDispatcherInterface
}

type EmitDataInputDTO struct {
	Id primitive.ObjectID `json:"id"`
}

type EmitDataOutputDTO struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Receiver  string  `json:"receiver"`
	Amount    string  `json:"amount"`
	Data      string  `json:"data"` // JSON string
}

func NewEmitDataUseCase(
	emitData events.EventInterface,
	sensorRepository repository.SensorRepository,
	eventDispatcher events.EventDispatcherInterface,
) *EmitDataUseCase {
	return &EmitDataUseCase{
		DataEmitted:      emitData,
		SensorRepository: sensorRepository,
		EventDispatcher:  eventDispatcher,
	}
}

func (e *EmitDataUseCase) Execute(ctx context.Context, input *EmitDataInputDTO) (*EmitDataOutputDTO, error) {
	res, err := e.SensorRepository.FindSensorById(ctx, input.Id)
	if err != nil {
		return nil, err
	}

	s := sampling.NewConfidenceIntervalGenerator()

	data := make(map[string]float64, len(res.Params))
	for key, interval := range res.Params {
		data[key] = s.GenerateValue(interval.Min, interval.Max, interval.Factor)
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	dto := &EmitDataOutputDTO{
		Name:      res.Name,
		Latitude:  res.Latitude,
		Longitude: res.Longitude,
		Receiver:  res.Receiver,
		Amount:    res.Amount,
		Data:      string(dataBytes),
	}

	e.DataEmitted.SetPayload(dto)
	if err := e.EventDispatcher.Dispatch(e.DataEmitted); err != nil {
		return nil, err
	}
	return dto, nil
}
