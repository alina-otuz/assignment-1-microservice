# AP2 Assignment 1 — Clean Architecture Microservices (Order & Payment)

## Overview

A two-service platform built in Go, demonstrating Clean Architecture, bounded contexts, separate data ownership, and resilient synchronous REST communication.

---

## Architecture Decisions

### Clean Architecture (per service)

Each service is structured as four concentric layers. The **Dependency Rule** is strictly enforced: inner layers never import outer ones.

```
  ┌────────────────────────────────────┐
  │   Delivery (transport/http)        │  ← Gin handlers; parses HTTP, calls use case
  │ ┌──────────────────────────────┐   │
  │ │   Use Case (usecase/)        │   │  ← All business logic & orchestration
  │ │ ┌──────────────────────┐     │   │
  │ │ │   Domain (domain/)   │     │   │  ← Entities, invariants, sentinel errors
  │ │ └──────────────────────┘     │   │
  │ │   Repository Port (interface)│   │
  │ └──────────────────────────────┘   │
  │   Repository (repository/postgres) │  ← Concrete DB adapter
  └────────────────────────────────────┘
```

**Why this structure?**
- **Thin handlers**: handlers only parse requests, call one use case method, and map errors to HTTP codes. No business logic.
- **Use case owns decisions**: state transitions (`MarkPaid`, `MarkFailed`, `Cancel`) are triggered by the use case after interacting with ports.
- **Domain owns invariants**: `NewOrder` rejects `amount <= 0` before anything is persisted. `Cancel()` enforces the paid-order rule.
- **Interfaces (Ports)**: `OrderRepository` and `PaymentClient` are interfaces defined in the use case layer. The use case is testable without a database or HTTP server.

---

### Microservice Decomposition & Bounded Contexts

| Concern              | Order Service              | Payment Service             |
|----------------------|----------------------------|-----------------------------|
| Database             | `orders` DB (port 5432)   | `payments` DB (port 5433)  |
| Domain entity        | `domain.Order`             | `domain.Payment`            |
| Responsibility       | Order lifecycle management | Payment authorization        |
| Owns data            | `orders` table             | `payments` table            |

**No shared code**: each service has its own `internal/domain` package. There is no `common/` or `shared/` module — a distributed monolith anti-pattern.

---

### REST Communication & Timeout

Order Service → Payment Service via `POST /payments`.

```
Order Service                     Payment Service
     │                                  │
     │── POST /payments ──────────────► │
     │   {"order_id": ..., "amount": ...}│
     │                                  │── apply business rule (amount > 100000?)
     │◄── 201 {"status": "Authorized"} ─│
     │                                  │
     │  (update order → "Paid")         │
```

The outbound `http.Client` is created at the Composition Root with a **2-second timeout**:
```go
paymentHTTPClient := &http.Client{Timeout: 2 * time.Second}
```
This satisfies the required failure scenario: if Payment Service is slow or down, the Order Service never hangs.

---

### Failure Handling

| Scenario                         | Behaviour                                              |
|----------------------------------|--------------------------------------------------------|
| Payment Service down / timeout   | Order marked `Failed`, HTTP 503 returned to client     |
| Payment declined (amount > 1000) | Order marked `Failed`, HTTP 201 returned (order exists)|
| Payment authorized               | Order marked `Paid`, HTTP 201 returned                 |

**Design decision — why `Failed` instead of `Pending` on timeout?**

Leaving the order as `Pending` implies it is still actionable, which is misleading: the payment attempt was made but did not complete. Marking it `Failed` gives the order a definite terminal state. The client can create a new order to retry. This avoids ghost `Pending` orders accumulating in the database.

---

### Idempotency

Pass an `Idempotency-Key` header on `POST /orders`. If the same key is sent twice, the original order is returned without creating a duplicate order or calling the Payment Service again.

```
POST /orders
Idempotency-Key: alina-key-meow
```

Implementation: the key is stored in a `UNIQUE` column on the `orders` table. The use case checks for an existing order with that key before proceeding.

---

## Project Structure

