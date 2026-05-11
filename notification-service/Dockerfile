# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod
COPY notification-service/go.mod notification-service/go.sum ./notification-service/
RUN cd /app/notification-service && go mod download

# Copy source code
COPY notification-service/ ./notification-service/

WORKDIR /app/notification-service

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o notification-service ./cmd/notification-service

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/notification-service/notification-service .

# Run the binary
CMD ["./notification-service"]