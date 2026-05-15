package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"notification-service/internal/idempotency"
	"notification-service/internal/messagebroker/nats"
	"notification-service/internal/notification"
)

func main() {
	maxNotificationRetries := parseIntEnv("MAX_NOTIFICATION_RETRIES", 3)
	initialBackoff := time.Duration(parseIntEnv("INITIAL_BACKOFF_SECONDS", 2)) * time.Second

	// ── Message Broker ──────────────────────────────────────────────────
	natsURL := getEnv("NATS_URL", "nats://nats:4222")
	consumer, err := nats.NewConsumer(natsURL, "payment")
	if err != nil {
		log.Fatalf("failed to create NATS consumer: %v", err)
	}
	defer consumer.Close()

	provider, err := notification.NewProviderFromEnv()
	if err != nil {
		log.Fatalf("failed to initialize notification provider: %v", err)
	}

	idempotencyStore, err := idempotency.NewRedisStoreFromEnv()
	if err != nil {
		log.Fatalf("failed to initialize Redis idempotency store: %v", err)
	}
	defer idempotencyStore.Close()

	log.Printf("Notification service started using provider mode=%s", notification.GetProviderMode())

	// Start consuming messages
	msgs, err := consumer.Consume()
	if err != nil {
		log.Fatalf("failed to start consuming: %v", err)
	}

	log.Println("Notification service started. Waiting for messages...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for msg := range msgs {
			var paymentMsg nats.PaymentCompletedMessage
			if err := json.Unmarshal(msg.Data, &paymentMsg); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				msg.Ack()
				continue
			}

			ctx := context.Background()
			processed, err := idempotencyStore.HasProcessed(ctx, paymentMsg.ID)
			if err != nil {
				log.Printf("Redis idempotency lookup failed for %s: %v", paymentMsg.ID, err)
				if err := msg.Nak(); err != nil {
					log.Printf("Failed to NAK message after Redis lookup error: %v", err)
				}
				continue
			}

			if processed {
				log.Printf("Duplicate message received, skipping: %s", paymentMsg.ID)
				msg.Ack()
				continue
			}

			locked, err := idempotencyStore.AcquireProcessing(ctx, paymentMsg.ID)
			if err != nil {
				log.Printf("Failed to acquire processing lock for %s: %v", paymentMsg.ID, err)
				if err := msg.Nak(); err != nil {
					log.Printf("Failed to NAK message after lock acquisition failure: %v", err)
				}
				continue
			}

			if !locked {
				log.Printf("Message already being processed by another worker: %s", paymentMsg.ID)
				if err := msg.Nak(); err != nil {
					log.Printf("Failed to NAK message when lock unavailable: %v", err)
				}
				continue
			}

			if paymentMsg.CustomerEmail == "fail@example.com" {
				_ = idempotencyStore.ReleaseProcessing(ctx, paymentMsg.ID)
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

			if err := sendWithRetry(ctx, provider, paymentMsg.CustomerEmail, paymentMsg.OrderID, paymentMsg.Amount, maxNotificationRetries, initialBackoff); err != nil {
				log.Printf("Notification delivery failed for order %s after retries: %v", paymentMsg.OrderID, err)
				_ = idempotencyStore.ReleaseProcessing(ctx, paymentMsg.ID)
				if err := msg.Nak(); err != nil {
					log.Printf("Failed to NAK message: %v", err)
				}
				continue
			}

			if err := idempotencyStore.MarkProcessed(ctx, paymentMsg.ID); err != nil {
				log.Printf("Failed to mark message processed for %s: %v", paymentMsg.ID, err)
				_ = idempotencyStore.ReleaseProcessing(ctx, paymentMsg.ID)
				if err := msg.Nak(); err != nil {
					log.Printf("Failed to NAK message after mark processed failure: %v", err)
				}
				continue
			}

			_ = idempotencyStore.ReleaseProcessing(ctx, paymentMsg.ID)

			if err := msg.Ack(); err != nil {
				log.Printf("Failed to ACK message: %v", err)
			}
		}
	}()

	<-sigChan
	log.Println("Shutting down notification service...")
}

func sendWithRetry(ctx context.Context, provider notification.NotificationProvider, email, orderID string, amountCents int64, maxNotificationRetries int, initialBackoff time.Duration) error {
	for attempt := 1; attempt <= maxNotificationRetries; attempt++ {
		err := provider.SendNotification(ctx, email, orderID, amountCents)
		if err == nil {
			return nil
		}

		if attempt == maxNotificationRetries {
			return err
		}

		backoff := initialBackoff * time.Duration(1<<(attempt-1))
		log.Printf("Notification retry %d/%d for order %s after error: %v. Backing off %s", attempt, maxNotificationRetries, orderID, err, backoff)

		select {
		case <-time.After(backoff):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func parseIntEnv(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Invalid integer for %s=%q, using fallback %d", key, value, fallback)
		return fallback
	}
	return parsed
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
