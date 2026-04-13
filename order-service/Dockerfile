FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o order-service ./cmd/order-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/order-service .
EXPOSE 8080
CMD ["./order-service"]
