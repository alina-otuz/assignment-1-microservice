FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY payment-service/go.mod payment-service/go.sum ./payment-service/
COPY . .
RUN cd payment-service && go mod download
RUN cd payment-service && go build -o payment-service ./cmd/payment-service

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/payment-service/payment-service .
EXPOSE 8081
CMD ["./payment-service"]
