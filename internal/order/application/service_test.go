package application

import (
	"context"
	"errors"
	"grpc-course/internal/order/domain"
	"grpc-course/internal/order/mocks"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// These dependencies keep the original repository/payment scenarios isolated.
type availableInventoryStub struct{}

func (availableInventoryStub) ReserveStock(context.Context, string, string, int64) error {
	return nil
}

type orderMetricsStub struct{}

func (orderMetricsStub) OrderCreated() {}

func TestOrderService_CreateOrder_HappyPath(t *testing.T) {
	repo := mocks.NewOrderRepository(t)
	payment := mocks.NewPaymentGateway(t)

	items := []domain.OrderItem{
		{
			ProductID:      "product-001",
			Quantity:       2,
			UnitPriceMinor: 200,
		},
	}

	repo.On("CreateOrder", mock.Anything, mock.MatchedBy(func(order *domain.Order) bool {
		return order.ID == "order-001" &&
			order.Currency == "USD" &&
			len(order.Items) == 1 &&
			order.Items[0].ProductID == "product-001" &&
			order.Items[0].UnitPriceMinor == 200 &&
			order.Items[0].Quantity == 2
	})).
		Return(nil)
	payment.On("CreatePayment", mock.Anything, "order-001", int64(400), "USD").
		Return(nil)

	service := NewOrderService(repo, payment, availableInventoryStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)), orderMetricsStub{})
	err := service.CreateOrder(
		context.Background(),
		"order-001",
		"USD",
		items,
	)

	require.NoError(t, err)
}

func prepareForOrderService(t *testing.T) (
	*mocks.OrderRepository,
	*mocks.PaymentGateway,
	[]domain.OrderItem,
) {
	t.Helper()

	repo := mocks.NewOrderRepository(t)
	payment := mocks.NewPaymentGateway(t)

	items := []domain.OrderItem{
		{
			ProductID:      "product-001",
			Quantity:       2,
			UnitPriceMinor: 250,
		},
	}

	return repo, payment, items
}

func TestOrderService_CreateOrder_FailsWhenRepositoryFails(t *testing.T) {
	repo := mocks.NewOrderRepository(t)
	payment := mocks.NewPaymentGateway(t)

	items := []domain.OrderItem{
		{
			ProductID:      "product-001",
			Quantity:       2,
			UnitPriceMinor: 250,
		},
	}

	repoErr := errors.New("repository failure")
	repo.On("CreateOrder", mock.Anything, mock.Anything).
		Return(repoErr)

	service := NewOrderService(repo, payment, availableInventoryStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)), orderMetricsStub{})
	err := service.CreateOrder(
		context.Background(),
		"order-001",
		"USD",
		items,
	)
	require.ErrorIs(t, err, repoErr)

	payment.AssertNotCalled(t,
		"CreatePayment",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestOrderService_CreateOrder_FailsWhenPaymentFails(t *testing.T) {
	repo, payment, items := prepareForOrderService(t)
	payErr := errors.New("payment failure")

	repo.On("CreateOrder", mock.Anything, mock.Anything).
		Return(nil)
	payment.On("CreatePayment", mock.Anything, "order-001", int64(500), "USD").
		Return(payErr)

	service := NewOrderService(repo, payment, availableInventoryStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)), orderMetricsStub{})
	err := service.CreateOrder(
		context.Background(),
		"order-001",
		"USD",
		items,
	)
	require.ErrorIs(t, err, payErr)
}

func TestOrderService_CreateOrder_InvalidArgs(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		currency string
		items    []domain.OrderItem
		wantErr  error
	}{
		{
			name:     "empty id",
			id:       "",
			currency: "USD",
			items: []domain.OrderItem{
				{
					ProductID:      "product-001",
					Quantity:       2,
					UnitPriceMinor: 250,
				},
			},
			wantErr: domain.ErrOrderIDRequired,
		},
		{
			name:     "empty currency",
			id:       "order-001",
			currency: "",
			items: []domain.OrderItem{
				{
					ProductID:      "product-001",
					Quantity:       2,
					UnitPriceMinor: 250,
				},
			},
			wantErr: domain.ErrCurrencyRequired,
		},
		{
			name:     "empty items",
			id:       "order-001",
			currency: "USD",
			items:    nil,
			wantErr:  domain.ErrItemsRequired,
		},
		{
			name:     "negative quantity",
			id:       "order-001",
			currency: "USD",
			items: []domain.OrderItem{
				{
					ProductID:      "product-001",
					Quantity:       -2,
					UnitPriceMinor: 250,
				},
			},
			wantErr: domain.ErrQuantityMustBePositive,
		},
		{
			name:     "negative unitPrice",
			id:       "order-001",
			currency: "USD",
			items: []domain.OrderItem{
				{
					ProductID:      "product-001",
					Quantity:       2,
					UnitPriceMinor: -250,
				},
			},
			wantErr: domain.ErrInvalidPrice,
		},
		{
			name:     "empty product ID",
			id:       "order-001",
			currency: "USD",
			items: []domain.OrderItem{
				{
					ProductID:      "",
					Quantity:       2,
					UnitPriceMinor: 250,
				},
			},
			wantErr: domain.ErrProductIDRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, payment, _ := prepareForOrderService(t)

			service := NewOrderService(repo, payment, availableInventoryStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)), orderMetricsStub{})

			err := service.CreateOrder(
				context.Background(),
				tt.id,
				tt.currency,
				tt.items,
			)

			require.ErrorIs(t, err, tt.wantErr)

			repo.AssertNotCalled(t,
				"CreateOrder",
				mock.Anything,
				mock.Anything,
			)

			payment.AssertNotCalled(t,
				"CreatePayment",
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
			)
		})
	}
}
