package usecase

import (
	"context"
	"errors"

	"order-service/internal/domain"
)

// ErrCacheMiss indicates the order is not present in the cache (cache-aside read miss).
var ErrCacheMiss = errors.New("cache miss")

// OrderRepository is the Port (driven adapter interface) for persistence.
// The use case depends only on this abstraction, never on *sql.DB.
type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	GetByID(ctx context.Context, id string) (*domain.Order, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)
	ListRecentPaid(ctx context.Context, limit int) ([]domain.Order, error)
	Update(ctx context.Context, order *domain.Order) error
}

// PaymentClient is the Port for the outbound gRPC call to Payment Service.
// This keeps the use case completely decoupled from transport details.
type PaymentClient interface {
	// Authorize calls POST /payments on the Payment Service.
	// Returns the status ("Authorized"/"Declined") and a transactionID.
	Authorize(ctx context.Context, orderID string, amount int64, email string) (status string, transactionID string, err error)
}

// OrderCache is the Port for Redis cache-aside storage of orders.
type OrderCache interface {
	Get(ctx context.Context, id string) (*domain.Order, error)
	Set(ctx context.Context, order *domain.Order) error
	Delete(ctx context.Context, id string) error
}
