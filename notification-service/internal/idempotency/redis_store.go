package idempotency

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client       *redis.Client
	processedTTL time.Duration
	lockTTL      time.Duration
}

const (
	redisAddrEnv            = "REDIS_ADDR"
	processedTTLEnv         = "IDEMPOTENCY_TTL_SECONDS"
	processingLockTTLEnv    = "IDEMPOTENCY_LOCK_TTL_SECONDS"
	defaultProcessedTTLSec  = 86400
	defaultProcessingLockSec = 30
)

func NewRedisStoreFromEnv() (*RedisStore, error) {
	addr := getEnv(redisAddrEnv, "localhost:6379")
	processedTTL := parseSecondsEnv(processedTTLEnv, defaultProcessedTTLSec)
	lockTTL := parseSecondsEnv(processingLockTTLEnv, defaultProcessingLockSec)

	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("cannot reach Redis at %s: %w", addr, err)
	}

	return &RedisStore{
		client:       client,
		processedTTL: time.Duration(processedTTL) * time.Second,
		lockTTL:      time.Duration(lockTTL) * time.Second,
	}, nil
}

func (r *RedisStore) Close() error {
	return r.client.Close()
}

func (r *RedisStore) HasProcessed(ctx context.Context, paymentID string) (bool, error) {
	res, err := r.client.Exists(ctx, processedKey(paymentID)).Result()
	if err != nil {
		return false, err
	}
	return res > 0, nil
}

func (r *RedisStore) MarkProcessed(ctx context.Context, paymentID string) error {
	return r.client.Set(ctx, processedKey(paymentID), "processed", r.processedTTL).Err()
}

func (r *RedisStore) AcquireProcessing(ctx context.Context, paymentID string) (bool, error) {
	res, err := r.client.SetNX(ctx, processingKey(paymentID), "locked", r.lockTTL).Result()
	if err != nil {
		return false, err
	}
	return res, nil
}

func (r *RedisStore) ReleaseProcessing(ctx context.Context, paymentID string) error {
	return r.client.Del(ctx, processingKey(paymentID)).Err()
}

func processedKey(paymentID string) string {
	return fmt.Sprintf("notification:processed:%s", paymentID)
}

func processingKey(paymentID string) string {
	return fmt.Sprintf("notification:processing:%s", paymentID)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseSecondsEnv(envKey string, fallback int) int {
	value := getEnv(envKey, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
