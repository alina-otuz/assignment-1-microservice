package usecase

import (
	"context"
	"fmt"
	"order-service/internal/domain"

	"github.com/google/uuid"
)

// OrderUseCase orchestrates all business flows for the Order bounded context.
// It depends only on the Port interfaces defined in interfaces.go.
type OrderUseCase struct {
	repo          OrderRepository
	paymentClient PaymentClient
}

func NewOrderUseCase(repo OrderRepository, paymentClient PaymentClient) *OrderUseCase {
	return &OrderUseCase{repo: repo, paymentClient: paymentClient}
}

// CreateOrder is the primary use case:
//  1. Validate the request via domain.NewOrder (domain invariants).
//  2. Persist the Pending order.
//  3. Synchronously call the Payment Service.
//  4. Update order status based on the payment result.
//
// Failure modes:
//   - Payment service unreachable (timeout/network) → mark Failed, return ErrPaymentServiceUnavailable.
//   - Payment declined → mark Failed, return order (not an error from caller perspective).
//
// Idempotency: if an idempotency key is provided and already used, return the existing order.
func (uc *OrderUseCase) CreateOrder(
	ctx context.Context,
	customerID, itemName string,
	amount int64,
	idempotencyKey string,
) (*domain.Order, error) {

	// --- Idempotency check (Bonus) ---
	if idempotencyKey != "" {
		existing, err := uc.repo.GetByIdempotencyKey(ctx, idempotencyKey)
		if err == nil && existing != nil {
			return existing, nil // Return the already-processed order.
		}
	}

	// --- Domain construction (validates invariants) ---
	order, err := domain.NewOrder(uuid.New().String(), customerID, itemName, amount, idempotencyKey)
	if err != nil {
		return nil, err
	}

	// --- Persist Pending order first ---
	if err := uc.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to persist order: %w", err)
	}

	// --- Call Payment Service (synchronous REST, 2-second timeout) ---
	status, _, err := uc.paymentClient.Authorize(ctx, order.ID, order.Amount)
	if err != nil {
		// Payment service is down or timed out.
		// Design decision: mark as Failed so the order has a definite terminal state
		// rather than being orphaned as Pending. During defense: "Failed is honest –
		// the payment was attempted and did not complete successfully."
		order.MarkFailed()
		_ = uc.repo.Update(ctx, order) // Best-effort update; ignore secondary error.
		return nil, domain.ErrPaymentServiceUnavailable
	}

	// --- Apply payment result to order status ---
	if status == "Authorized" {
		order.MarkPaid()
	} else {
		order.MarkFailed()
	}

	if err := uc.repo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	return order, nil
}

// GetOrder retrieves an order by its ID.
func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return order, nil
}

// CancelOrder enforces the cancellation business rule via the domain entity.
func (uc *OrderUseCase) CancelOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := order.Cancel(); err != nil {
		// domain.Cancel returns domain-level errors (ErrCannotCancelPaidOrder, etc.)
		return nil, err
	}

	if err := uc.repo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to persist cancellation: %w", err)
	}

	return order, nil
}

// GetRecentPurchases returns most recent paid orders (purchases) sorted by time descending.
func (uc *OrderUseCase) GetRecentPurchases(ctx context.Context, limit int) ([]domain.Order, error) {
	return uc.repo.ListRecentPaid(ctx, limit)
}
