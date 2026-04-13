package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"payment-service/internal/repository/postgres"
	httpTransport "payment-service/internal/transport/http"
	"payment-service/internal/usecase"
)

func main() {
	// ── Database connection ──────────────────────────────────────────────
	dsn := getEnv("PAYMENT_DB_DSN", "host=localhost port=5432 user=postgres password=postgres dbname=payments sslmode=disable")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("cannot reach payments DB: %v", err)
	}
	log.Println("Connected to payments database")

	// ── Manual Dependency Injection (Composition Root) ─────────────────
	paymentRepo := postgres.NewPaymentRepository(db)
	paymentUC := usecase.NewPaymentUseCase(paymentRepo)
	handler := httpTransport.NewHandler(paymentUC)

	// ── Router ──────────────────────────────────────────────────────────
	router := gin.Default()
	handler.RegisterRoutes(router)

	port := getEnv("PAYMENT_PORT", "8081")
	log.Printf("Payment Service listening on :%s", port)
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
