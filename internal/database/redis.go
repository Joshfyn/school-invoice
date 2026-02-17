package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	Client *redis.Client
}

func NewRedis(redisURL string) (*Redis, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &Redis{Client: client}, nil
}

func (r *Redis) Close() error {
	return r.Client.Close()
}

// Health checks if the Redis connection is healthy
func (r *Redis) Health(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Set stores a key-value pair with expiration
func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

// Delete removes a key
func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, keys...).Err()
}

// Enqueue adds a job to a queue (for background workers)
func (r *Redis) Enqueue(ctx context.Context, queue string, data string) error {
	return r.Client.LPush(ctx, queue, data).Err()
}

// Dequeue retrieves and removes a job from a queue (blocking)
func (r *Redis) Dequeue(ctx context.Context, queue string, timeout time.Duration) (string, error) {
	result, err := r.Client.BRPop(ctx, timeout, queue).Result()
	if err != nil {
		return "", err
	}
	if len(result) < 2 {
		return "", fmt.Errorf("unexpected result from BRPop")
	}
	return result[1], nil
}
