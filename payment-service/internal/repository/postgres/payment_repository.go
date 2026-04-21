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

// FindByAmountRange fetches all payments within the specified amount range.
// If minAmount is 0, no lower limit is applied.
// If maxAmount is 0, no upper limit is applied.
func (r *PaymentRepository) FindByAmountRange(ctx context.Context, minAmount, maxAmount int64) ([]*domain.Payment, error) {
var q string
var args []interface{}

if minAmount > 0 && maxAmount > 0 {
q = `SELECT id, order_id, transaction_id, amount, status
     FROM payments WHERE amount >= $1 AND amount <= $2`
args = []interface{}{minAmount, maxAmount}
} else if minAmount > 0 {
q = `SELECT id, order_id, transaction_id, amount, status
     FROM payments WHERE amount >= $1`
args = []interface{}{minAmount}
} else if maxAmount > 0 {
q = `SELECT id, order_id, transaction_id, amount, status
     FROM payments WHERE amount <= $1`
args = []interface{}{maxAmount}
} else {
q = `SELECT id, order_id, transaction_id, amount, status FROM payments`
}

rows, err := r.db.QueryContext(ctx, q, args...)
if err != nil {
return nil, fmt.Errorf("PaymentRepository.FindByAmountRange: %w", err)
}
defer rows.Close()

var payments []*domain.Payment
for rows.Next() {
var p domain.Payment
if err := rows.Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status); err != nil {
return nil, fmt.Errorf("PaymentRepository.FindByAmountRange: %w", err)
}
payments = append(payments, &p)
}

if err = rows.Err(); err != nil {
return nil, fmt.Errorf("PaymentRepository.FindByAmountRange: %w", err)
}

return payments, nil
}