package domain

import (
	"errors"
	"testing"
)

func TestNewPaymentValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		id       string
		order    string
		amount   int64
		currency string
		wantErr  error
	}{
		{name: "payment ID is required", order: "order-001", amount: 1_000, currency: "RUB", wantErr: ErrPaymentIDRequired},
		{name: "order ID is required", id: "payment-001", amount: 1_000, currency: "RUB", wantErr: ErrPaymentOrderIDRequired},
		{name: "negative amount", id: "payment-001", order: "order-001", amount: -1, currency: "RUB", wantErr: ErrAmountMinorNegative},
		{name: "currency is required", id: "payment-001", order: "order-001", amount: 1_000, wantErr: ErrPaymentCurrencyRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewPayment(tt.id, tt.order, tt.amount, tt.currency)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewPayment() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewPaymentSuccess(t *testing.T) {
	t.Parallel()

	payment, err := NewPayment("payment-001", "order-001", 1_000, "RUB")
	if err != nil {
		t.Fatalf("NewPayment() error = %v, want nil", err)
	}
	if payment.Currency != "RUB" {
		t.Errorf("payment currency = %q, want %q", payment.Currency, "RUB")
	}
}
