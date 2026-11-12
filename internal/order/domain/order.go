package domain

import (
	"context"
)

// Order представляет заказ в предметной области.
type Order struct {
	ID       string
	Currency string
	Items    []OrderItem
}

type OrderItem struct {
	ProductID      string
	Quantity       int64
	UnitPriceMinor int64
}

// OrderRepository определяет порт для сохранения заказов.
type OrderRepository interface {
	CreateOrder(ctx context.Context, order *Order) error
}

func (o *Order) TotalAmountMinor() int64 {
	var total int64
	for _, item := range o.Items {
		total += int64(item.Quantity) * item.UnitPriceMinor
	}

	return total
}

// NewOrder создаёт валидный заказ из его идентификатора, валюты и позиций.
func NewOrder(id, currency string, items []OrderItem) (*Order, error) {
	if id == "" {
		return nil, ErrOrderIDRequired
	}
	if currency == "" {
		return nil, ErrCurrencyRequired
	}
	if len(items) == 0 {
		return nil, ErrItemsRequired
	}

	for _, item := range items {
		if item.Quantity < 1 {
			return nil, ErrQuantityMustBePositive
		}
		if item.UnitPriceMinor < 0 {
			return nil, ErrInvalidPrice
		}
		if item.ProductID == "" {
			return nil, ErrProductIDRequired
		}
	}

	return &Order{
		ID:       id,
		Items:    items,
		Currency: currency,
	}, nil
}
