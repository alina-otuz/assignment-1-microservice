package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// PaymentClient is the concrete adapter that satisfies the usecase.PaymentClient port.
// It holds a *http.Client configured with a 2-second Timeout (set at the Composition Root).
type PaymentClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewPaymentClient(httpClient *http.Client, baseURL string) *PaymentClient {
	return &PaymentClient{httpClient: httpClient, baseURL: baseURL}
}

type authorizeRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type authorizeResponse struct {
	Status        string `json:"status"`
	TransactionID string `json:"transaction_id"`
}

// Authorize posts to POST /payments on the Payment Service.
// The http.Client timeout (2s) is enforced by the caller's Transport layer,
// so a network failure or slow response automatically returns an error here.
func (c *PaymentClient) Authorize(ctx context.Context, orderID string, amount int64) (string, string, error) {
	body, err := json.Marshal(authorizeRequest{OrderID: orderID, Amount: amount})
	if err != nil {
		return "", "", fmt.Errorf("PaymentClient: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/payments", bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("PaymentClient: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Covers: connection refused, timeout (context deadline exceeded), DNS failure, etc.
		return "", "", fmt.Errorf("PaymentClient: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("PaymentClient: unexpected status %d", resp.StatusCode)
	}

	var result authorizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("PaymentClient: decode response: %w", err)
	}

	return result.Status, result.TransactionID, nil
}
