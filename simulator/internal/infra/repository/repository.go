package repository

import (
	"context"

	"github.com/henriquemarlon/city.fun/simulator/internal/domain/entity"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SensorRepository interface {
	CreateSensor(ctx context.Context, sensor *entity.Sensor) (*entity.Sensor, error)
	FindSensorById(ctx context.Context, id primitive.ObjectID) (*entity.Sensor, error)
	FindAllSensors(ctx context.Context) ([]*entity.Sensor, error)
}

type Repository interface {
	SensorRepository
	Close() error
}
