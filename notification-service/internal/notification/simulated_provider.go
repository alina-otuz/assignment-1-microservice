package notification

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"
)

type SimulatedProvider struct {
	minLatency  time.Duration
	maxLatency  time.Duration
	failureRate float32
	rand        *rand.Rand
}

func NewSimulatedProvider() *SimulatedProvider {
	return &SimulatedProvider{
		minLatency:  200 * time.Millisecond,
		maxLatency:  800 * time.Millisecond,
		failureRate: 0.20,
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *SimulatedProvider) SendNotification(ctx context.Context, email, orderID string, amountCents int64) error {
	latency := s.minLatency + time.Duration(s.rand.Int63n(int64(s.maxLatency-s.minLatency+1)))

	select {
	case <-time.After(latency):
		// continue
	case <-ctx.Done():
		return ctx.Err()
	}

	if s.rand.Float32() < s.failureRate {
		return fmt.Errorf("simulated network failure while sending notification to %s", email)
	}

	log.Printf("[SimulatedProvider] Sent notification to %s for order %s. Amount: $%.2f. Latency: %s", email, orderID, float64(amountCents)/100, latency)
	return nil
}
