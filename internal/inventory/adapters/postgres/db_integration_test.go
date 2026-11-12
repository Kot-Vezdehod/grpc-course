package postgres

import (
	"context"
	"database/sql"
	"grpc-course/internal/inventory/domain"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

type InventoryDatabaseTestSuite struct {
	suite.Suite
	DataSourceURL     string
	postgresContainer *tcpostgres.PostgresContainer
	db                *sql.DB
}

func (i *InventoryDatabaseTestSuite) SetupSuite() {
	ctx := context.Background()

	postgresContainer, err := tcpostgres.Run(
		ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("inventory_db"),
		tcpostgres.WithUsername("inventory"),
		tcpostgres.WithPassword(uuid.NewString()),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		i.T().Fatal(err)
	}

	i.T().Cleanup(func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			i.T().Errorf("terminate postgres container failed: %v", err)
		}
	})

	i.postgresContainer = postgresContainer

	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		i.T().Fatal(err)
	}

	i.DataSourceURL = dsn

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		i.T().Fatal(err)
	}

	i.T().Cleanup(func() {
		if err := db.Close(); err != nil {
			i.T().Errorf("close database connection: %v", err)
		}
	})

	if err := db.PingContext(ctx); err != nil {
		i.T().Fatal(err)
	}
	i.db = db

	if err := goose.Up(db, "../../../../migrations/inventory"); err != nil {
		i.T().Fatal(err)
	}
}

func TestOrderDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(InventoryDatabaseTestSuite))
}

func (i *InventoryDatabaseTestSuite) TestCreateAndGetByID() {
	ctx := context.Background()
	repo := NewInventoryRepository(i.db)
	prefix := strconv.FormatInt(time.Now().UnixNano(), 10)
	productID := prefix + "-product-001"

	inventory, err := domain.NewInventory(
		productID,
		"iphone 2000",
		"p-001",
		100,
		1299,
	)
	i.Require().NoError(err)

	err = repo.Create(ctx, inventory)
	i.Require().NoError(err)

	got, err := repo.GetByID(ctx, inventory.ProductID)
	i.Require().NoError(err)
	i.Require().Equal(inventory, got)
}

func (i *InventoryDatabaseTestSuite) TestCreate_ProductIDAlreadyExists() {
	ctx := context.Background()
	repo := NewInventoryRepository(i.db)
	prefix := strconv.FormatInt(time.Now().UnixNano(), 10)
	productID := prefix + "-product-001"

	inventory1, err := domain.NewInventory(
		productID,
		"iphone 2000",
		"p-001",
		100,
		1299,
	)
	i.Require().NoError(err)

	inventory2, err := domain.NewInventory(
		productID,
		"iphone 3000",
		"p-002",
		200,
		1290,
	)
	i.Require().NoError(err)

	err = repo.Create(ctx, inventory1)
	i.Require().NoError(err)

	err = repo.Create(ctx, inventory2)
	i.Require().ErrorIs(err, domain.ErrInventoryAlreadyExists)
}

func (i *InventoryDatabaseTestSuite) TestGetByID_ProductDoesNotExist() {
	ctx := context.Background()
	repo := NewInventoryRepository(i.db)
	productID := "not-exists-product"

	got, err := repo.GetByID(ctx, productID)
	i.Require().ErrorIs(err, domain.ErrInventoryDoesNotExist)
	i.Require().Nil(got)
}
