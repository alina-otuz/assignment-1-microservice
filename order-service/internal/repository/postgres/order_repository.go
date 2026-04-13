package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"order-service/internal/domain"
)

// OrderRepository implements usecase.OrderRepository using PostgreSQL.
// The use case never knows this concrete type exists – it only sees the interface.
type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create inserts a new order row.
func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	const q = `
		INSERT INTO orders (id, customer_id, item_name, amount, status, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6,''), $7)`

	_, err := r.db.ExecContext(ctx, q,
		order.ID, order.CustomerID, order.ItemName,
		order.Amount, order.Status, order.IdempotencyKey, order.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("OrderRepository.Create: %w", err)
	}
	return nil
}

// GetByID fetches a single order or returns domain.ErrOrderNotFound.
func (r *OrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	const q = `
		SELECT id, customer_id, item_name, amount, status,
		       COALESCE(idempotency_key,''), created_at
		FROM orders WHERE id = $1`

	var o domain.Order
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&o.ID, &o.CustomerID, &o.ItemName,
		&o.Amount, &o.Status, &o.IdempotencyKey, &o.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("OrderRepository.GetByID: %w", err)
	}
	return &o, nil
}

// GetByIdempotencyKey supports the idempotency bonus feature.
func (r *OrderRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	const q = `
		SELECT id, customer_id, item_name, amount, status,
		       COALESCE(idempotency_key,''), created_at
		FROM orders WHERE idempotency_key = $1`

	var o domain.Order
	err := r.db.QueryRowContext(ctx, q, key).Scan(
		&o.ID, &o.CustomerID, &o.ItemName,
		&o.Amount, &o.Status, &o.IdempotencyKey, &o.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("OrderRepository.GetByIdempotencyKey: %w", err)
	}
	return &o, nil
}

// ListRecentPaid returns the most recent paid orders sorted by created_at descending.
func (r *OrderRepository) ListRecentPaid(ctx context.Context, limit int) ([]domain.Order, error) {
	const q = `
		SELECT id, customer_id, item_name, amount, status,
		       COALESCE(idempotency_key,''), created_at
		FROM orders
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, q, domain.StatusPaid, limit)
	if err != nil {
		return nil, fmt.Errorf("OrderRepository.ListRecentPaid: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Order, 0, limit)
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(
			&o.ID, &o.CustomerID, &o.ItemName,
			&o.Amount, &o.Status, &o.IdempotencyKey, &o.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("OrderRepository.ListRecentPaid scan: %w", err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("OrderRepository.ListRecentPaid rows: %w", err)
	}

	return out, nil
}

// Update saves the mutated status field and publishes a DB notification.
func (r *OrderRepository) Update(ctx context.Context, order *domain.Order) error {
	const q = `UPDATE orders SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, q, order.Status, order.ID)
	if err != nil {
		return fmt.Errorf("OrderRepository.Update: %w", err)
	}

	const notify = `NOTIFY order_updates, $1`
	_, err = r.db.ExecContext(ctx, notify, order.ID)
	if err != nil {
		return fmt.Errorf("OrderRepository.Update notify: %w", err)
	}

	return nil
}
