package http

import (
	"context"
	"encoding/json/v2"
	orderpb "grpc-course/gen/order"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHealthHandler(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	recorder := httptest.NewRecorder()
	handler(recorder, req)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}

	body := recorder.Body.String()
	if body != "OK" {
		t.Fatalf("expected body 'OK', got '%s'", body)
	}
}

type mockOrderClient struct {
	orderpb.OrderServiceClient

	receivedRequest *orderpb.CreateOrderRequest
	err             error
}

func (m *mockOrderClient) CreateOrder(
	ctx context.Context,
	req *orderpb.CreateOrderRequest,
	opts ...grpc.CallOption,
) (*orderpb.CreateOrderResponse, error) {
	m.receivedRequest = req

	if m.err != nil {
		return nil, m.err
	}

	return &orderpb.CreateOrderResponse{
		Id: req.GetId(),
	}, nil
}

func TestCreateOrderHandler(t *testing.T) {
	orderClient := &mockOrderClient{}
	handler := NewHandler(orderClient, nil)

	body := `{
    "id": "order-test-001",
    "currency": "RUB",
    "items": [
        {
            "product_id": "product-001",
            "quantity": 2,
            "unit_price_minor": 1500
        }
    ]
}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.CreateOrderHandler(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var response CreateOrderResponse

	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.ID != "order-test-001" {
		t.Fatalf("expected id 'order-test-001', got '%s'", response.ID)
	}

	if orderClient.receivedRequest == nil {
		t.Fatal("expected gRPC CreateOrder to be called")
	}
	if orderClient.receivedRequest.GetCurrency() != "RUB" {
		t.Errorf(
			"expected currency RUB, got %s",
			orderClient.receivedRequest.GetCurrency(),
		)
	}

	items := orderClient.receivedRequest.GetItems()

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].GetProductId() != "product-001" {
		t.Errorf("expected product-001, got %s", items[0].GetProductId())
	}
	if items[0].GetQuantity() != 2 {
		t.Errorf("expected quantity 2, got %d", items[0].GetQuantity())
	}
	if items[0].GetUnitPriceMinor() != 1500 {
		t.Errorf("expected unit price_minor 1500, got %d", items[0].GetUnitPriceMinor())
	}
}

func TestCreateOrderHandler_AlreadyExists(t *testing.T) {
	orderClient := &mockOrderClient{
		err: status.Error(codes.AlreadyExists, "order already exists"),
	}
	handler := NewHandler(orderClient, nil)

	body := `{
    "id": "order-test-001",
    "currency": "RUB",
    "items": [
        {
            "product_id": "product-001",
            "quantity": 2,
            "unit_price_minor": 1500
        }
    ]
}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.CreateOrderHandler(recorder, req)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}

	var response ErrorResponse

	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != "AlreadyExists" {
		t.Errorf("expected code AlreadyExists, got %s", response.Error.Code)
	}
	if response.Error.Message != "order creation failed" {
		t.Errorf("expected message 'order creation failed', got '%s'", response.Error.Message)
	}

}
