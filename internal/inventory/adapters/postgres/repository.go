package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"grpc-course/internal/inventory/domain"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
)

type InventoryRepository struct {
	db *sql.DB
}

func NewInventoryRepository(db *sql.DB) *InventoryRepository {
	return &InventoryRepository{
		db: db,
	}
}

func (r *InventoryRepository) Create(
	ctx context.Context,
	inventory *domain.Inventory,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query, args, err := squirrel.Insert("inventory").
		Columns("product_id", "product_name", "category_id", "stock", "unit_price_minor").
		Values(inventory.ProductID, inventory.ProductName, inventory.CategoryID, inventory.Stock, inventory.UnitPriceMinor).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, query, args...)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrInventoryAlreadyExists
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *InventoryRepository) GetByID(ctx context.Context, productID string) (*domain.Inventory, error) {
	query, args, err := squirrel.Select("product_id", "product_name", "category_id", "stock", "unit_price_minor").
		From("inventory").
		Where(squirrel.Eq{"product_id": productID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	inventory := &domain.Inventory{}

	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&inventory.ProductID,
		&inventory.ProductName,
		&inventory.CategoryID,
		&inventory.Stock,
		&inventory.UnitPriceMinor,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrInventoryDoesNotExist
	}
	if err != nil {
		return nil, err
	}

	return inventory, nil
}

func (r *InventoryRepository) ReserveStock(
	ctx context.Context,
	orderID string,
	productID string,
	quantity int64,
) error {
	if strings.TrimSpace(productID) == "" ||
		strings.TrimSpace(orderID) == "" ||
		quantity <= 0 {
		return fmt.Errorf("reservation parameters required")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin reservation transaction: %w", err)
	}
	defer tx.Rollback()

	query, args, err := squirrel.
		Insert("inventory_reservations").
		Columns("order_id", "product_id", "quantity").
		Values(orderID, productID, quantity).
		Suffix("ON CONFLICT (order_id) DO NOTHING").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build reservation insert: %w", err)
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert reservation: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reservation rows affected: %w", err)
	}
	if affected == 0 {
		var existingProductID string
		var existingQuantity int64

		err = tx.QueryRowContext(
			ctx,
			`SELECT product_id, quantity
FROM inventory_reservations
WHERE order_id = $1`,
			orderID,
		).Scan(&existingProductID, &existingQuantity)

		if err != nil {
			return fmt.Errorf("get existing reservation: %w", err)
		}

		if existingProductID != productID || existingQuantity != quantity {
			return fmt.Errorf("%w: order %s has different parameters", domain.ErrReservationConflict, orderID)
		}
		return tx.Commit()
	}

	query, args, err = squirrel.
		Update("inventory").
		Set("stock", squirrel.Expr("stock - ?", quantity)).
		Where(squirrel.Eq{"product_id": productID}).
		Where(squirrel.GtOrEq{"stock": quantity}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build reservation update: %w", err)
	}

	result, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update stock reservation: %w", err)
	}

	affected, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update stock reservation rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrInsufficientStock
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reservation transaction: %w", err)
	}

	return nil
}
