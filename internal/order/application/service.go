package application

import (
	"context"
	"errors"
	"fmt"
	"grpc-course/internal/order/domain"
	"log/slog"
	"time"
)

// OrderService создаёт заказы через репозиторий предметной области.
type OrderService struct {
	repo      domain.OrderRepository
	payment   PaymentGateway
	inventory InventoryClient
	logger    *slog.Logger
	metrics   OrderMetrics
}

type OrderMetrics interface {
	OrderCreated()
}

type InventoryClient interface {
	ReserveStock(
		ctx context.Context,
		orderID string,
		productID string,
		quantity int64,
	) error
}

// NewOrderService создаёт сервис заказов с указанным репозиторием.
func NewOrderService(
	repo domain.OrderRepository,
	payment PaymentGateway,
	inventory InventoryClient,
	logger *slog.Logger,
	metrics OrderMetrics,
) *OrderService {
	return &OrderService{
		repo:      repo,
		payment:   payment,
		inventory: inventory,
		logger:    logger,
		metrics:   metrics,
	}
}

// CreateOrder создаёт заказ и сохраняет его в репозитории.
func (s *OrderService) CreateOrder(
	ctx context.Context,
	id string,
	currency string,
	items []domain.OrderItem,
) error {
	order, err := domain.NewOrder(id, currency, items)
	if err != nil {
		return err
	}

	// временное ограничение из-за того, что inventory пока отрабатывает только одну позицию за раз
	if len(items) != 1 {
		return domain.ErrUnsupportedOrderItems
	}

	orderLogger := s.logger.With("order_id", order.ID)
	orderLogger.Info("order creation started")

	inventoryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := s.inventory.ReserveStock(
		inventoryCtx,
		order.ID,
		order.Items[0].ProductID,
		int64(order.Items[0].Quantity),
	); err != nil {
		if errors.Is(err, domain.ErrInsufficientStock) {
			orderLogger.Warn(
				"inventory stock insufficient",
				"product_id", order.Items[0].ProductID,
				"quantity", order.Items[0].Quantity,
			)
		}

		return fmt.Errorf("reserve inventory: %w", err)
	}

	orderLogger.Info(
		"inventory stock reserved",
		"product_id",
		order.Items[0].ProductID,
		"quantity",
		order.Items[0].Quantity,
	)

	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return err
	}

	orderLogger.Info("order created")

	paymentCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	amount := order.TotalAmountMinor()
	err = s.payment.CreatePayment(paymentCtx, order.ID, amount, order.Currency)
	if err != nil {
		return err
	}

	orderLogger.Info("payment created")

	s.metrics.OrderCreated()

	return nil
}
