package usecase

import (
	"context"
	"payment-service/internal/domain"
)

// PaymentRepository is the driven-adapter Port for payment persistence.
type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
}
