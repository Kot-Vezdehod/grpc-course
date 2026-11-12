package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"grpc-course/gen/hello"
	inventorypb "grpc-course/gen/inventory"
	orderpb "grpc-course/gen/order"
	paymentpb "grpc-course/gen/payment"
	ordergrpc "grpc-course/internal/order/adapters/grpc"
	"grpc-course/internal/order/adapters/inventorygrpc"
	"grpc-course/internal/order/adapters/paymentgrpc"
	"grpc-course/internal/order/adapters/postgres"
	"grpc-course/internal/order/application"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	metricsadapter "grpc-course/internal/order/adapters/metrics"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// server реализует gRPC-сервис Greeter.
type server struct {
	hello.UnimplementedGreeterServer
}

// SayHello возвращает приветствие для имени из запроса.
func (s *server) SayHello(ctx context.Context, req *hello.HelloRequest) (*hello.HelloReply, error) {

	return &hello.HelloReply{
		Message: "Hello, " + req.Name,
	}, nil
}

// connectDB открывает и проверяет подключение к базе данных заказов.
func connectDB(ctx context.Context) (*sql.DB, error) {
	dsn := strings.TrimSpace(os.Getenv("ORDER_DATABASE_URL"))
	if dsn == "" {
		return nil, fmt.Errorf("ORDER_DATABASE_URL is required")
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

func timingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()

	resp, err := handler(ctx, req)

	duration := time.Since(start)

	slog.Info(
		"gRPC request completed",
		"method", info.FullMethod,
		"duration", duration.String(),
		"code", status.Code(err).String(),
	)

	return resp, err
}

func newTimingInterceptor(
	requests *prometheus.CounterVec,
	duration *prometheus.HistogramVec,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		start := time.Now()

		resp, err = handler(ctx, req)

		elapsed := time.Since(start)
		duration.WithLabelValues(info.FullMethod).Observe(elapsed.Seconds())

		requests.WithLabelValues(
			info.FullMethod,
			status.Code(err).String(),
		).Inc()

		slog.Info(
			"gRPC request completed",
			"method", info.FullMethod,
			"duration", elapsed.String(),
			"code", status.Code(err).String(),
		)

		return resp, err
	}
}

// Main настраивает зависимости и запускает gRPC-сервер.
func main() {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, nil),
	).With("service", "order")

	ordersCreated := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orders_created_total",
		Help: "The total number of orders created",
	})
	prometheus.MustRegister(ordersCreated)

	metricsAdapter := metricsadapter.NewOrderMetrics(ordersCreated)

	grpcRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_requests_total",
			Help: "Total number of gRPC requests by method and status code",
		},
		[]string{"method", "code"},
	)
	prometheus.MustRegister(grpcRequestsTotal)

	grpcDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_duration_seconds",
			Help:    "Duration of gRPC requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)
	prometheus.MustRegister(grpcDuration)

	ctx := context.Background()
	db, err := connectDB(ctx)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	repo := postgres.NewOrderRepository(db)

	caCert, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		log.Fatalf("failed to read CA certificate: %v", err)
	}

	certPool := x509.NewCertPool()

	if !certPool.AppendCertsFromPEM(caCert) {
		log.Fatalf("failed to append CA certificate")
	}

	clientCert, err := tls.LoadX509KeyPair(
		"certs/order-client.crt",
		"certs/order-client.key",
	)
	if err != nil {
		log.Fatalf("failed to load client certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      certPool,
		ServerName:   "localhost",
		Certificates: []tls.Certificate{clientCert},
	}
	creds := credentials.NewTLS(tlsConfig)

	paymentAddress := os.Getenv("PAYMENT_SERVICE_ADDR")
	if paymentAddress == "" {
		paymentAddress = "localhost:50052"
	}

	paymentConn, err := grpc.NewClient(
		paymentAddress,
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(
			grpc_retry.UnaryClientInterceptor(
				// повторяем только transient ошибки;
				grpc_retry.WithCodes(
					codes.Unavailable,
					codes.ResourceExhausted,
				),
				// ограничение на кол-во повторных попыток
				grpc_retry.WithMax(3),
				// задержка между попытками
				grpc_retry.WithBackoff(
					grpc_retry.BackoffLinear(200*time.Millisecond),
				),
			),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer paymentConn.Close()

	inventoryAddress := os.Getenv("INVENTORY_SERVICE_ADDRESS")
	if inventoryAddress == "" {
		inventoryAddress = "localhost:50053"
	}

	inventoryConn, err := grpc.NewClient(
		inventoryAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to create inventory client: %v", err)
	}
	defer inventoryConn.Close()

	paymentClient := paymentpb.NewPaymentServiceClient(paymentConn)
	paymentGateway := paymentgrpc.NewClient(paymentClient)

	inventoryPBClient := inventorypb.NewInventoryServiceClient(inventoryConn)
	inventoryClient := inventorygrpc.NewClient(inventoryPBClient)

	orderService := application.NewOrderService(repo, paymentGateway, inventoryClient, logger, metricsAdapter)
	orderServer := ordergrpc.NewServer(orderService)

	address := os.Getenv("ADDRESS")
	if address == "" {
		address = "localhost:50051"
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(newTimingInterceptor(grpcRequestsTotal, grpcDuration)),
	)
	hello.RegisterGreeterServer(grpcServer, &server{})
	orderpb.RegisterOrderServiceServer(grpcServer, orderServer)

	logger.Info("gRPC server started", "address", address)

	metricsMux := http.NewServeMux()

	metricsMux.Handle("/metrics", promhttp.Handler())

	go func() {
		err := http.ListenAndServe("localhost:9091", metricsMux)
		if err != nil {
			logger.Error("metrics server failed", "error", err)
		}
	}()

	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("gRPC server failed", "error", err)
	}

}
