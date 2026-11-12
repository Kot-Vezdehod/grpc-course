package domain

import "errors"

var (
	// ErrProductIDRequired возвращается, когда ID товара не задан.
	ErrProductIDRequired = errors.New("productID is required")
	// ErrProductNameRequired возвращается, когда название товара не задано.
	ErrProductNameRequired = errors.New("productName is required")
	// ErrCategoryIDRequired возвращается, когда ID категории не задан.
	ErrCategoryIDRequired = errors.New("categoryID is required")
	// ErrStockNegative возвращается, когда остаток товара отрицателен.
	ErrStockNegative = errors.New("stock can't be negative")
	// ErrUnitPriceMinorMustBePositive возвращается, когда цена товара не положительна.
	ErrUnitPriceMinorMustBePositive = errors.New("unitPriceMinor must be positive")
	// ErrInventoryAlreadyExists возвращается, когда запись об остатке товара уже существует.
	ErrInventoryAlreadyExists = errors.New("inventory already exists")
	// ErrInventoryDoesNotExist возвращается, когда запись об остатке товара не найдена.
	ErrInventoryDoesNotExist = errors.New("inventory does not exist")
	// ErrInsufficientStock возвращается, когда остатка товара недостаточно для резервирования.
	ErrInsufficientStock = errors.New("insufficient stock quantity")
	// ErrReservationConflict возвращается, когда заказ уже зарезервирован с другими параметрами.
	ErrReservationConflict = errors.New("reservation conflict")
)
