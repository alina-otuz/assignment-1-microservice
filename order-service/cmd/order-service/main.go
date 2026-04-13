package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"order-service/internal/repository/postgres"
	httpTransport "order-service/internal/transport/http"
	"order-service/internal/usecase"
	"order-service/internal/usecase/client"
)

func main() {
	// ── Database connection ──────────────────────────────────────────────
	dsn := getEnv("ORDER_DB_DSN", "host=localhost port=5432 user=postgres password=postgres dbname=orders sslmode=disable")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("cannot reach orders DB: %v", err)
	}
	log.Println("Connected to orders database")

	// ── Outbound HTTP client (required: 2-second timeout) ─────────────
	paymentHTTPClient := &http.Client{
		Timeout: 2 * time.Second, // Required failure scenario: order service must not hang.
	}

	// ── Manual Dependency Injection (Composition Root) ─────────────────
	paymentBaseURL := getEnv("PAYMENT_SERVICE_URL", "http://localhost:8081")
	paymentClient := client.NewPaymentClient(paymentHTTPClient, paymentBaseURL)

	orderRepo := postgres.NewOrderRepository(db)
	orderUC := usecase.NewOrderUseCase(orderRepo, paymentClient)
	handler := httpTransport.NewHandler(orderUC)

	// ── Router ──────────────────────────────────────────────────────────
	router := gin.Default()
	handler.RegisterRoutes(router)

	port := getEnv("ORDER_PORT", "8080")
	log.Printf("Order Service listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("router.Run: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
