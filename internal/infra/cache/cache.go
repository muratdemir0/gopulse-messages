package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewCache(client *redis.Client, ttl time.Duration) *Cache {
	return &Cache{
		client: client,
		ttl:    ttl,
	}
}

func (s *Cache) Set(ctx context.Context, key string, value interface{}) error {
	err := s.client.Set(ctx, key, value, s.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache key %s: %w", key, err)
	}
	return nil
}

func (s *Cache) Get(ctx context.Context, key string) (string, error) {
	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil // not found
		}
		return "", fmt.Errorf("failed to get cache key %s: %w", key, err)
	}
	return data, nil
}

func (s *Cache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check cache key %s: %w", key, err)
	}
	return count > 0, nil
}

func (s *Cache) Delete(ctx context.Context, key string) error {
	err := s.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache key %s: %w", key, err)
	}
	return nil
}
