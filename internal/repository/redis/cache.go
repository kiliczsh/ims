// Package redis provides Redis implementations for caching and session storage.
// It implements the cache repository interface for high-performance data access.
package redis

import (
	"context"
	"encoding/json"
	"time"

	"ims/internal/repository"

	"github.com/redis/go-redis/v9"
)

type cacheRepository struct {
	client *redis.Client
}

func NewCacheRepository(client *redis.Client) repository.CacheRepository {
	return &cacheRepository{client: client}
}

func (r *cacheRepository) SetMessageCache(ctx context.Context, messageID string, data interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	key := "message:" + messageID
	return r.client.Set(ctx, key, jsonData, ttl).Err()
}

func (r *cacheRepository) GetMessageCache(ctx context.Context, messageID string) (interface{}, error) {
	key := "message:" + messageID

	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var data interface{}
	err = json.Unmarshal([]byte(result), &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func NewRedisClient(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return client, nil
}
