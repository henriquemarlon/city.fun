package usecase

import (
	"context"
	"fmt"

	"github.com/henriquemarlon/city.fun/relayer/internal/infra/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UpdateRewardTxHashUseCase struct {
	Repository repository.Repository
}

func NewUpdateRewardTxHashUseCase(repository repository.Repository) *UpdateRewardTxHashUseCase {
	return &UpdateRewardTxHashUseCase{
		Repository: repository,
	}
}

func (uc *UpdateRewardTxHashUseCase) Execute(ctx context.Context, rewardId primitive.ObjectID, txHash string) error {
	err := uc.Repository.UpdateRewardTxHash(ctx, rewardId, txHash)
	if err != nil {
		return fmt.Errorf("failed to update reward tx hash: %w", err)
	}
	return nil
}
