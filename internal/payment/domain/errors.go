package domain

import "errors"

// Ошибки предметной области платежей.
var (
	// ErrPaymentIDRequired возвращается, когда ID платежа не задан.
	ErrPaymentIDRequired = errors.New("payment ID is required")
	// ErrPaymentOrderIDRequired возвращается, когда ID заказа для платежа не задан.
	ErrPaymentOrderIDRequired = errors.New("payment order ID is required")
	// ErrAmountMinorNegative возвращается, когда сумма платежа в minor units отрицательна.
	ErrAmountMinorNegative = errors.New("payment amount minor must not be negative")
	// ErrPaymentCurrencyRequired возвращается, когда валюта платежа не задана.
	ErrPaymentCurrencyRequired = errors.New("payment currency is required")
	// ErrPaymentNotFound возвращается, когда платеж не найден.
	ErrPaymentNotFound = errors.New("payment not found")
	// ErrPaymentAlreadyExists возвращается, когда платеж уже существует.
	ErrPaymentAlreadyExists = errors.New("payment already exists")
)
