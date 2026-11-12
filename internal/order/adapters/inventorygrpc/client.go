package inventorygrpc

import (
	"context"
	"fmt"
	inventorypb "grpc-course/gen/inventory"
	"grpc-course/internal/order/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Client struct {
	client inventorypb.InventoryServiceClient
}

func NewClient(client inventorypb.InventoryServiceClient) *Client {
	return &Client{client: client}
}

func (c *Client) ReserveStock(
	ctx context.Context,
	orderID string,
	productID string,
	quantity int64,
) error {
	req := &inventorypb.ReserveStockRequest{
		OrderId:   orderID,
		ProductId: productID,
		Quantity:  quantity,
	}

	resp, err := c.client.ReserveStock(ctx, req)
	if err != nil {
		switch status.Code(err) {
		case codes.FailedPrecondition:
			return domain.ErrInsufficientStock

		case codes.AlreadyExists:
			return domain.ErrReservationConflict

		case codes.Unavailable:
			return domain.ErrInventoryUnavailable

		case codes.DeadlineExceeded:
			return domain.ErrInventoryTimeout

		default:
			return fmt.Errorf("inventory.ReserveStock: %w", err)
		}
	}

	if resp == nil || !resp.Reserved {
		return fmt.Errorf("inventory.ReserveStock: reservation not confirmed")
	}

	return nil
}
