package grpc

import (
	"context"
	"errors"
	pb "grpc-course/gen/inventory"
	"grpc-course/internal/inventory/application"
	"grpc-course/internal/inventory/domain"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type InventoryServer struct {
	pb.UnimplementedInventoryServiceServer
	service *application.InventoryService
}

func NewInventoryServer(service *application.InventoryService) *InventoryServer {
	return &InventoryServer{
		service: service,
	}
}

func (s *InventoryServer) CreateInventory(ctx context.Context, req *pb.CreateInventoryRequest) (*pb.Inventory, error) {
	inv, err := domain.NewInventory(
		req.ProductId,
		req.ProductName,
		req.CategoryId,
		req.Stock,
		req.UnitPriceMinor,
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrProductIDRequired),
			errors.Is(err, domain.ErrProductNameRequired),
			errors.Is(err, domain.ErrCategoryIDRequired),
			errors.Is(err, domain.ErrStockNegative),
			errors.Is(err, domain.ErrUnitPriceMinorMustBePositive):

			return nil, status.Error(codes.InvalidArgument, err.Error())

		default:
			return nil, status.Error(codes.InvalidArgument, err.Error())

		}
	}

	if err := s.service.Create(ctx, inv); err != nil {
		switch {
		case errors.Is(err, domain.ErrInventoryAlreadyExists):
			return nil, status.Error(codes.AlreadyExists, "inventory already exists")

		default:
			return nil, status.Error(codes.Internal, "internal order-server error")
		}
	}

	return &pb.Inventory{
		ProductId:      inv.ProductID,
		ProductName:    inv.ProductName,
		CategoryId:     inv.CategoryID,
		Stock:          inv.Stock,
		UnitPriceMinor: inv.UnitPriceMinor,
	}, nil
}

func (s *InventoryServer) GetInventoryByID(
	ctx context.Context,
	req *pb.InventoryItemRequest,
) (*pb.Inventory, error) {

	if req == nil || strings.TrimSpace(req.ProductId) == "" {
		return nil, status.Error(codes.InvalidArgument, "product id is required")
	}

	got, err := s.service.GetByID(ctx, req.ProductId)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInventoryDoesNotExist):
			return nil, status.Error(codes.NotFound, "inventory not found")

		default:
			return nil, status.Error(codes.Internal, "internal order-server error")
		}
	}

	return &pb.Inventory{
		ProductId:      got.ProductID,
		ProductName:    got.ProductName,
		CategoryId:     got.CategoryID,
		Stock:          got.Stock,
		UnitPriceMinor: got.UnitPriceMinor,
	}, nil
}

func (s *InventoryServer) ReserveStock(
	ctx context.Context,
	req *pb.ReserveStockRequest,
) (*pb.ReserveStockResponse, error) {
	if req == nil ||
		strings.TrimSpace(req.ProductId) == "" ||
		strings.TrimSpace(req.OrderId) == "" ||
		req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid reservation parameters")
	}

	err := s.service.ReserveStock(
		ctx,
		req.OrderId,
		req.ProductId,
		req.Quantity,
	)
	if errors.Is(err, domain.ErrInsufficientStock) {
		return nil, status.Error(codes.FailedPrecondition, "insufficient stock")
	}
	if errors.Is(err, domain.ErrReservationConflict) {
		return nil, status.Error(codes.AlreadyExists, "reservation already exists with different parameters")
	}

	if err != nil {
		return &pb.ReserveStockResponse{
			Reserved: false,
		}, status.Error(codes.Internal, "internal inventory-server error")
	}

	return &pb.ReserveStockResponse{
		Reserved: true,
	}, nil
}
