package repository

import (
	"context"

	"github.com/henriquemarlon/city.fun/relayer/internal/domain/entity"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RewardRepository interface {
	CreateReward(ctx context.Context, reward *entity.Reward) (*entity.Reward, error)
	FindRewardByLocation(ctx context.Context, latitude, longitude float64) (*entity.Reward, error)
	UpdateReward(ctx context.Context, reward *entity.Reward) (*entity.Reward, error)
	UpdateRewardTxHash(ctx context.Context, rewardId primitive.ObjectID, txHash string) error
}

type Repository interface {
	RewardRepository
	Close() error
}
