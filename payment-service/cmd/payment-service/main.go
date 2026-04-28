package main

import (
	"database/sql"
	"log"
	"net"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"

	"payment-service/internal/repository/postgres"
	httpTransport "payment-service/internal/transport/http"
	grpcTransport "payment-service/internal/transport/grpc"
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
	httpHandler := httpTransport.NewHandler(paymentUC)
	grpcHandler := grpcTransport.NewServer(paymentUC)

	// ── Start gRPC server ───────────────────────────────────────────────
	grpcPort := getEnv("PAYMENT_GRPC_PORT", "50051")
	grpcListener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	grpcHandler.Register(grpcServer)

	go func() {
		log.Printf("Payment gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// ── Router ──────────────────────────────────────────────────────────
	router := gin.Default()
	httpHandler.RegisterRoutes(router)

	port := getEnv("PAYMENT_PORT", "8081")
	log.Printf("Payment HTTP listening on :%s", port)
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
