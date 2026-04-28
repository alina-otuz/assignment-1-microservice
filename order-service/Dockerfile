FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY assignment-1-microservice/order-service/go.mod assignment-1-microservice/order-service/go.sum ./order-service/
COPY assignment-1-microservice/protos-gen/ ./protos-gen/
COPY assignment-1-microservice/order-service/ ./order-service/
WORKDIR /app/order-service
RUN go mod download
RUN go build -o order-service ./cmd/order-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/order-service/order-service .
EXPOSE 8080 50052
CMD ["./order-service"]
