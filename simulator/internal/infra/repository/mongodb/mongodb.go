package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBRepository struct {
	Collection *mongo.Collection
}

func NewMongoDBRepository(conn, database, collection string) (*MongoDBRepository, error) {
	clientOpts := options.Client().ApplyURI(conn)
	client, err := mongo.Connect(context.TODO(), clientOpts)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(context.TODO(), nil); err != nil {
		return nil, err
	}

	coll := client.Database(database).Collection(collection)

	return &MongoDBRepository{
		Collection: coll,
	}, nil
}

func (m *MongoDBRepository) Close() error {
	return m.Collection.Database().Client().Disconnect(context.Background())
}