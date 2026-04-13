package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"

	"order-service/internal/repository/postgres"
	httpTransport "order-service/internal/transport/http"
	grpcTransport "order-service/internal/transport/grpc"
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

	// ── Outbound gRPC client (required: 2-second timeout) ──────────────
	paymentHost := getEnv("PAYMENT_SERVICE_GRPC_HOST", "localhost")
	paymentPort := getEnv("PAYMENT_SERVICE_GRPC_PORT", "50051")
	paymentAddr := fmt.Sprintf("%s:%s", paymentHost, paymentPort)

	paymentClient, err := client.NewPaymentClient(paymentAddr, 2*time.Second)
	if err != nil {
		log.Fatalf("failed to create payment gRPC client: %v", err)
	}
	defer paymentClient.Close()

	// ── Manual Dependency Injection (Composition Root) ─────────────────
	orderRepo := postgres.NewOrderRepository(db)
	orderUC := usecase.NewOrderUseCase(orderRepo, paymentClient)
	handler := httpTransport.NewHandler(orderUC)
	grpcHandler := grpcTransport.NewServer(orderUC, dsn)

	// ── Start gRPC server for order updates and order queries ────────────
	grpcPort := getEnv("ORDER_GRPC_PORT", "50052")
	grpcListener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	grpcHandler.Register(grpcServer)

	go func() {
		log.Printf("Order Service gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// ── Router ──────────────────────────────────────────────────────────
	router := gin.Default()
	handler.RegisterRoutes(router)

	port := getEnv("ORDER_PORT", "8080")
	log.Printf("Order Service HTTP listening on :%s", port)
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
