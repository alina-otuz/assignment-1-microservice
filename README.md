# AP2 Assignment 1 — Clean Architecture Microservices

## Overview

This repository implements a small Go microservice platform with separate Order, Payment, and Notification bounded contexts.

- `order-service`: HTTP + gRPC API for creating orders, querying orders, cancelling pending orders, and streaming order status.
- `payment-service`: HTTP + gRPC API for payment authorization and payment lookup.
- `notification-service`: NATS JetStream consumer that logs simulated email notifications for completed payments.
- `protos/`: protobuf definitions for the order/payment APIs.
- `protos-gen/`: generated Go code from the protobuf definitions.

The architecture uses Clean Architecture principles, separate PostgreSQL databases, and gRPC plus message broker communication between services.

---

## Architecture

Each service is organized with a thin delivery layer, business use cases, domain entities, and repository adapters.

- `internal/transport`: delivery adapters (HTTP, gRPC)
- `internal/usecase`: application/business logic and ports
- `internal/domain`: domain entities, invariants, and sentinel errors
- `internal/repository/postgres`: PostgreSQL adapters

Dependencies flow inward only: transport → usecase → domain/repository ports.

### Service boundaries

| Service               | Responsibility                            | Owns data                  | Communication |
|-----------------------|--------------------------------------------|----------------------------|---------------|
| `order-service`       | Order lifecycle, create/cancel/query orders | `orders` DB                | gRPC → payment-service, gRPC external clients |
| `payment-service`     | Authorize payments, publish payment events | `payments` DB              | receives gRPC from order-service, publishes NATS events |
| `notification-service`| Consume payment events and simulate notifications | none                       | NATS JetStream |

### Infrastructure

`docker-compose.yml` wires:
- `orders-db` (`postgres:16-alpine`) on host port `5432`
- `payments-db` (`postgres:16-alpine`) on host port `5433`
- `nats` broker on host port `4222`
- `order-service` HTTP `8080`, gRPC `50052`
- `payment-service` HTTP `8081`, gRPC `50051`
- `notification-service` consuming from NATS

---

## Project Structure

```
assignment-1-microservice/
├── docker-compose.yml
├── order-service/
│   ├── cmd/order-service/main.go
│   ├── internal/
│   │   ├── domain/order.go
│   │   ├── repository/postgres/order_repository.go
│   │   ├── transport/http/handler.go
│   │   ├── transport/grpc/server.go
│   │   └── usecase/
│   │       ├── client/payment_client.go
│   │       ├── interfaces.go
│   │       └── order_usecase.go
│   ├── migrations/001_create_orders.sql
│   └── Dockerfile
├── payment-service/
│   ├── cmd/payment-service/main.go
│   ├── internal/
│   │   ├── domain/payment.go
│   │   ├── messagebroker/nats/publisher.go
│   │   ├── repository/postgres/payment_repository.go
│   │   ├── transport/http/handler.go
│   │   ├── transport/grpc/server.go
│   │   └── usecase/
│   │       ├── interfaces.go
│   │       └── payment_usecase.go
│   ├── migrations/001_create_payments.sql
│   └── Dockerfile
├── notification-service/
│   ├── cmd/notification-service/main.go
│   └── internal/messagebroker/nats/consumer.go
├── protos/
└── protos-gen/
```

---

## Running Locally

### Option A — Docker Compose (recommended)

```bash
docker-compose up --build
```

Open ports:
- Order Service HTTP: `http://localhost:8080`
- Payment Service HTTP: `http://localhost:8081`
- Payment Service gRPC: `localhost:50051`
- Order Service gRPC: `localhost:50052`
- NATS: `localhost:4222`

### Option B — Manual

**Prerequisites:** Go 1.21+, PostgreSQL.

```bash
psql -U postgres -c "CREATE DATABASE orders;"
psql -U postgres -c "CREATE DATABASE payments;"
psql -U postgres -d orders   -f order-service/migrations/001_create_orders.sql
psql -U postgres -d payments -f payment-service/migrations/001_create_payments.sql

cd payment-service
go mod tidy
go run ./cmd/payment-service

cd ../order-service
go mod tidy
go run ./cmd/order-service
```

> The notification service requires NATS and is started automatically by Docker Compose.

---

## HTTP API

### Order Service

`POST /orders`
- Creates a new order and triggers payment authorization over gRPC.
- Request body:
  - `customer_id` (string, required)
  - `item_name` (string, required)
  - `amount` (int64, required, cents)
  - `email` (string, required)
- Optional header: `Idempotency-Key`

`GET /orders/:id`
- Fetch an order by ID.

`GET /orders/recent?limit={n}`
- Returns recent paid purchases.

`PATCH /orders/:id/cancel`
- Cancels a pending order only.

### Payment Service

`GET /`
- Health check.

`POST /payments`
- Authorize a payment manually.
- Request body:
  - `order_id` (string, required)
  - `amount` (int64, required, cents)
  - `email` (string, required)

`GET /payments/:order_id`
- Get payment status for a given order.

---

## gRPC API Examples

#### Create an order using gRPC
```bash
grpcurl -plaintext -d '{"customer_id":"cust-001","item_name":"Laptop","amount":50000,"email":"customer@example.com"}' localhost:50052 api.v1.OrderService/CreateOrder
```

#### Subscribe to real-time order updates
```bash
go run ./order-service/cmd/order-subscriber --addr localhost:50052 --order-id {order_id}
```

#### Get recent purchases using gRPC
```bash
grpcurl -plaintext -d '{"limit":5}' localhost:50052 api.v1.OrderService/GetRecentPurchases
```

---

## Notification Service

`notification-service` consumes `payment.completed` events from NATS JetStream and simulates sending email notifications. It logs successful notifications and handles duplicate messages with an in-memory idempotency check.

Invoke-RestMethod -Uri 'http://localhost:8081/payments' -Method Post -ContentType 'application/json' -Body (@{order_id='order-dlq-1'; amount=1000; email='fail@example.com'} | ConvertTo-Json)
---

## Business Rules Summary

| Rule                                  | Implemented in                                 |
|---------------------------------------|------------------------------------------------|
| Order amount must be > 0               | `order-service/internal/domain/order.go`       |
| Order cancellation only when Pending   | `order-service/internal/domain/order.go`       |
| Paid orders cannot be cancelled        | `order-service/internal/domain/order.go`       |
| Payment amount > 100000 is declined    | `payment-service/internal/domain/payment.go`   |
| Payment event publication on success   | `payment-service/internal/usecase/payment_usecase.go` |
| Order idempotency with header          | `order-service/internal/transport/http/handler.go` |
| gRPC payment timeout                  | `order-service/cmd/order-service/main.go`      |

---

## Protobuf and Generated Code

Protobuf sources live under `protos/`.
The repo includes generated Go code in `protos-gen/`, which corresponds to the external module `github.com/alina-otuz/repo-b`.

Run `buf generate` from the `protos/` directory to regenerate Go artifacts locally.
