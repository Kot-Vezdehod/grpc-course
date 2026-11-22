package main

import (
	inventorypb "grpc-course/gen/inventory"
	orderpb "grpc-course/gen/order"
	gatewayhttp "grpc-course/internal/gateway/http"
	"log"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("write response: %v", err)
		}
	})

	orderAddress := os.Getenv("ORDER_SERVICE_ADDRESS")
	if orderAddress == "" {
		orderAddress = "localhost:50051"
	}
	inventoryAddress := os.Getenv("INVENTORY_SERVICE_ADDRESS")
	if inventoryAddress == "" {
		inventoryAddress = "localhost:50053"
	}

	conn, err := grpc.NewClient(
		orderAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	inventoryConn, err := grpc.NewClient(
		inventoryAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer inventoryConn.Close()

	orderClient := orderpb.NewOrderServiceClient(conn)
	inventoryClient := inventorypb.NewInventoryServiceClient(inventoryConn)

	handler := gatewayhttp.NewHandler(orderClient, inventoryClient)

	handler.RegisterRoutes(mux)

	log.Println("API Gateway listening on :8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}

}
