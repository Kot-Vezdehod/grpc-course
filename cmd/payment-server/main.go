package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	paymentpb "grpc-course/gen/payment"
	paymentgrpc "grpc-course/internal/payment/adapters/grpc"
	paymentpostgres "grpc-course/internal/payment/adapters/postgres"
	"grpc-course/internal/payment/application"
	"log"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func connectDB(ctx context.Context) (*sql.DB, error) {
	dsn := strings.TrimSpace(os.Getenv("PAYMENT_DATABASE_URL"))
	if dsn == "" {
		return nil, fmt.Errorf("PAYMENT_DATABASE_URL is required")
	}

	db, err := sql.Open("pgx", dsn)
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

func main() {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, nil),
	).With("service", "payment")

	ctx := context.Background()

	db, err := connectDB(ctx)
	if err != nil {
		log.Fatalf("connectDB: %v", err)
	}
	defer db.Close()

	repo := paymentpostgres.NewPaymentRepository(db)

	service := application.NewPaymentService(repo, logger)
	paymentServer := paymentgrpc.NewServer(service)

	address := os.Getenv("ADDRESS")
	if address == "" {
		address = "localhost:50052"
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	serverCert, err := tls.LoadX509KeyPair(
		"certs/payment-server.crt",
		"certs/payment-server.key",
	)
	if err != nil {
		log.Fatalf("failed to load payment-server certificate: %v", err)
	}

	caCert, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		log.Fatalf("failed to read CA certificate: %v", err)
	}

	clientCAPool := x509.NewCertPool()
	if !clientCAPool.AppendCertsFromPEM(caCert) {
		log.Fatalf("failed to append CA certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    clientCAPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	creds := credentials.NewTLS(tlsConfig)

	server := grpc.NewServer(grpc.Creds(creds))

	paymentpb.RegisterPaymentServiceServer(server, paymentServer)

	logger.Info("starting gRPC server on:", "address", address)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
