package mongodb

import (
	"context"

	"github.com/henriquemarlon/city.fun/relayer/internal/domain/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *MongoDBRepository) CreateReward(ctx context.Context, input *entity.Reward) (*entity.Reward, error) {
	res, err := s.Collection.InsertOne(ctx, input)
	if err != nil {
		return nil, err
	}

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, mongo.ErrNilValue
	}

	return s.FindRewardById(ctx, id)
}

func (s *MongoDBRepository) FindRewardById(ctx context.Context, id primitive.ObjectID) (*entity.Reward, error) {
	var reward entity.Reward
	err := s.Collection.FindOne(ctx, bson.M{"_id": id}).Decode(&reward)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, entity.ErrRewardNotFound
		}
		return nil, err
	}
	return &reward, nil
}

func (s *MongoDBRepository) FindRewardByLocation(ctx context.Context, latitude, longitude float64) (*entity.Reward, error) {
	var reward entity.Reward
	filter := bson.M{
		"latitude":  latitude,
		"longitude": longitude,
	}
	err := s.Collection.FindOne(ctx, filter).Decode(&reward)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, entity.ErrRewardNotFound
		}
		return nil, err
	}
	return &reward, nil
}

func (s *MongoDBRepository) UpdateReward(ctx context.Context, reward *entity.Reward) (*entity.Reward, error) {
	filter := bson.M{"_id": reward.Id}
	update := bson.M{
		"$set": bson.M{
			"token":      reward.Token,
			"amount":     reward.Amount,
			"receiver":   reward.Receiver,
			"latitude":   reward.Latitude,
			"longitude":  reward.Longitude,
			"tx_hash":    reward.TxHash,
			"data":       reward.Data,
			"updated_at": reward.UpdatedAt,
		},
	}

	result, err := s.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, entity.ErrRewardNotFound
	}

	return s.FindRewardById(ctx, reward.Id)
}

func (s *MongoDBRepository) UpdateRewardTxHash(ctx context.Context, rewardId primitive.ObjectID, txHash string) error {
	filter := bson.M{"_id": rewardId}
	update := bson.M{"$set": bson.M{"tx_hash": txHash}}

	result, err := s.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return entity.ErrRewardNotFound
	}

	return nil
}
