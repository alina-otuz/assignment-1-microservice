package main

import (
	"context"
	"flag"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	v1 "github.com/alina-otuz/repo-b/protos-gen/api/v1"
)

func main() {
	addr := flag.String("addr", "localhost:50052", "gRPC server address")
	orderID := flag.String("order-id", "", "Order ID to subscribe to")
	flag.Parse()

	if *orderID == "" {
		log.Fatal("--order-id is required")
	}

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("failed to dial gRPC server: %v", err)
	}
	defer conn.Close()

	client := v1.NewOrderServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	stream, err := client.SubscribeToOrderUpdates(ctx, &v1.GetOrderRequest{Id: *orderID})
	if err != nil {
		log.Fatalf("SubscribeToOrderUpdates failed: %v", err)
	}

	log.Printf("subscribed to order updates for %s", *orderID)
	for {
		update, err := stream.Recv()
		if err == io.EOF {
			log.Println("stream closed by server")
			return
		}
		if err != nil {
			log.Fatalf("stream receive error: %v", err)
		}
		log.Printf("order %s status changed: %s", update.GetId(), update.GetStatus())
	}
}
