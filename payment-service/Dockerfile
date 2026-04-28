FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY assignment-1-microservice/payment-service/go.mod assignment-1-microservice/payment-service/go.sum ./payment-service/
COPY assignment-1-microservice/protos-gen/ ./protos-gen/
COPY assignment-1-microservice/payment-service/ ./payment-service/
WORKDIR /app/payment-service
RUN go mod download
RUN go build -o payment-service ./cmd/payment-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/payment-service/payment-service .
EXPOSE 8081 50051
CMD ["./payment-service"]
