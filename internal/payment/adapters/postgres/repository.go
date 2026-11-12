package postgres

import (
	"context"
	"database/sql"
	"errors"
	"grpc-course/internal/payment/domain"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{
		db: db,
	}
}

func (r *PaymentRepository) CreatePayment(
	ctx context.Context,
	payment *domain.Payment,
) error {
	query, args, err := squirrel.Insert("payments").
		Columns("id", "order_id", "amount_minor", "currency").
		Values(payment.ID, payment.OrderID, payment.AmountMinor, payment.Currency).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "payments_order_id_key" {

			return domain.ErrPaymentAlreadyExists
		}

		return err
	}

	return nil
}

func (r *PaymentRepository) GetByOrderID(
	ctx context.Context,
	orderID string,
) (*domain.Payment, error) {
	query, args, err := squirrel.Select("id", "order_id", "amount_minor", "currency").
		From("payments").
		Where(squirrel.Eq{"order_id": orderID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	payment := &domain.Payment{}

	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.AmountMinor,
		&payment.Currency,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPaymentNotFound
	}
	if err != nil {
		return nil, err
	}

	return payment, nil
}
