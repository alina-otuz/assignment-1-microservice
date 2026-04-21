package grpc

import (
"context"
"errors"

v1 "github.com/alina-otuz/repo-b/protos-gen/api/v1"
"payment-service/internal/domain"
"payment-service/internal/usecase"
"google.golang.org/grpc"
"google.golang.org/grpc/codes"
"google.golang.org/grpc/status"
"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements the gRPC PaymentServiceServer interface.
type Server struct {
v1.UnimplementedPaymentServiceServer
uc *usecase.PaymentUseCase
}

func NewServer(uc *usecase.PaymentUseCase) *Server {
return &Server{uc: uc}
}

func (s *Server) Register(grpcServer *grpc.Server) {
v1.RegisterPaymentServiceServer(grpcServer, s)
}

func (s *Server) ProcessPayment(ctx context.Context, req *v1.PaymentRequest) (*v1.PaymentResponse, error) {
payment, err := s.uc.Authorize(ctx, req.GetOrderId(), req.GetAmount())
if err != nil {
return nil, mapPaymentError(err)
}

return &v1.PaymentResponse{
Id:            payment.ID,
OrderId:       payment.OrderID,
TransactionId: payment.TransactionID,
Amount:        payment.Amount,
Status:        payment.Status,
CreatedAt:     timestamppb.Now(),
}, nil
}

func (s *Server) GetByOrderID(ctx context.Context, req *v1.GetPaymentByOrderIDRequest) (*v1.PaymentResponse, error) {
payment, err := s.uc.GetByOrderID(ctx, req.GetOrderId())
if err != nil {
return nil, mapPaymentError(err)
}

return &v1.PaymentResponse{
Id:            payment.ID,
OrderId:       payment.OrderID,
TransactionId: payment.TransactionID,
Amount:        payment.Amount,
Status:        payment.Status,
CreatedAt:     timestamppb.Now(),
}, nil
}

func (s *Server) ListPayments(ctx context.Context, req *v1.ListPaymentsRequest) (*v1.ListPaymentsResponse, error) {
payments, err := s.uc.ListPayments(ctx, req.GetMinAmount(), req.GetMaxAmount())
if err != nil {
return nil, mapPaymentError(err)
}

var responses []*v1.PaymentResponse
for _, payment := range payments {
responses = append(responses, &v1.PaymentResponse{
Id:            payment.ID,
OrderId:       payment.OrderID,
TransactionId: payment.TransactionID,
Amount:        payment.Amount,
Status:        payment.Status,
CreatedAt:     timestamppb.Now(),
})
}

return &v1.ListPaymentsResponse{
Payments: responses,
}, nil
}

func mapPaymentError(err error) error {
switch {
case errors.Is(err, domain.ErrInvalidPaymentAmount), errors.Is(err, domain.ErrMissingOrderID):
return status.Error(codes.InvalidArgument, err.Error())
case errors.Is(err, domain.ErrPaymentNotFound):
return status.Error(codes.NotFound, err.Error())
default:
return status.Error(codes.Internal, "internal server error")
}
}