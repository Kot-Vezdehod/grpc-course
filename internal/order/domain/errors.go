package domain

import "errors"

// Ошибки предметной области заказов.
var (
	// ErrOrderAlreadyExists возвращается при попытке сохранить заказ с занятым ID.
	ErrOrderAlreadyExists = errors.New("order already exists")
	// ErrOrderIDRequired возвращается, когда ID заказа не задан.
	ErrOrderIDRequired = errors.New("order ID is required")
	// ErrProductIDRequired возвращается, когда ID товара не задан.
	ErrProductIDRequired = errors.New("product ID is required")
	// ErrQuantityMustBePositive возвращается, когда количество товара не положительное.
	ErrQuantityMustBePositive = errors.New("quantity must be positive")

	// ErrCurrencyRequired возвращается, когда валюта заказа не задана.
	ErrCurrencyRequired = errors.New("currency is required")
	// ErrItemsRequired возвращается, когда заказ не содержит позиций.
	ErrItemsRequired = errors.New("items are required")
	// ErrInvalidPrice возвращается, когда цена позиции отрицательна.
	ErrInvalidPrice = errors.New("invalid price")
	// ErrPaymentUnavailable возвращается, когда сервис платежей недоступен.
	ErrPaymentUnavailable = errors.New("payment is unavailable")
	// ErrPaymentTimeout возвращается, когда истекло время ожидания ответа сервиса платежей.
	ErrPaymentTimeout = errors.New("payment timeout")
	// ErrInsufficientStock возвращается, когда товара недостаточно для заказа.
	ErrInsufficientStock = errors.New("insufficient stock")
	// ErrReservationConflict возвращается при конфликте резервирования товара.
	ErrReservationConflict = errors.New("reservation conflict")
	// ErrInventoryUnavailable возвращается, когда сервис остатков недоступен.
	ErrInventoryUnavailable = errors.New("inventory unavailable")
	// ErrInventoryTimeout возвращается, когда истекло время ожидания ответа сервиса остатков.
	ErrInventoryTimeout      = errors.New("inventory timeout")
	ErrUnsupportedOrderItems = errors.New("MVP supports exactly one order item")
)
