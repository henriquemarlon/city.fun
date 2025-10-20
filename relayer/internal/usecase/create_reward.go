package usecase

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ethereum/go-ethereum/common"
	"github.com/henriquemarlon/city.fun/relayer/internal/domain/entity"
	"github.com/henriquemarlon/city.fun/relayer/internal/infra/repository"
)

type CreateRewardInputDTO struct {
	Token     common.Address `json:"token"`
	Amount    string         `json:"amount"`
	Receiver  string         `json:"receiver"`
	Latitude  float64        `json:"latitude"`
	Longitude float64        `json:"longitude"`
	Data      string         `json:"data"`
}

type CreateRewardOutputDTO struct {
	Id        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Token     string             `json:"token"`
	Amount    string             `json:"amount"`
	Receiver  string             `json:"receiver"`
	Latitude  float64            `json:"latitude"`
	Longitude float64            `json:"longitude"`
	TxHash    string             `json:"tx_hash"`
	Data      string             `json:"data"`
}

type CreateRewardUseCase struct {
	Repository repository.Repository
}

func NewCreateRewardUseCase(repository repository.Repository) *CreateRewardUseCase {
	return &CreateRewardUseCase{
		Repository: repository,
	}
}

func (uc *CreateRewardUseCase) Execute(ctx context.Context, input *CreateRewardInputDTO) (*CreateRewardOutputDTO, error) {
	amount, ok := new(big.Int).SetString(input.Amount, 10)
	if !ok {
		return nil, errors.New("invalid amount")
	}

	existingReward, err := uc.Repository.FindRewardByLocation(ctx, input.Latitude, input.Longitude)

	var result *entity.Reward

	if err == nil && existingReward != nil {
		existingReward.Data = input.Data
		existingReward.Receiver = common.HexToAddress(input.Receiver).Hex()
		existingReward.Amount = amount.String()
		existingReward.UpdatedAt = time.Now()

		result, err = uc.Repository.UpdateReward(ctx, existingReward)
		if err != nil {
			return nil, fmt.Errorf("failed to update reward: %w", err)
		}
	} else if err == entity.ErrRewardNotFound {
		var createErr error
		newReward, createErr := entity.NewReward(input.Token, amount, common.HexToAddress(input.Receiver), input.Latitude, input.Longitude, input.Data)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create reward: %w", createErr)
		}

		result, createErr = uc.Repository.CreateReward(ctx, newReward)
		if createErr != nil {
			return nil, fmt.Errorf("failed to save reward: %w", createErr)
		}
	} else {
		return nil, fmt.Errorf("failed to check existing reward: %w", err)
	}

	return &CreateRewardOutputDTO{
		Id:        result.Id,
		Token:     result.Token,
		Amount:    result.Amount,
		Receiver:  result.Receiver,
		Latitude:  result.Latitude,
		Longitude: result.Longitude,
		TxHash:    result.TxHash,
		Data:      result.Data,
	}, nil
}
