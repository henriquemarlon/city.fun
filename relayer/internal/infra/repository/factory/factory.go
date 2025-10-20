package factory

import (
	"context"
	"fmt"
	"strings"

	. "github.com/henriquemarlon/city.fun/relayer/internal/infra/repository"
	"github.com/henriquemarlon/city.fun/relayer/internal/infra/repository/mongodb"
)

func NewRepositoryFromConnectionString(ctx context.Context, conn, database, collection string) (Repository, error) {
	lowerConn := strings.ToLower(conn)
	switch {
	case strings.HasPrefix(lowerConn, "mongodb://"):
		return newMongoDBRepository(ctx, conn, database, collection)
	default:
		return nil, fmt.Errorf("unrecognized connection string format: %s", conn)
	}
}

func newMongoDBRepository(ctx context.Context, conn, database, collection string) (Repository, error) {
	mongodbRepo, err := mongodb.NewMongoDBRepository(ctx, conn, database, collection)
	if err != nil {
		return nil, err
	}
	return mongodbRepo, nil
}
