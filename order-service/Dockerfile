FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY order-service/go.mod order-service/go.sum ./order-service/
COPY . .
RUN cd order-service && go mod download
RUN cd order-service && go build -o order-service ./cmd/order-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/order-service/order-service .
EXPOSE 8080
CMD ["./order-service"]
