package notification

import (
	"context"
	"fmt"
	"net/smtp"
	"strconv"
)

type SMTPProvider struct {
	host      string
	port      int
	username  string
	password  string
	fromEmail string
	addr      string
	auth      smtp.Auth
}

func NewSMTPProviderFromEnv() (NotificationProvider, error) {
	host := getEnv("SMTP_HOST", "")
	if host == "" {
		return nil, fmt.Errorf("SMTP_HOST is required for REAL provider")
	}

	portStr := getEnv("SMTP_PORT", "587")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT %q: %w", portStr, err)
	}

	fromEmail := getEnv("SMTP_FROM", "no-reply@example.com")
	username := getEnv("SMTP_USERNAME", "")
	password := getEnv("SMTP_PASSWORD", "")

	addr := fmt.Sprintf("%s:%d", host, port)
	var auth smtp.Auth
	if username != "" || password != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}

	return &SMTPProvider{
		host:      host,
		port:      port,
		username:  username,
		password:  password,
		fromEmail: fromEmail,
		addr:      addr,
		auth:      auth,
	}, nil
}

func (s *SMTPProvider) SendNotification(ctx context.Context, email, orderID string, amountCents int64) error {
	body := fmt.Sprintf("To: %s\r\nSubject: Order %s Completed\r\n\r\nYour order %s has been completed. Amount: $%.2f\r\n", email, orderID, orderID, float64(amountCents)/100)

	err := smtp.SendMail(s.addr, s.auth, s.fromEmail, []string{email}, []byte(body))
	if err != nil {
		return fmt.Errorf("smtp send failed: %w", err)
	}

	return nil
}
