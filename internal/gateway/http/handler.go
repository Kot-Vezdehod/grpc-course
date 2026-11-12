package http

import (
	inventorypb "grpc-course/gen/inventory"
	orderpb "grpc-course/gen/order"
)

type Handler struct {
	orderClient     orderpb.OrderServiceClient
	inventoryClient inventorypb.InventoryServiceClient
	//paymentClient paymentpb.PaymentServiceClient
}

func NewHandler(
	orderClient orderpb.OrderServiceClient,
	inventoryClient inventorypb.InventoryServiceClient,
) *Handler {
	return &Handler{
		orderClient:     orderClient,
		inventoryClient: inventoryClient,
	}
}
