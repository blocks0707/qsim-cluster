package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements CacheStore using Redis
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache store
func NewRedisCache(client *redis.Client) CacheStore {
	return &RedisCache{client: client}
}

// Set stores a value in the cache with expiration
func (c *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	ctx := context.Background()
	
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = c.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache value: %w", err)
	}

	return nil
}

// Get retrieves a value from the cache
func (c *RedisCache) Get(key string, dest interface{}) error {
	ctx := context.Background()
	
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get cache value: %w", err)
	}

	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// Delete removes a value from the cache
func (c *RedisCache) Delete(key string) error {
	ctx := context.Background()
	
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache value: %w", err)
	}

	return nil
}