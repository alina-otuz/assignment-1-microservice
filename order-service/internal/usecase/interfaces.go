package usecase

import (
	"context"
	"order-service/internal/domain"
)

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
	Authorize(ctx context.Context, orderID string, amount int64) (status string, transactionID string, err error)
}
