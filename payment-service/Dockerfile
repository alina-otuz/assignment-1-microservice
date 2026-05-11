FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY payment-service/go.mod payment-service/go.sum ./payment-service/
COPY protos-gen/ ./protos-gen/
COPY payment-service/ ./payment-service/
WORKDIR /app/payment-service
RUN go mod download
RUN go build -o payment-service ./cmd/payment-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/payment-service/payment-service .
EXPOSE 8081 50051
CMD ["./payment-service"]
