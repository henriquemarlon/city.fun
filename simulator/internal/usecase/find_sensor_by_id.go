package usecase

import (
	"context"

	"github.com/henriquemarlon/city.fun/simulator/internal/domain/entity"
	"github.com/henriquemarlon/city.fun/simulator/internal/infra/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FindSensorByIdUseCase struct {
	SensorRepository repository.SensorRepository
}

type FindSensorByIdInputDTO struct {
	Id primitive.ObjectID `json:"id"`
}

type FindSensorByIdOutputDTO struct {
	Id        primitive.ObjectID      `json:"id"`
	Name      string                  `json:"name"`
	Latitude  float64                 `json:"latitude"`
	Longitude float64                 `json:"longitude"`
	Receiver  string                  `json:"receiver"`
	Amount    string                  `json:"amount"`
	Params    map[string]entity.Param `json:"params"`
}

func NewFindSensorByIdUseCase(sensorRepository repository.SensorRepository) *FindSensorByIdUseCase {
	return &FindSensorByIdUseCase{SensorRepository: sensorRepository}
}

func (f *FindSensorByIdUseCase) Execute(ctx context.Context, input *FindSensorByIdInputDTO) (*FindSensorByIdOutputDTO, error) {
	sensor, err := f.SensorRepository.FindSensorById(ctx, input.Id)
	if err != nil {
		return nil, err
	}
	return &FindSensorByIdOutputDTO{
		Id:        sensor.Id,
		Name:      sensor.Name,
		Latitude:  sensor.Latitude,
		Longitude: sensor.Longitude,
		Receiver:  sensor.Receiver,
		Amount:    sensor.Amount,
		Params:    sensor.Params,
	}, nil
}
