package factory

import (
	"context"
	"fmt"
	"strings"

	. "github.com/henriquemarlon/city.fun/simulator/internal/infra/repository"
	"github.com/henriquemarlon/city.fun/simulator/internal/infra/repository/mongodb"
)

func NewRepositoryFromConnectionString(ctx context.Context, conn, database, collection string) (Repository, error) {
	lowerConn := strings.ToLower(conn)
	switch {
	case strings.HasPrefix(lowerConn, "mongodb://"):
		return newMongoDBRepository(conn, database, collection)
	default:
		return nil, fmt.Errorf("unrecognized connection string format: %s", conn)
	}
}

func newMongoDBRepository(conn, database, collection string) (Repository, error) {
	mongodbRepo, err := mongodb.NewMongoDBRepository(conn, database, collection)
	if err != nil {
		return nil, err
	}
	return mongodbRepo, nil
}
