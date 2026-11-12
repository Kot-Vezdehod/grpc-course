package http

type CreateOrderRequest struct {
	ID       string         `json:"id"`
	Currency string         `json:"currency"`
	Items    []OrderItemDTO `json:"items"`
}

type OrderItemDTO struct {
	ProductID      string `json:"product_id"`
	Quantity       int64  `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
}

type CreateOrderResponse struct {
	ID string `json:"id"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreateInventoryRequest struct {
	ProductID      string `json:"product_id"`
	ProductName    string `json:"product_name"`
	CategoryID     string `json:"category_id"`
	Stock          int64  `json:"stock"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
}

type CreateInventoryResponse struct {
	ProductID      string `json:"product_id"`
	ProductName    string `json:"product_name"`
	CategoryID     string `json:"category_id"`
	Stock          int64  `json:"stock"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
}
