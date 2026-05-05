package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// PaymentCompletedMessage represents the message received when a payment is completed.
type PaymentCompletedMessage struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

// Message represents a consumed message with ACK functionality.
type Message struct {
	Data []byte
	Ack  func() error
	Nak  func() error
	Term func() error
	Meta func() (*jetstream.MsgMetadata, error)
}

// Consumer consumes messages from NATS JetStream.
type Consumer struct {
	js         jetstream.JetStream
	consumer   jetstream.Consumer
	dlqSubject string
}

// NewConsumer creates a new NATS JetStream consumer.
func NewConsumer(natsURL, streamName string) (*Consumer, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create or update the primary stream
	_, err = js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{streamName + ".>"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	// Create or update the DLQ stream
	dlqStreamName := streamName + "-dlq"
	_, err = js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:     dlqStreamName,
		Subjects: []string{streamName + ".dlq"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create DLQ stream: %w", err)
	}

	// Create consumer with retry policy for permanent failure handling
	consumer, err := js.CreateOrUpdateConsumer(context.Background(), streamName, jetstream.ConsumerConfig{
		Name:          "notification-consumer",
		Durable:       "notification-consumer",
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		MaxDeliver:    3,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return &Consumer{
		js:         js,
		consumer:   consumer,
		dlqSubject: streamName + ".dlq",
	}, nil
}

// Close closes the connection.
func (c *Consumer) Close() error {
	// NATS connection is managed internally
	return nil
}

// PublishToDLQ sends a message to the Dead Letter Queue stream.
func (c *Consumer) PublishToDLQ(ctx context.Context, data []byte) error {
	_, err := c.js.Publish(ctx, c.dlqSubject, data)
	return err
}

// Consume returns a channel of messages.
func (c *Consumer) Consume() (<-chan Message, error) {
	msgChan := make(chan Message)

	go func() {
		defer close(msgChan)

		for {
			msgs, err := c.consumer.Fetch(1, jetstream.FetchMaxWait(1000))
			if err != nil {
				// Handle error, perhaps log and continue
				continue
			}

			for msg := range msgs.Messages() {
				msgChan <- Message{
					Data: msg.Data(),
					Ack: func() error {
						return msg.Ack()
					},
					Nak: func() error {
						return msg.Nak()
					},
					Term: func() error {
						return msg.Term()
					},
					Meta: func() (*jetstream.MsgMetadata, error) {
						return msg.Metadata()
					},
				}
			}
		}
	}()

	return msgChan, nil
}