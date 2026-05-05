package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// PaymentCompletedMessage represents the message sent when a payment is completed.
type PaymentCompletedMessage struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

// Publisher publishes messages to NATS JetStream.
type Publisher struct {
	js jetstream.JetStream
}

// NewPublisher creates a new NATS JetStream publisher.
func NewPublisher(natsURL, streamName string) (*Publisher, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create or update the stream
	_, err = js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{streamName + ".>"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	return &Publisher{
		js: js,
	}, nil
}

// Close closes the connection.
func (p *Publisher) Close() error {
	// NATS connection is managed internally
	return nil
}

// PublishPaymentCompleted publishes a payment completed message.
func (p *Publisher) PublishPaymentCompleted(ctx context.Context, orderID, customerEmail string, amount int64, status string) error {
	msg := PaymentCompletedMessage{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		Amount:        amount,
		CustomerEmail: customerEmail,
		Status:        status,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish to JetStream
	_, err = p.js.Publish(ctx, "payment.completed", body)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published payment completed message for order %s", orderID)
	return nil
}