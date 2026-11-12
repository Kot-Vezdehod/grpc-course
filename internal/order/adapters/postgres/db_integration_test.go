package postgres

import (
	"context"
	"database/sql"
	"grpc-course/internal/order/domain"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

type OrderDatabaseTestSuite struct {
	suite.Suite
	DataSourceURL     string
	postgresContainer *tcpostgres.PostgresContainer
	db                *sql.DB
}

func (o *OrderDatabaseTestSuite) SetupSuite() {
	ctx := context.Background()

	postgresContainer, err := tcpostgres.Run(
		ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("orders"),
		tcpostgres.WithUsername("order_user"),
		tcpostgres.WithPassword(uuid.NewString()),
		tcpostgres.BasicWaitStrategies(),
	)

	if err != nil {
		o.T().Fatal(err)
	}

	o.postgresContainer = postgresContainer

	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		o.T().Fatal(err)
	}

	o.DataSourceURL = dsn

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		o.T().Fatal(err)
	}

	if err := db.PingContext(ctx); err != nil {
		o.T().Fatal(err)
	}
	o.db = db

	if err := goose.Up(db, "../../../../migrations/orders"); err != nil {
		o.T().Fatal(err)
	}

}

func (o *OrderDatabaseTestSuite) TearDownSuite() {
	if o.db != nil {
		if err := o.db.Close(); err != nil {
			o.T().Fatal(err)
		}
	}

	if err := testcontainers.TerminateContainer(
		o.postgresContainer,
	); err != nil {
		o.T().Fatal(err)
	}
}

func (o *OrderDatabaseTestSuite) TestCrateOrder_HappyPath() {
	repo := NewOrderRepository(o.db)

	order := &domain.Order{
		ID:       "order-001",
		Currency: "USD",
		Items: []domain.OrderItem{
			{
				ProductID:      "product-001",
				Quantity:       2,
				UnitPriceMinor: 500,
			},
		},
	}

	err := repo.CreateOrder(context.Background(), order)
	o.Require().NoError(err)

	var (
		id       string
		currency string
	)

	err = o.db.QueryRow(
		`SELECT id, currency FROM orders WHERE id = $1`,
		order.ID,
	).Scan(&id, &currency)

	o.Require().NoError(err)
	o.Equal(order.ID, id)
	o.Equal(order.Currency, currency)

	var (
		productID      string
		quantity       int32
		unitPriceMinor int64
	)

	err = o.db.QueryRow(
		`SELECT product_id, quantity, unit_price_minor
	FROM order_items
	WHERE order_id = $1`,
		order.ID,
	).Scan(&productID, &quantity, &unitPriceMinor)

	o.Require().NoError(err)
	o.Equal("product-001", productID)
	o.Equal(int32(2), quantity)
	o.Equal(int64(500), unitPriceMinor)
}

func TestOrderDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(OrderDatabaseTestSuite))
}
