package domain

import (
	"errors"
	"time"
)

// Sentinel errors – imported by both use case and transport layers.
var (
	ErrInvalidAmount             = errors.New("amount must be greater than 0")
	ErrMissingCustomerID         = errors.New("customer_id is required")
	ErrMissingItemName           = errors.New("item_name is required")
	ErrOrderNotFound             = errors.New("order not found")
	ErrCannotCancelPaidOrder     = errors.New("paid orders cannot be cancelled")
	ErrOnlyPendingCanBeCancelled = errors.New("only pending orders can be cancelled")
	ErrPaymentServiceUnavailable = errors.New("payment service unavailable")
	ErrDuplicateIdempotencyKey   = errors.New("duplicate idempotency key")
)

// Status constants for Order.
const (
	StatusPending   = "Pending"
	StatusPaid      = "Paid"
	StatusFailed    = "Failed"
	StatusCancelled = "Cancelled"
)

// Order is the core domain entity for the Order bounded context.
// It must NOT import any HTTP, JSON, or framework-specific packages.
type Order struct {
	ID             string
	CustomerID     string
	ItemName       string
	Amount         int64 // in cents; int64 – never float64 for money
	Status         string
	IdempotencyKey string
	CreatedAt      time.Time
}

// NewOrder validates invariants and returns a new Pending order.
func NewOrder(id, customerID, itemName string, amount int64, idempotencyKey string) (*Order, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if customerID == "" {
		return nil, ErrMissingCustomerID
	}
	if itemName == "" {
		return nil, ErrMissingItemName
	}
	return &Order{
		ID:             id,
		CustomerID:     customerID,
		ItemName:       itemName,
		Amount:         amount,
		Status:         StatusPending,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now().UTC(),
	}, nil
}

// MarkPaid transitions status to Paid.
func (o *Order) MarkPaid() {
	o.Status = StatusPaid
}

// MarkFailed transitions status to Failed.
func (o *Order) MarkFailed() {
	o.Status = StatusFailed
}

// Cancel enforces the cancellation invariant:
//   - Only Pending orders may be cancelled.
//   - Paid orders must never be cancelled.
func (o *Order) Cancel() error {
	if o.Status == StatusPaid {
		return ErrCannotCancelPaidOrder
	}
	if o.Status != StatusPending {
		return ErrOnlyPendingCanBeCancelled
	}
	o.Status = StatusCancelled
	return nil
}
