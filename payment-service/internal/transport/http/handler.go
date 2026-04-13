package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"payment-service/internal/domain"
)

// PaymentUseCase is the interface this handler depends on (inward dependency).
type PaymentUseCase interface {
	Authorize(ctx context.Context, orderID string, amount int64) (*domain.Payment, error)
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
}

// Handler is the thin delivery layer for the Payment Service.
type Handler struct {
	uc PaymentUseCase
}

func NewHandler(uc PaymentUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/", h.health)
	r.POST("/payments", h.authorize)
	r.GET("/payments/:order_id", h.getByOrderID)
}

// ── DTOs ──────────────────────────────────────────────────────────────────

type authorizeRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Amount  int64  `json:"amount"   binding:"required,gt=0"`
}

type paymentResponse struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
}

func toResponse(p *domain.Payment) paymentResponse {
	return paymentResponse{
		ID:            p.ID,
		OrderID:       p.OrderID,
		TransactionID: p.TransactionID,
		Amount:        p.Amount,
		Status:        p.Status,
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────

// POST /payments – called by Order Service to authorize a payment.
func (h *Handler) authorize(c *gin.Context) {
	var req authorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.uc.Authorize(c.Request.Context(), req.OrderID, req.Amount)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidPaymentAmount) || errors.Is(err, domain.ErrMissingOrderID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Always 201 – even for Declined payments, the resource was created.
	c.JSON(http.StatusCreated, toResponse(payment))
}

// GET /payments/:order_id – look up payment status for a given order.
func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "payment-service"})
}

func (h *Handler) getByOrderID(c *gin.Context) {
	payment, err := h.uc.GetByOrderID(c.Request.Context(), c.Param("order_id"))
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, toResponse(payment))
}
