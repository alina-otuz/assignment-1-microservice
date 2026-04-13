package usecase

import (
	"context"
	"fmt"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

// PaymentUseCase orchestrates payment authorization.
// Business rules (e.g., MaxAmount limit) live in domain.NewPayment, not here –
// the use case calls the domain and persists the result.
type PaymentUseCase struct {
	repo PaymentRepository
}

func NewPaymentUseCase(repo PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{repo: repo}
}

// Authorize creates and persists a new payment for the given order.
// The domain entity decides Authorized vs Declined based on business rules.
func (uc *PaymentUseCase) Authorize(ctx context.Context, orderID string, amount int64) (*domain.Payment, error) {
	transactionID := uuid.New().String()
	payment, err := domain.NewPayment(uuid.New().String(), orderID, transactionID, amount)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("PaymentUseCase.Authorize: %w", err)
	}

	return payment, nil
}

// GetByOrderID retrieves the payment record associated with a given order.
func (uc *PaymentUseCase) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return uc.repo.GetByOrderID(ctx, orderID)
}
