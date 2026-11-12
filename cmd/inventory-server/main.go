package main

import (
	"context"
	"database/sql"
	"fmt"
	inventorypb "grpc-course/gen/inventory"
	inventorygrpc "grpc-course/internal/inventory/adapters/grpc"
	"grpc-course/internal/inventory/adapters/postgres"
	"grpc-course/internal/inventory/application"
	"log"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
)

// connectDB открывает подключение к базе данных inventory и проверяет его с тайм-аутом.
func connectDB(ctx context.Context) (*sql.DB, error) {
	dsn := strings.TrimSpace(os.Getenv("INVENTORY_DATABASE_URL"))
	if dsn == "" {
		return nil, fmt.Errorf("INVENTORY_DATABASE_URL is required")
	}

	db, err := sql.Open(
		"pgx",
		dsn,
	)
	if err != nil {
		return nil, fmt.Errorf("connectDB: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connectDB: %w", err)
	}

	return db, nil
}

// main настраивает зависимости и запускает gRPC-сервер inventory на порту 50053.
func main() {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, nil),
	).With("service", "inventory")

	ctx := context.Background()
	db, err := connectDB(ctx)
	if err != nil {
		log.Fatalf("connectDB: %v", err)
	}
	defer db.Close()

	repo := postgres.NewInventoryRepository(db)

	inventoryService := application.NewInventoryService(repo, logger)
	inventoryServer := inventorygrpc.NewInventoryServer(inventoryService)

	address := os.Getenv("ADDRESS")
	if address == "" {
		address = "localhost:50053"
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	inventorypb.RegisterInventoryServiceServer(grpcServer, inventoryServer)

	logger.Info("starting gRPC server on:", "address", address)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
