package grpc

import (
	"context"
	"errors"
	"fmt"
	"grpc-course/gen/order"
	"grpc-course/internal/order/application"
	"grpc-course/internal/order/domain"
	"io"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server реализует gRPC-сервис заказов.
type Server struct {
	order.UnimplementedOrderServiceServer
	service orderCreator
}

// orderCreator определяет операцию создания заказа, необходимую gRPC-адаптеру.
type orderCreator interface {
	CreateOrder(ctx context.Context, id, currency string, items []domain.OrderItem) error
}

// NewServer создаёт gRPC-сервер заказов с сервисом приложения.
func NewServer(service *application.OrderService) *Server {
	return newServer(service)
}

// newServer создаёт gRPC-сервер с реализацией операции создания заказа.
func newServer(service orderCreator) *Server {
	return &Server{
		service: service,
	}
}

// CreateOrder создаёт заказ через сервис приложения и возвращает его идентификатор.
func (s *Server) CreateOrder(
	ctx context.Context,
	request *order.CreateOrderRequest,
) (*order.CreateOrderResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	items := make([]domain.OrderItem, 0, len(request.GetItems()))
	for _, item := range request.GetItems() {
		items = append(items, domain.OrderItem{
			ProductID:      item.GetProductId(),
			Quantity:       item.GetQuantity(),
			UnitPriceMinor: item.GetUnitPriceMinor(),
		})
	}

	err := s.service.CreateOrder(
		ctx,
		request.GetId(),
		request.GetCurrency(),
		items,
	)
	if errors.Is(err, domain.ErrOrderAlreadyExists) {
		return nil, status.Error(codes.AlreadyExists, "order already exists")
	}
	if errors.Is(err, domain.ErrQuantityMustBePositive) {
		return nil, invalidArgument("items.quantity", "must be positive")
	}
	if errors.Is(err, domain.ErrOrderIDRequired) {
		return nil, invalidArgument("id", "is required")
	}
	if errors.Is(err, domain.ErrProductIDRequired) {
		return nil, invalidArgument("items.product_id", "is required")
	}
	if errors.Is(err, domain.ErrCurrencyRequired) {
		return nil, invalidArgument("currency", "is required")
	}
	if errors.Is(err, domain.ErrItemsRequired) {
		return nil, invalidArgument("items", "at least one item is required")
	}
	if errors.Is(err, domain.ErrInvalidPrice) {
		return nil, invalidArgument("items.unit_price_minor", "must not be negative")
	}
	if errors.Is(err, domain.ErrPaymentUnavailable) {
		return nil, status.Error(codes.Unavailable, "payment service unavailable")
	}
	if errors.Is(err, domain.ErrPaymentTimeout) {
		return nil, status.Error(codes.DeadlineExceeded, "payment request timeout")
	}
	if errors.Is(err, domain.ErrReservationConflict) {
		return nil, status.Error(codes.AlreadyExists, "reservation already exists with different parameters")
	}
	if errors.Is(err, domain.ErrInventoryUnavailable) {
		return nil, status.Error(codes.Unavailable, "inventory service unavailable")
	}
	if errors.Is(err, domain.ErrInventoryTimeout) {
		return nil, status.Error(codes.DeadlineExceeded, "inventory request timeout")
	}
	if errors.Is(err, domain.ErrUnsupportedOrderItems) {
		return nil, invalidArgument("items", "MVP supports exactly one order item")
	}
	if errors.Is(err, domain.ErrInsufficientStock) {
		return nil, status.Error(codes.FailedPrecondition, "insufficient stock")
	}
	// Fallback для остальных необработанных ошибок
	if err != nil {
		return nil, status.Error(codes.Internal, "internal order-server error")
	}

	response := &order.CreateOrderResponse{
		Id: request.GetId(),
	}
	return response, nil
}

func (s *Server) WatchOrder(
	req *order.WatchOrderRequest,
	stream grpc.ServerStreamingServer[order.WatchOrderResponse],
) error {
	response := &order.WatchOrderResponse{
		Id:     req.Id,
		Status: "created",
	}
	if err := stream.Send(response); err != nil {
		return fmt.Errorf("streaming error: %w", err)
	}

	select {
	case <-stream.Context().Done():
		return stream.Context().Err()
	case <-time.After(5 * time.Second):
	}

	response = &order.WatchOrderResponse{
		Id:     req.Id,
		Status: "paid",
	}
	if err := stream.Send(response); err != nil {
		return fmt.Errorf("streaming error: %w", err)
	}

	return nil
}

func (s *Server) CalculateOrderTotal(
	stream grpc.ClientStreamingServer[
		order.OrderItemRequest,
		order.OrderTotalResponse],
) error {
	var total int64

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("streaming to order-server error: %w", err)
		}
		total += req.GetQuantity()
	}

	return stream.SendAndClose(&order.OrderTotalResponse{
		TotalQuantity: total,
	})
}

func invalidArgument(field, description string) error {
	st := status.New(codes.InvalidArgument, "invalid order")

	stWithDetails, detailsErr := st.WithDetails(
		&errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{
					Field:       field,
					Description: description,
				},
			},
		},
	)
	if detailsErr != nil {
		return st.Err()
	}

	return stWithDetails.Err()
}
