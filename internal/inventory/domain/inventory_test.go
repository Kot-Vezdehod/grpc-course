package domain

import (
	"errors"
	"testing"
)

func TestNewInventory(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name            string
		productID       string
		productName     string
		categoryID      string
		stock           int64
		unitPriceMinor  int64
		wantErr         error
		wantProductID   string
		wantProductName string
		wantCategoryID  string
	}{
		{
			name:           "productID is empty",
			productID:      "",
			productName:    "iPhone 20",
			categoryID:     "001",
			stock:          1000,
			unitPriceMinor: 1299,
			wantErr:        ErrProductIDRequired,
		},
		{
			name:           "productName is empty",
			productID:      "a-001",
			productName:    "",
			categoryID:     "001",
			stock:          1000,
			unitPriceMinor: 1299,
			wantErr:        ErrProductNameRequired,
		},
		{
			name:           "categoryID is empty",
			productID:      "a-001",
			productName:    "iPhone 20",
			categoryID:     "",
			stock:          1000,
			unitPriceMinor: 1299,
			wantErr:        ErrCategoryIDRequired,
		},
		{
			name:           "stock is negative",
			productID:      "a-001",
			productName:    "iPhone 20",
			categoryID:     "001",
			stock:          -1000,
			unitPriceMinor: 1299,
			wantErr:        ErrStockNegative,
		},
		{
			name:           "unitPriceMinor is not positive",
			productID:      "a-001",
			productName:    "iPhone 20",
			categoryID:     "001",
			stock:          1000,
			unitPriceMinor: -1299,
			wantErr:        ErrUnitPriceMinorMustBePositive,
		},
		{
			name:            "spaces are trimmed",
			productID:       "a-001   ",
			productName:     "  iPhone 20 ",
			categoryID:      "  001 ",
			stock:           1000,
			unitPriceMinor:  1299,
			wantErr:         nil,
			wantProductID:   "a-001",
			wantProductName: "iPhone 20",
			wantCategoryID:  "001",
		},
		{
			name:            "zero stock",
			productID:       "a-001",
			productName:     "iPhone 20",
			categoryID:      "001",
			stock:           0,
			unitPriceMinor:  1299,
			wantErr:         nil,
			wantProductID:   "a-001",
			wantProductName: "iPhone 20",
			wantCategoryID:  "001",
		},
		{
			name:           "zero price",
			productID:      "a-001",
			productName:    "iPhone 20",
			categoryID:     "001",
			stock:          10,
			unitPriceMinor: 0,
			wantErr:        ErrUnitPriceMinorMustBePositive,
		},
		{
			name:           "spaces as productID",
			productID:      "      ",
			productName:    "iPhone 20",
			categoryID:     "001",
			stock:          1000,
			unitPriceMinor: 1299,
			wantErr:        ErrProductIDRequired,
		},
		{
			name:            "productID normalized to lowercase",
			productID:       "A-001",
			productName:     "iPhone 20",
			categoryID:      "001",
			stock:           10,
			unitPriceMinor:  1299,
			wantProductID:   "a-001",
			wantProductName: "iPhone 20",
			wantCategoryID:  "001",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NewInventory(
				tc.productID,
				tc.productName,
				tc.categoryID,
				tc.stock,
				tc.unitPriceMinor,
			)

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("NewInventory() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err == nil && got == nil {
				t.Fatal("NewInventory() returned nil inventory without error")
			}
			if err == nil {
				if got.ProductID != tc.wantProductID {
					t.Errorf("NewInventory() ProductID = %q, want %q", got.ProductID, tc.wantProductID)
				}
				if got.ProductName != tc.wantProductName {
					t.Errorf("NewInventory() ProductName = %q, want %q", got.ProductName, tc.wantProductName)
				}
				if got.CategoryID != tc.wantCategoryID {
					t.Errorf("NewInventory() CategoryID = %q, want %q", got.CategoryID, tc.wantCategoryID)
				}
			}
		})
	}
}
