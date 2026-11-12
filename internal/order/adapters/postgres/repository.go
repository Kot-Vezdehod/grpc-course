package postgres

import (
	"context"
	"database/sql"
	"errors"
	"grpc-course/internal/order/domain"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
)

// OrderRepository сохраняет заказы в базе данных PostgreSQL.
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository создаёт PostgreSQL-репозиторий с указанным подключением.
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{
		db: db,
	}
}

// CreateOrder сохраняет заказ в базе данных.
func (r *OrderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query, args, err := squirrel.Insert("orders").
		Columns("id", "currency").
		Values(order.ID, order.Currency).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, query, args...)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrOrderAlreadyExists
	}
	if err != nil {
		return err
	}

	for _, item := range order.Items {
		query, args, err := squirrel.Insert("order_items").
			Columns("order_id", "product_id", "quantity", "unit_price_minor").
			Values(order.ID, item.ProductID, item.Quantity, item.UnitPriceMinor).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
