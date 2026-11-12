package http

import (
	"encoding/json/v2"
	orderpb "grpc-course/gen/order"
	"log"
	"net/http"

	"google.golang.org/grpc/status"
)

func (h *Handler) CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest

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

	grpcRequest := &orderpb.CreateOrderRequest{
		Id:       req.ID,
		Currency: req.Currency,
	}

	for _, item := range req.Items {
		grpcRequest.Items = append(grpcRequest.Items, &orderpb.OrderItem{
			ProductId:      item.ProductID,
			Quantity:       item.Quantity,
			UnitPriceMinor: item.UnitPriceMinor,
		})
	}

	grpcResponse, err := h.orderClient.CreateOrder(r.Context(), grpcRequest)
	if err != nil {
		writeJSONError(
			w,
			grpcToHTTPStatus(err),
			status.Code(err).String(),
			"order creation failed",
		)
		return
	}

	response := CreateOrderResponse{
		ID: grpcResponse.GetId(),
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
