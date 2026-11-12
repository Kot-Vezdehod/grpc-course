package e2e

import (
	"context"
	"database/sql"
	"fmt"
	orderpb "grpc-course/gen/order"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestCreateOrder_EndToEnd(t *testing.T) {
	if os.Getenv("RUN_E2E") != "1" {
		t.Skip("set RUN_E2E=1 to run against a dedicated test environment")
	}
	orderDSN := strings.TrimSpace(os.Getenv("E2E_ORDER_DATABASE_URL"))
	paymentDSN := strings.TrimSpace(os.Getenv("E2E_PAYMENT_DATABASE_URL"))
	if orderDSN == "" || paymentDSN == "" {
		t.Fatal("E2E_ORDER_DATABASE_URL and E2E_PAYMENT_DATABASE_URL are required")
	}
	orderAddress := strings.TrimSpace(os.Getenv("E2E_ORDER_SERVICE_ADDR"))
	if orderAddress == "" {
		orderAddress = "localhost:50051"
	}
	conn, err := grpc.NewClient(
		orderAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("create gRPC connection: %v", err)
	}
	defer conn.Close()

	client := orderpb.NewOrderServiceClient(conn)

	orderID := fmt.Sprintf("e2e-%d", time.Now().UnixNano())
	currency := "USD"

	item1 := &orderpb.OrderItem{
		ProductId:      "product-001",
		Quantity:       2,
		UnitPriceMinor: 500,
	}
	request := &orderpb.CreateOrderRequest{
		Id:       orderID,
		Currency: currency,
		Items:    []*orderpb.OrderItem{item1},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	paymentDB, err := sql.Open(
		"pgx",
		paymentDSN,
	)
	if err != nil {
		t.Fatalf("open payment database: %v", err)
	}

	orderDB, err := sql.Open(
		"pgx",
		orderDSN,
	)
	if err != nil {
		t.Fatalf("open order database: %v", err)
	}

	t.Cleanup(func() {
		orderDB.Close()
		paymentDB.Close()
	})

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		_, err := paymentDB.ExecContext(
			cleanupCtx,
			`DELETE FROM payments WHERE order_id = $1`,
			orderID,
		)
		if err != nil {
			t.Errorf("cleanup payment: %v", err)
		}

		_, err = orderDB.ExecContext(
			cleanupCtx,
			`DELETE FROM orders WHERE id = $1`,
			orderID,
		)
		if err != nil {
			t.Errorf("cleanup order: %v", err)
		}
	})

	resp, err := client.CreateOrder(ctx, request)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	if resp.GetId() != orderID {
		t.Fatalf("create order id mismatch: got %v, want %v", resp.GetId(), orderID)
	}

	var amountMinor int64
	var gotCurrency string

	err = paymentDB.QueryRowContext(
		ctx,
		`SELECT amount_minor, currency FROM payments
WHERE order_id = $1`, orderID,
	).Scan(&amountMinor, &gotCurrency)
	if err != nil {
		t.Fatalf("get payment amount: %v", err)
	}

	if gotCurrency != currency {
		t.Fatalf("got %v, want %v", gotCurrency, currency)
	}
	if amountMinor != 1000 {
		t.Fatalf("amountMinor %v, want 1000", amountMinor)
	}

	var orderCurrency string

	err = orderDB.QueryRowContext(
		ctx,
		`SELECT currency FROM orders WHERE id = $1`,
		orderID,
	).Scan(&orderCurrency)
	if err != nil {
		t.Fatalf("get order currency: %v", err)
	}

	if orderCurrency != currency {
		t.Fatalf("order currency mismatch: got %v, want %v", orderCurrency, currency)
	}

	var itemCount int

	err = orderDB.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM order_items WHERE order_id = $1`,
		orderID,
	).Scan(&itemCount)
	if err != nil {
		t.Fatalf("get item count: %v", err)
	}

	if itemCount != len(request.Items) {
		t.Fatalf("item count mismatch: got %v, want %v", itemCount, len(request.Items))
	}
}
