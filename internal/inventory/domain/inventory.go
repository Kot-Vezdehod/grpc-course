package domain

import (
	"strings"
)

type Inventory struct {
	ProductID      string
	ProductName    string
	CategoryID     string
	Stock          int64
	UnitPriceMinor int64
}

func NewInventory(
	productID string,
	productName string,
	categoryID string,
	stock int64,
	unitPriceMinor int64,
) (*Inventory, error) {
	productID = strings.TrimSpace(productID)
	productID = strings.ToLower(productID)

	productName = strings.TrimSpace(productName)

	categoryID = strings.TrimSpace(categoryID)
	categoryID = strings.ToLower(categoryID)

	if productID == "" {
		return nil, ErrProductIDRequired
	}
	if productName == "" {
		return nil, ErrProductNameRequired
	}
	if categoryID == "" {
		return nil, ErrCategoryIDRequired
	}
	if stock < 0 {
		return nil, ErrStockNegative
	}
	if unitPriceMinor <= 0 {
		return nil, ErrUnitPriceMinorMustBePositive
	}

	return &Inventory{
		ProductID:      productID,
		ProductName:    productName,
		CategoryID:     categoryID,
		Stock:          stock,
		UnitPriceMinor: unitPriceMinor,
	}, nil
}
