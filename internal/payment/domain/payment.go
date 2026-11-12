package domain

import "context"

type Payment struct {
	ID          string
	OrderID     string
	AmountMinor int64
	Currency    string
}

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment *Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
}

func NewPayment(id, orderID string, amountMinor int64, currency string) (*Payment, error) {
	if id == "" {
		return nil, ErrPaymentIDRequired
	}
	if orderID == "" {
		return nil, ErrPaymentOrderIDRequired
	}
	if amountMinor < 0 {
		return nil, ErrAmountMinorNegative
	}
	if currency == "" {
		return nil, ErrPaymentCurrencyRequired
	}

	return &Payment{
		ID:          id,
		OrderID:     orderID,
		AmountMinor: amountMinor,
		Currency:    currency,
	}, nil
}
