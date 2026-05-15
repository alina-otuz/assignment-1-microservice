package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"order-service/internal/domain"
	"order-service/internal/usecase"
)

const orderKeyPrefix = "order:"

// OrderCache implements usecase.OrderCache with Redis (cache-aside store).
type OrderCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewOrderCache(client *redis.Client, ttl time.Duration) *OrderCache {
	return &OrderCache{client: client, ttl: ttl}
}

func orderKey(id string) string {
	return orderKeyPrefix + id
}

// Get returns a cached order or usecase.ErrCacheMiss when the key is absent.
func (c *OrderCache) Get(ctx context.Context, id string) (*domain.Order, error) {
	data, err := c.client.Get(ctx, orderKey(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, usecase.ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("OrderCache.Get: %w", err)
	}

	var order domain.Order
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, fmt.Errorf("OrderCache.Get unmarshal: %w", err)
	}
	return &order, nil
}

// Set stores an order with the configured TTL.
func (c *OrderCache) Set(ctx context.Context, order *domain.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("OrderCache.Set marshal: %w", err)
	}
	if err := c.client.Set(ctx, orderKey(order.ID), data, c.ttl).Err(); err != nil {
		return fmt.Errorf("OrderCache.Set: %w", err)
	}
	return nil
}

// Delete removes the cached order (invalidation on status change).
func (c *OrderCache) Delete(ctx context.Context, id string) error {
	if err := c.client.Del(ctx, orderKey(id)).Err(); err != nil {
		return fmt.Errorf("OrderCache.Delete: %w", err)
	}
	return nil
}
