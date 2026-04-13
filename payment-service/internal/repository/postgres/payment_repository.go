package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"payment-service/internal/domain"
)

// PaymentRepository implements usecase.PaymentRepository using PostgreSQL.
type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create inserts a new payment row into the payments database.
// Note: this is a completely separate database from the orders database.
func (r *PaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	const q = `
		INSERT INTO payments (id, order_id, transaction_id, amount, status)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.ExecContext(ctx, q,
		payment.ID, payment.OrderID, payment.TransactionID,
		payment.Amount, payment.Status,
	)
	if err != nil {
		return fmt.Errorf("PaymentRepository.Create: %w", err)
	}
	return nil
}

// GetByOrderID fetches the payment for a given order ID.
func (r *PaymentRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	const q = `
		SELECT id, order_id, transaction_id, amount, status
		FROM payments WHERE order_id = $1`

	var p domain.Payment
	err := r.db.QueryRowContext(ctx, q, orderID).Scan(
		&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("PaymentRepository.GetByOrderID: %w", err)
	}
	return &p, nil
}
