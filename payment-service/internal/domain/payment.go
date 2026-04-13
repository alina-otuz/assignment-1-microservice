package domain

import "errors"

// Sentinel errors for the Payment bounded context.
var (
	ErrInvalidPaymentAmount = errors.New("amount must be greater than 0")
	ErrMissingOrderID       = errors.New("order_id is required")
	ErrPaymentNotFound      = errors.New("payment not found")
)

// Status constants for Payment.
const (
	StatusAuthorized = "Authorized"
	StatusDeclined   = "Declined"

	// MaxAmount is the business rule limit: amounts above this are declined.
	// 100000 cents = 1000 units ($1000.00).
	MaxAmount int64 = 100000
)

// Payment is the Payment bounded context's core entity.
// It is intentionally separate from the Order entity in the Order Service –
// no shared code or "common" package is used (microservice rule).
type Payment struct {
	ID            string
	OrderID       string
	TransactionID string
	Amount        int64 // in cents; int64 – never float64 for money
	Status        string
}

// NewPayment validates inputs and applies the payment limit business rule.
func NewPayment(id, orderID, transactionID string, amount int64) (*Payment, error) {
	if orderID == "" {
		return nil, ErrMissingOrderID
	}
	if amount <= 0 {
		return nil, ErrInvalidPaymentAmount
	}

	status := StatusAuthorized
	if amount > MaxAmount {
		// Business rule: amounts exceeding 100,000 cents are always declined.
		status = StatusDeclined
	}

	return &Payment{
		ID:            id,
		OrderID:       orderID,
		TransactionID: transactionID,
		Amount:        amount,
		Status:        status,
	}, nil
}
