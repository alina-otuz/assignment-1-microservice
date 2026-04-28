package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/alina-otuz/repo-b/api/v1"
	"order-service/internal/domain"
	"order-service/internal/usecase"
)

// Server implements the gRPC OrderService.
type Server struct {
	v1.UnimplementedOrderServiceServer
	uc    *usecase.OrderUseCase
	dbDSN string
}

func NewServer(uc *usecase.OrderUseCase, dbDSN string) *Server {
	return &Server{uc: uc, dbDSN: dbDSN}
}

func (s *Server) Register(grpcServer *grpc.Server) {
	v1.RegisterOrderServiceServer(grpcServer, s)
}

func (s *Server) CreateOrder(ctx context.Context, req *v1.CreateOrderRequest) (*v1.CreateOrderResponse, error) {
	order, err := s.uc.CreateOrder(ctx, req.GetCustomerId(), req.GetItemName(), req.GetAmount(), "")
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &v1.CreateOrderResponse{Order: toProto(order)}, nil
}

func (s *Server) GetOrder(ctx context.Context, req *v1.GetOrderRequest) (*v1.GetOrderResponse, error) {
	order, err := s.uc.GetOrder(ctx, req.GetId())
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &v1.GetOrderResponse{Order: toProto(order)}, nil
}

func (s *Server) CancelOrder(ctx context.Context, req *v1.CancelOrderRequest) (*v1.CancelOrderResponse, error) {
	order, err := s.uc.CancelOrder(ctx, req.GetId())
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &v1.CancelOrderResponse{Order: toProto(order)}, nil
}

func (s *Server) GetRecentPurchases(ctx context.Context, req *v1.GetRecentPurchasesRequest) (*v1.GetRecentPurchasesResponse, error) {
	orders, err := s.uc.GetRecentPurchases(ctx, int(req.GetLimit()))
	if err != nil {
		return nil, mapOrderError(err)
	}

	resp := &v1.GetRecentPurchasesResponse{Orders: make([]*v1.Order, 0, len(orders))}
	for _, order := range orders {
		resp.Orders = append(resp.Orders, toProto(&order))
	}
	return resp, nil
}

func (s *Server) SubscribeToOrderUpdates(req *v1.SubscribeToOrderUpdatesRequest, stream v1.OrderService_SubscribeToOrderUpdatesServer) error {
	order, err := s.uc.GetOrder(stream.Context(), req.GetId())
	if err != nil {
		return mapOrderError(err)
	}

	if err := stream.Send(&v1.SubscribeToOrderUpdatesResponse{
		Id:        order.ID,
		Status:    order.Status,
		UpdatedAt: timestamppb.Now(),
	}); err != nil {
		return err
	}

	listener := pq.NewListener(s.dbDSN, 10*time.Second, time.Minute, func(event pq.ListenerEventType, err error) {
		if err != nil {
			// connection issues will bubble through Notify channel / Ping
		}
	})
	if err := listener.Listen("order_updates"); err != nil {
		return status.Errorf(codes.Internal, "listen failed: %v", err)
	}
	defer listener.Close()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case n := <-listener.Notify:
			if n == nil || n.Extra != req.GetId() {
				continue
			}
			order, err := s.uc.GetOrder(stream.Context(), req.GetId())
			if err != nil {
				return mapOrderError(err)
			}
			if err := stream.Send(&v1.SubscribeToOrderUpdatesResponse{
				Id:        order.ID,
				Status:    order.Status,
				UpdatedAt: timestamppb.Now(),
			}); err != nil {
				return err
			}
		case <-time.After(90 * time.Second):
			if err := listener.Ping(); err != nil {
				return status.Errorf(codes.Internal, "listener ping failed: %v", err)
			}
		}
	}
}

func toProto(order *domain.Order) *v1.Order {
	return &v1.Order{
		Id:             order.ID,
		CustomerId:     order.CustomerID,
		ItemName:       order.ItemName,
		Amount:         order.Amount,
		Status:         order.Status,
		IdempotencyKey: order.IdempotencyKey,
		CreatedAt:      timestamppb.New(order.CreatedAt),
	}
}

func mapOrderError(err error) error {
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidAmount),
		errors.Is(err, domain.ErrMissingCustomerID),
		errors.Is(err, domain.ErrMissingItemName):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrCannotCancelPaidOrder),
		errors.Is(err, domain.ErrOnlyPendingCanBeCancelled):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrPaymentServiceUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
