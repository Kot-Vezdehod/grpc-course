package http

import (
	"encoding/json/v2"
	inventorypb "grpc-course/gen/inventory"
	"log"
	"net/http"

	"google.golang.org/grpc/status"
)

func (h *Handler) CreateInventoryHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateInventoryRequest

	if err := json.UnmarshalRead(
		r.Body,
		&req,
		json.RejectUnknownMembers(true),
	); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"InvalidArgument",
			"invalid JSON",
		)
		return
	}

	grpcRequest := &inventorypb.CreateInventoryRequest{
		ProductId:      req.ProductID,
		ProductName:    req.ProductName,
		CategoryId:     req.CategoryID,
		Stock:          req.Stock,
		UnitPriceMinor: req.UnitPriceMinor,
	}

	grpcResponse, err := h.inventoryClient.CreateInventory(
		r.Context(),
		grpcRequest,
	)
	if err != nil {
		writeJSONError(
			w,
			grpcToHTTPStatus(err),
			status.Code(err).String(),
			"inventory creation failed",
		)
		return
	}

	response := CreateInventoryResponse{
		ProductID:      grpcResponse.ProductId,
		ProductName:    grpcResponse.ProductName,
		CategoryID:     grpcResponse.CategoryId,
		Stock:          grpcResponse.Stock,
		UnitPriceMinor: grpcResponse.UnitPriceMinor,
	}

	data, err := json.Marshal(response)
	if err != nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"Internal",
			"failed to encode response",
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write(data); err != nil {
		log.Printf("failed to write response: %v", err)
	}

}

func (h *Handler) GetInventoryHandler(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")

	grpcRequest := &inventorypb.InventoryItemRequest{
		ProductId: productID,
	}

	grpcResponse, err := h.inventoryClient.GetInventoryByID(
		r.Context(),
		grpcRequest,
	)
	if err != nil {
		writeJSONError(
			w,
			grpcToHTTPStatus(err),
			status.Code(err).String(),
			"failed to retrieve inventory",
		)
		return
	}

	response := CreateInventoryResponse{
		ProductID:      grpcResponse.GetProductId(),
		ProductName:    grpcResponse.GetProductName(),
		CategoryID:     grpcResponse.GetCategoryId(),
		Stock:          grpcResponse.GetStock(),
		UnitPriceMinor: grpcResponse.GetUnitPriceMinor(),
	}

	data, err := json.Marshal(response)
	if err != nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"Internal",
			"failed to encode response",
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(data); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
