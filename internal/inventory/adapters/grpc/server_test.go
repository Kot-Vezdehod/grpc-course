package grpc

import (
	"context"
	pb "grpc-course/gen/inventory"
	"grpc-course/internal/inventory/application"
	"grpc-course/internal/inventory/domain"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type InventoryRepositoryStub struct {
	CreateFn func(
		ctx context.Context,
		inventory *domain.Inventory,
	) error

	GetByIDFn func(
		ctx context.Context,
		productID string,
	) (*domain.Inventory, error)
}

func (s *InventoryRepositoryStub) Create(
	ctx context.Context,
	inventory *domain.Inventory,
) error {
	return s.CreateFn(ctx, inventory)
}

func (s *InventoryRepositoryStub) GetByID(
	ctx context.Context,
	productID string,
) (*domain.Inventory, error) {
	return s.GetByIDFn(ctx, productID)
}

func (s *InventoryRepositoryStub) ReserveStock(context.Context, string, string, int64) error {
	panic("unexpected ReserveStock call")
}

func TestGetInventoryByID_NotFound(t *testing.T) {
	stub := &InventoryRepositoryStub{
		GetByIDFn: func(
			ctx context.Context,
			productID string,
		) (*domain.Inventory, error) {
			return nil, domain.ErrInventoryDoesNotExist
		},
	}

	service := application.NewInventoryService(stub, slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := NewInventoryServer(service)
	productID := "productID"
	req := &pb.InventoryItemRequest{
		ProductId: productID,
	}

	got, err := server.GetInventoryByID(
		context.Background(),
		req,
	)
	require.Nil(t, got)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestCreateInventory_AlreadyExists(t *testing.T) {
	stub := &InventoryRepositoryStub{
		CreateFn: func(
			ctx context.Context,
			inventory *domain.Inventory,
		) error {
			return domain.ErrInventoryAlreadyExists
		},
	}

	service := application.NewInventoryService(stub, slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := NewInventoryServer(service)
	req := &pb.CreateInventoryRequest{
		ProductId:      "productID",
		ProductName:    "productName",
		CategoryId:     "categoryID",
		Stock:          100,
		UnitPriceMinor: 1000,
	}

	got, err := server.CreateInventory(context.Background(), req)
	require.Nil(t, got)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}