```
AP2_Assignment1/
├── docker-compose.yml
├── order-service/
│   ├── cmd/order-service/main.go          ← Composition Root (manual DI)
│   ├── internal/
│   │   ├── domain/order.go                ← Entity + invariants + sentinel errors
│   │   ├── usecase/
│   │   │   ├── interfaces.go              ← Ports: OrderRepository, PaymentClient
│   │   │   ├── order_usecase.go           ← Business logic
│   │   │   └── client/payment_client.go   ← Outbound HTTP adapter
│   │   ├── repository/postgres/
│   │   │   └── order_repository.go        ← DB adapter
│   │   └── transport/http/
│   │       └── handler.go                 ← Gin handlers (thin delivery layer)
│   ├── migrations/001_create_orders.sql
│   └── Dockerfile
└── payment-service/
    ├── cmd/payment-service/main.go         ← Composition Root (manual DI)
    ├── internal/
    │   ├── domain/payment.go               ← Entity + business rule (MaxAmount)
    │   ├── usecase/
    │   │   ├── interfaces.go               ← Port: PaymentRepository
    │   │   └── payment_usecase.go          ← Business logic
    │   ├── repository/postgres/
    │   │   └── payment_repository.go       ← DB adapter
    │   └── transport/http/
    │       └── handler.go                  ← Gin handlers
    ├── migrations/001_create_payments.sql
    └── Dockerfile
```

---

## Running Locally

### Option A — Docker Compose (recommended)

```bash
docker-compose up --build
```

Services:
- Order Service: http://localhost:8080
- Payment Service: http://localhost:8081

### Option B — Manual

**Prerequisites:** Go 1.21+, PostgreSQL running.

```bash
# Create databases
psql -U postgres -c "CREATE DATABASE orders;"
psql -U postgres -c "CREATE DATABASE payments;"

# Run migrations
psql -U postgres -d orders   -f order-service/migrations/001_create_orders.sql
psql -U postgres -d payments -f payment-service/migrations/001_create_payments.sql

# Start Payment Service (terminal 1)
cd payment-service
go mod tidy
go run ./cmd/payment-service

# Start Order Service (terminal 2)
cd order-service
go mod tidy
go run ./cmd/order-service
```

---

## API Examples

### Order Service

#### Create an order (payment authorized)
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "cust-001", "item_name": "Laptop", "amount": 50000}'
```
Expected: `201 Created`, order status `"Paid"`

#### Create an order (payment declined — amount > 100000)
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "cust-001", "item_name": "Sports Car", "amount": 500000}'
```
Expected: `201 Created`, order status `"Failed"`

#### Get order by ID
```bash
curl http://localhost:8080/orders/{id}
```

#### Cancel a pending order
```bash
curl -X PATCH http://localhost:8080/orders/{id}/cancel
```
Expected: `200 OK` if Pending; `409 Conflict` if Paid.

#### Idempotent order creation (bonus)
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: order-idem-key-001" \
  -d '{"customer_id": "cust-001", "item_name": "Laptop", "amount": 50000}'
# Second identical call returns the same order without duplicating it.
```

### Payment Service

#### Get payment status for an order
```bash
curl http://localhost:8081/payments/{order_id}
```

#### Simulate payment service down (for 503 test)
```bash
# Stop payment-service, then:
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "cust-001", "item_name": "Test", "amount": 10000}'
# Expected: 503 Service Unavailable (within ~2 seconds)
```

---

## Business Rules Summary

| Rule                          | Location            | Detail                                       |
|-------------------------------|---------------------|----------------------------------------------|
| Amount must be > 0            | `domain.NewOrder`   | Returns `ErrInvalidAmount`                   |
| Amount stored as int64        | All layers          | Never float64; monetary precision guaranteed |
| Amount > 100,000 → Declined   | `domain.NewPayment` | Hard limit in Payment bounded context        |
| Paid orders cannot be cancelled | `domain.Order.Cancel` | Returns `ErrCannotCancelPaidOrder`       |
| Only Pending can be cancelled | `domain.Order.Cancel` | Returns `ErrOnlyPendingCanBeCancelled`   |
| HTTP client timeout: 2 seconds | `main.go` (Order)  | `&http.Client{Timeout: 2 * time.Second}`     |

---

## Grading Self-Assessment

| Criterion              | Implementation                                                        |
|------------------------|-----------------------------------------------------------------------|
| Clean Architecture     | 4 layers per service; interfaces as ports; DI at composition root    |
| Microservice decomposition | Separate DBs, separate modules, no shared code                   |
| REST communication     | Order→Payment via HTTP with 2s timeout; proper status codes          |
| Functionality          | All 5 endpoints; PostgreSQL; all business rules enforced             |
| Documentation & Diagram | This README + architecture diagram (architecture_diagram.svg)      |
| Bonus (Idempotency)    | Idempotency-Key header; unique DB constraint; use case check         |
