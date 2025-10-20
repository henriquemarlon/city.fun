package entity

import (
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrRewardNotFound = errors.New("reward not found")
	ErrInvalidReward  = errors.New("invalid reward")
)

type Reward struct {
	Id        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Token     string             `bson:"token" json:"token"`
	Amount    string             `bson:"amount" json:"amount"`
	Receiver  string             `bson:"receiver" json:"receiver"`
	Latitude  float64            `bson:"latitude" json:"latitude"`
	Longitude float64            `bson:"longitude" json:"longitude"`
	TxHash    string             `bson:"tx_hash,omitempty" json:"tx_hash"`
	Data      string             `bson:"data" json:"data"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

func NewReward(token common.Address, amount *big.Int, receiver common.Address, latitude float64, longitude float64, data string) (*Reward, error) {
	reward := &Reward{
		Token:     token.Hex(),
		Amount:    amount.String(),
		Receiver:  receiver.Hex(),
		Latitude:  latitude,
		Longitude: longitude,
		Data:      data,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := reward.Validate(); err != nil {
		return nil, err
	}
	return reward, nil
}

func (r *Reward) Validate() error {
	if r.Token == "" || !common.IsHexAddress(r.Token) {
		return ErrInvalidReward
	}
	amount := new(big.Int)
	if _, ok := amount.SetString(r.Amount, 10); !ok || amount.Sign() <= 0 {
		return ErrInvalidReward
	}
	if r.Receiver == "" || !common.IsHexAddress(r.Receiver) {
		return ErrInvalidReward
	}
	if r.Data == "" {
		return ErrInvalidReward
	}
	if r.CreatedAt.IsZero() {
		return ErrInvalidReward
	}
	return nil
}
