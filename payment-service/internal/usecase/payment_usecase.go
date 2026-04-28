package usecase

import (
"context"
"fmt"
"payment-service/internal/domain"

"github.com/google/uuid"
)

// PaymentUseCase orchestrates payment authorization.
// Business rules (e.g., MaxAmount limit) live in domain.NewPayment, not here.
// The use case calls the domain and persists the result.
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

// ListPayments retrieves all payments within an optional amount range.
// Validates that minAmount <= maxAmount if both are specified.
// If maxAmount is 0, it means unlimited. If minAmount is 0, it means no lower bound.
func (uc *PaymentUseCase) ListPayments(ctx context.Context, minAmount, maxAmount int64) ([]*domain.Payment, error) {
// Validation: if both are specified and non-zero, ensure min <= max
if minAmount > 0 && maxAmount > 0 && minAmount > maxAmount {
return nil, fmt.Errorf("PaymentUseCase.ListPayments: minAmount must be less than or equal to maxAmount")
}

payments, err := uc.repo.FindByAmountRange(ctx, minAmount, maxAmount)
if err != nil {
return nil, fmt.Errorf("PaymentUseCase.ListPayments: %w", err)
}

return payments, nil
}