package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	v1 "github.com/alina-otuz/repo-b/api/v1"
)

// PaymentClient is the concrete adapter that satisfies the usecase.PaymentClient port.
// It calls the Payment Service over gRPC.
type PaymentClient struct {
	grpcClient v1.PaymentServiceClient
	conn       *grpc.ClientConn
	timeout    time.Duration
}

func NewPaymentClient(grpcAddr string, timeout time.Duration) (*PaymentClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("PaymentClient: dial %s: %w", grpcAddr, err)
	}

	return &PaymentClient{
		grpcClient: v1.NewPaymentServiceClient(conn),
		conn:       conn,
		timeout:    timeout,
	}, nil
}

func (c *PaymentClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Authorize calls the Payment Service via gRPC.
func (c *PaymentClient) Authorize(ctx context.Context, orderID string, amount int64) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.grpcClient.ProcessPayment(ctx, &v1.ProcessPaymentRequest{OrderId: orderID, Amount: amount})
	if err != nil {
		return "", "", fmt.Errorf("PaymentClient.ProcessPayment: %w", err)
	}

	payment := resp.GetPayment()
	return payment.GetStatus(), payment.GetTransactionId(), nil
}
