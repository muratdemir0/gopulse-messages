package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/muratdemir0/gopulse-messages/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/extra/redisotel/v9"
)

func New(cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := redisotel.InstrumentTracing(rdb); err != nil {
		return nil, fmt.Errorf("failed to instrument Redis tracing: %w", err)
	}

	if err := redisotel.InstrumentMetrics(rdb); err != nil {
		return nil, fmt.Errorf("failed to instrument Redis metrics: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return rdb, nil
}
