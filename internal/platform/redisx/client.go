package redisx

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
)

func New(cfg config.Redis) (*redis.Client, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("redis: parse url: %w", err)
	}
	return redis.NewClient(opts), nil
}

// Checker satisfies httpx.Readiness.
type Checker struct {
	Client *redis.Client
}

func (c Checker) Name() string { return "redis" }

func (c Checker) Ready(ctx context.Context) error {
	return c.Client.Ping(ctx).Err()
}
