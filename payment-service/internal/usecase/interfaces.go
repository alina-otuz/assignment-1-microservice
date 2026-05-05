package usecase

import (
"context"
"payment-service/internal/domain"
)

// PaymentRepository is the driven-adapter Port for payment persistence.
type PaymentRepository interface {
Create(ctx context.Context, payment *domain.Payment) error
GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
FindByAmountRange(ctx context.Context, minAmount, maxAmount int64) ([]*domain.Payment, error)
}
// MessagePublisher is the Port for publishing messages to the message broker.
type MessagePublisher interface {
	PublishPaymentCompleted(ctx context.Context, orderID, customerEmail string, amount int64, status string) error
}