package notification

import (
	"context"
	"os"
	"strings"
)

type NotificationProvider interface {
	SendNotification(ctx context.Context, email, orderID string, amountCents int64) error
}

const (
	providerModeEnv   = "PROVIDER_MODE"
	providerModeReal  = "REAL"
	providerModeSim   = "SIMULATED"
)

func NewProviderFromEnv() (NotificationProvider, error) {
	mode := strings.ToUpper(getEnv(providerModeEnv, providerModeSim))

	switch mode {
	case providerModeReal:
		return NewSMTPProviderFromEnv()
	case providerModeSim:
		fallthrough
	default:
		return NewSimulatedProvider(), nil
	}
}

func GetProviderMode() string {
	return strings.ToUpper(getEnv(providerModeEnv, providerModeSim))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
