package application

import (
	"context"
	"errors"
	"fmt"
	"grpc-course/internal/inventory/domain"
	"log/slog"
)

type InventoryRepository interface {
	Create(ctx context.Context, inventory *domain.Inventory) error
	GetByID(ctx context.Context, productID string) (*domain.Inventory, error)
	ReserveStock(
		ctx context.Context,
		orderID string,
		productID string,
		quantity int64,
	) error
}

type InventoryService struct {
	repository InventoryRepository
	logger     *slog.Logger
}

func NewInventoryService(repository InventoryRepository, logger *slog.Logger) *InventoryService {
	return &InventoryService{
		repository: repository,
		logger:     logger,
	}
}

func (i *InventoryService) Create(ctx context.Context, inventory *domain.Inventory) error {
	err := i.repository.Create(ctx, inventory)
	if err != nil {
		return fmt.Errorf("create inventory item error: %w", err)
	}

	return nil
}

func (i *InventoryService) GetByID(ctx context.Context, productID string) (*domain.Inventory, error) {
	inventory, err := i.repository.GetByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("get inventory item error: %w", err)
	}

	return inventory, nil
}

func (i *InventoryService) ReserveStock(
	ctx context.Context,
	orderID string,
	productID string,
	quantity int64,
) error {
	err := i.repository.ReserveStock(ctx, orderID, productID, quantity)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientStock) {
			i.logger.Warn(
				"inventory stock insufficient",
				"order_id", orderID,
				"product_id", productID,
				"quantity", quantity,
			)
		}

		return fmt.Errorf("reserve stock error: %w", err)
	}

	i.logger.Info(
		"inventory stock reserved",
		"order_id", orderID,
		"product_id", productID,
		"quantity", quantity,
	)

	return nil
}
