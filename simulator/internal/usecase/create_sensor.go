package usecase

import (
	"context"

	"github.com/henriquemarlon/city.fun/simulator/internal/domain/entity"
	"github.com/henriquemarlon/city.fun/simulator/internal/infra/repository"
	"github.com/henriquemarlon/city.fun/simulator/pkg/events"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CreateSensorUseCase struct {
	SensorCreated    events.EventInterface
	SensorRepository repository.SensorRepository
	EventDispatcher  events.EventDispatcherInterface
}

type CreateSensorInputDTO struct {
	Name      string                  `json:"name"`
	Latitude  float64                 `json:"latitude"`
	Longitude float64                 `json:"longitude"`
	Receiver  string                  `json:"receiver"`
	Amount    string                  `json:"amount"`
	Params    map[string]entity.Param `json:"params"`
}

type CreateSensorOutputDTO struct {
	Id        primitive.ObjectID      `json:"id"`
	Name      string                  `json:"name"`
	Latitude  float64                 `json:"latitude"`
	Longitude float64                 `json:"longitude"`
	Receiver  string                  `json:"receiver"`
	Amount    string                  `json:"amount"`
	Params    map[string]entity.Param `json:"params"`
}

func NewCreateSensorUseCase(sensorCreated events.EventInterface, sensorRepository repository.SensorRepository, eventDispatcher events.EventDispatcherInterface) *CreateSensorUseCase {
	return &CreateSensorUseCase{
		SensorCreated:    sensorCreated,
		SensorRepository: sensorRepository,
		EventDispatcher:  eventDispatcher,
	}
}

func (c *CreateSensorUseCase) Execute(ctx context.Context, input *CreateSensorInputDTO) (*CreateSensorOutputDTO, error) {
	sensor := entity.NewSensor(input.Name, input.Latitude, input.Longitude, input.Receiver, input.Amount, input.Params)
	res, err := c.SensorRepository.CreateSensor(ctx, sensor)
	if err != nil {
		return nil, err
	}

	dto := &CreateSensorOutputDTO{
		Id:        res.Id,
		Name:      res.Name,
		Latitude:  res.Latitude,
		Longitude: res.Longitude,
		Receiver:  res.Receiver,
		Amount:    res.Amount,
		Params:    res.Params,
	}

	c.SensorCreated.SetPayload(dto)
	if err := c.EventDispatcher.Dispatch(c.SensorCreated); err != nil {
		return nil, err
	}

	return dto, nil
}
