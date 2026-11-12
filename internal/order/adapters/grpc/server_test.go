package grpc

import (
	"context"
	"errors"
	"testing"

	"grpc-course/gen/order"
	"grpc-course/internal/order/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// orderCreatorStub — подставная реализация операции создания заказа для gRPC-адаптера.
type orderCreatorStub struct {
	createOrderErr error
	id             string
	currency       string
	items          []domain.OrderItem
}

// CreateOrder запоминает аргументы вызова и возвращает заданную ошибку.
func (s *orderCreatorStub) CreateOrder(_ context.Context, id, currency string, items []domain.OrderItem) error {
	s.id = id
	s.currency = currency
	s.items = items
	return s.createOrderErr
}

// TestServerCreateOrderSuccess проверяет успешный RPC-ответ и передачу полей в сервис приложения.
func TestServerCreateOrderSuccess(t *testing.T) {
	t.Parallel()

	service := &orderCreatorStub{}
	server := newServer(service)
	request := &order.CreateOrderRequest{
		Id:       "order-001",
		Currency: "RUB",
		Items: []*order.OrderItem{
			{ProductId: "product-001", Quantity: 2, UnitPriceMinor: 1_500},
			{ProductId: "product-002", Quantity: 1, UnitPriceMinor: 2_000},
		},
	}

	response, err := server.CreateOrder(context.Background(), request)
	if err != nil {
		t.Fatalf("CreateOrder() error = %v, want nil", err)
	}
	if response.GetId() != request.GetId() {
		t.Errorf("response ID = %q, want %q", response.GetId(), request.GetId())
	}
	if service.id != request.GetId() {
		t.Errorf("service order ID = %q, want %q", service.id, request.GetId())
	}
	if service.currency != request.GetCurrency() {
		t.Errorf("service currency = %q, want %q", service.currency, request.GetCurrency())
	}
	wantItems := []domain.OrderItem{
		{ProductID: "product-001", Quantity: 2, UnitPriceMinor: 1_500},
		{ProductID: "product-002", Quantity: 1, UnitPriceMinor: 2_000},
	}
	if len(service.items) != len(wantItems) {
		t.Fatalf("service items count = %d, want %d", len(service.items), len(wantItems))
	}
	for i, want := range wantItems {
		if got := service.items[i]; got != want {
			t.Errorf("service item %d = %#v, want %#v", i, got, want)
		}
	}
}

// TestServerCreateOrderReturnsApplicationError проверяет возврат ошибки сервиса приложения.
func TestServerCreateOrderReturnsApplicationError(t *testing.T) {
	t.Parallel()

	applicationErr := errors.New("cannot create order")
	server := newServer(&orderCreatorStub{createOrderErr: applicationErr})
	request := &order.CreateOrderRequest{
		Id:       "order-001",
		Currency: "RUB",
		Items: []*order.OrderItem{
			{ProductId: "product-001", Quantity: 2, UnitPriceMinor: 1_500},
		},
	}

	response, err := server.CreateOrder(context.Background(), request)
	if response != nil {
		t.Errorf("CreateOrder() response = %v, want nil", response)
	}
	if status.Code(err) != codes.Internal {
		t.Fatalf(
			"CreateOrder() code = %v, want %v",
			status.Code(err),
			codes.Internal,
		)
	}
}
