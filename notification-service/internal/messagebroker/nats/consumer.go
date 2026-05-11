package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type PaymentCompletedMessage struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type Message struct {
	Data []byte
	Ack  func() error
	Nak  func() error
	Term func() error
	Meta func() (*jetstream.MsgMetadata, error)
}

type Consumer struct {
	conn       *nats.Conn
	js         jetstream.JetStream
	consumer   jetstream.Consumer
	dlqSubject string
}


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


	_, err = js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{streamName + ".>"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	
	dlqStreamName := streamName + "-dlq"
	dlqSubject := "dlq." + streamName
	_, err = js.CreateOrUpdateStream(context.Background(), jetstream.StreamConfig{
		Name:     dlqStreamName,
		Subjects: []string{dlqSubject},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create DLQ stream: %w", err)
	}


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
		conn:       nc,
		js:         js,
		consumer:   consumer,
		dlqSubject: dlqSubject,
	}, nil
}

func (c *Consumer) Close() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

// Dead Letter Queue
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
			msgs, err := c.consumer.Fetch(1, jetstream.FetchMaxWait(2*time.Second))
			if err != nil {
				// Timeout
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, nats.ErrTimeout) {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				time.Sleep(250 * time.Millisecond)
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
