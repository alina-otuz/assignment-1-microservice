package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"order-service/internal/domain"
)

// OrderUseCase is the interface the handler depends on.
// Defined here (in the delivery layer) to keep the dependency arrow pointing inward.
type OrderUseCase interface {
	CreateOrder(ctx context.Context, customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, error)
	GetOrder(ctx context.Context, id string) (*domain.Order, error)
	CancelOrder(ctx context.Context, id string) (*domain.Order, error)
	GetRecentPurchases(ctx context.Context, limit int) ([]domain.Order, error)
}

// Handler is the thin HTTP delivery layer.
// Its only jobs: parse the request, delegate to the use case, map errors to HTTP codes.
type Handler struct {
	uc OrderUseCase
}

func NewHandler(uc OrderUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/", h.health)
	r.POST("/orders", h.createOrder)
	r.GET("/orders/recent", h.getRecentPurchases)
	r.GET("/orders/:id", h.getOrder)
	r.PATCH("/orders/:id/cancel", h.cancelOrder)
}

// ── Request / Response DTOs ────────────────────────────────────────────────

type createOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name"   binding:"required"`
	Amount     int64  `json:"amount"      binding:"required,gt=0"`
}

type orderResponse struct {
	ID             string    `json:"id"`
	CustomerID     string    `json:"customer_id"`
	ItemName       string    `json:"item_name"`
	Amount         int64     `json:"amount"`
	Status         string    `json:"status"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func toResponse(o *domain.Order) orderResponse {
	return orderResponse{
		ID:             o.ID,
		CustomerID:     o.CustomerID,
		ItemName:       o.ItemName,
		Amount:         o.Amount,
		Status:         o.Status,
		IdempotencyKey: o.IdempotencyKey,
		CreatedAt:      o.CreatedAt,
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────

// POST /orders
func (h *Handler) createOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Bonus: read optional Idempotency-Key header
	idempotencyKey := c.GetHeader("Idempotency-Key")

	order, err := h.uc.CreateOrder(c.Request.Context(), req.CustomerID, req.ItemName, req.Amount, idempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPaymentServiceUnavailable):
			// Required failure scenario: payment service is down/timed out.
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment service unavailable, please retry later"})
		case errors.Is(err, domain.ErrInvalidAmount),
			errors.Is(err, domain.ErrMissingCustomerID),
			errors.Is(err, domain.ErrMissingItemName):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, toResponse(order))
}

// GET /orders/:id
func (h *Handler) getOrder(c *gin.Context) {
	order, err := h.uc.GetOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, toResponse(order))
}

// PATCH /orders/:id/cancel
func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "order-service"})
}

func (h *Handler) cancelOrder(c *gin.Context) {
	order, err := h.uc.CancelOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrOrderNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		case errors.Is(err, domain.ErrCannotCancelPaidOrder),
			errors.Is(err, domain.ErrOnlyPendingCanBeCancelled):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, toResponse(order))
}

// GET /orders/recent?limit=5
func (h *Handler) getRecentPurchases(c *gin.Context) {
	limit := 5
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be an integer between 1 and 100"})
			return
		}
		limit = n
	}

	orders, err := h.uc.GetRecentPurchases(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	if len(orders) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no recent orders"})
		return
	}

	resp := make([]orderResponse, 0, len(orders))
	for i := range orders {
		o := orders[i]
		resp = append(resp, toResponse(&o))
	}
	c.JSON(http.StatusOK, resp)
}
