package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc(
		"POST /api/v1/orders",
		h.CreateOrderHandler,
	)

	mux.HandleFunc(
		"POST /api/v1/inventory",
		h.CreateInventoryHandler,
	)

	mux.HandleFunc(
		"GET /api/v1/inventory/{id}",
		h.GetInventoryHandler,
	)
}
