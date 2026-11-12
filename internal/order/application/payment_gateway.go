package application

import "context"

type PaymentGateway interface {
	CreatePayment(
		ctx context.Context,
		orderID string,
		amountMinor int64,
		currency string,
	) error
}
