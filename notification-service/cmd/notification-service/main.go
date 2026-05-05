package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"notification-service/internal/messagebroker/nats"
)

func main() {
	// ── Message Broker ──────────────────────────────────────────────────
	natsURL := getEnv("NATS_URL", "nats://nats:4222")
	consumer, err := nats.NewConsumer(natsURL, "payment")
	if err != nil {
		log.Fatalf("failed to create NATS consumer: %v", err)
	}
	defer consumer.Close()

	// Start consuming messages
	msgs, err := consumer.Consume()
	if err != nil {
		log.Fatalf("failed to start consuming: %v", err)
	}

	// In-memory store for processed message IDs (for idempotency)
	processedMessages := make(map[string]bool)

	log.Println("Notification service started. Waiting for messages...")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for msg := range msgs {
			var paymentMsg nats.PaymentCompletedMessage
			if err := json.Unmarshal(msg.Data, &paymentMsg); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				// For invalid messages, we can choose to ack or not; since it's invalid, ack to remove
				msg.Ack()
				continue
			}

			// Simulated permanent failure condition
			if paymentMsg.CustomerEmail == "fail@example.com" {
				meta, err := msg.Meta()
				attempts := 1
				if err == nil && meta != nil {
					attempts = int(meta.NumDelivered)
				}

				log.Printf("Permanent processing error for order %s, attempt %d/3", paymentMsg.OrderID, attempts)
				if attempts < 3 {
					msg.Nak()
					continue
				}

				if err := consumer.PublishToDLQ(context.Background(), msg.Data); err != nil {
					log.Printf("Failed to publish message to DLQ: %v", err)
				}
				log.Printf("Moved message to DLQ after %d failed attempts: %s", attempts, paymentMsg.ID)
				msg.Term()
				continue
			}

			// Idempotency check: skip if already processed
			if processedMessages[paymentMsg.ID] {
				log.Printf("Duplicate message received, skipping: %s", paymentMsg.ID)
				msg.Ack() // Acknowledge duplicate to remove from stream
				continue
			}

			// Simulate sending email
			log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%.2f", paymentMsg.CustomerEmail, paymentMsg.OrderID, float64(paymentMsg.Amount)/100)

			// Mark as processed for idempotency
			processedMessages[paymentMsg.ID] = true

			// Acknowledge only after successful processing
			msg.Ack()
		}
	}()

	<-sigChan
	log.Println("Shutting down notification service...")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}