FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY order-service/go.mod order-service/go.sum ./order-service/
COPY protos-gen/ ./protos-gen/
COPY order-service/ ./order-service/
WORKDIR /app/order-service
RUN go mod download
RUN go build -o order-service ./cmd/order-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/order-service/order-service .
EXPOSE 8080 50052
CMD ["./order-service"]
