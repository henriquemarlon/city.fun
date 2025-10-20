package usecase

import (
	"context"

	"github.com/henriquemarlon/city.fun/simulator/internal/domain/entity"
	"github.com/henriquemarlon/city.fun/simulator/internal/infra/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FindAllSensorsUseCase struct {
	SensorRepository repository.SensorRepository
}

type FindAllSensorsOutputDTO struct {
	Id        primitive.ObjectID      `json:"id"`
	Name      string                  `json:"name"`
	Latitude  float64                 `json:"latitude"`
	Longitude float64                 `json:"longitude"`
	Receiver  string                  `json:"receiver"`
	Amount    string                  `json:"amount"`
	Params    map[string]entity.Param `json:"params"`
}

func NewFindAllSensorsUseCase(sensorRepository repository.SensorRepository) *FindAllSensorsUseCase {
	return &FindAllSensorsUseCase{SensorRepository: sensorRepository}
}

func (f *FindAllSensorsUseCase) Execute(ctx context.Context) ([]FindAllSensorsOutputDTO, error) {
	sensors, err := f.SensorRepository.FindAllSensors(ctx)
	if err != nil {
		return nil, err
	}
	var output []FindAllSensorsOutputDTO
	for _, sensor := range sensors {
		output = append(output, FindAllSensorsOutputDTO{
			Id:        sensor.Id,
			Name:      sensor.Name,
			Latitude:  sensor.Latitude,
			Longitude: sensor.Longitude,
			Receiver:  sensor.Receiver,
			Amount:    sensor.Amount,
			Params:    sensor.Params,
		})
	}
	return output, nil
}
