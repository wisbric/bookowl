package platform

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	coreplatform "github.com/wisbric/core/pkg/platform"
)

type Platform struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func New(ctx context.Context, dbURL, redisURL string) (*Platform, error) {
	pool, err := coreplatform.NewPostgresPool(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("creating postgres pool: %w", err)
	}

	rdb, err := coreplatform.NewRedisClient(ctx, redisURL)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("creating redis client: %w", err)
	}

	return &Platform{DB: pool, Redis: rdb}, nil
}

func (p *Platform) Close() {
	_ = p.Redis.Close()
	p.DB.Close()
}
