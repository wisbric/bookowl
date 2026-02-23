package platform

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Platform struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func New(ctx context.Context, dbURL, redisURL string) (*Platform, error) {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("creating pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("parsing Redis URL: %w", err)
	}
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging Redis: %w", err)
	}

	return &Platform{DB: pool, Redis: rdb}, nil
}

func (p *Platform) Close() {
	_ = p.Redis.Close()
	p.DB.Close()
}
