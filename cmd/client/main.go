package main

import (
	"context"
	"log"

	"grpc-course/gen/hello"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// main подключается к gRPC-серверу и вызывает метод SayHello.
func main() {
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := hello.NewGreeterClient(conn)
	response, err := client.SayHello(context.Background(), &hello.HelloRequest{Name: "world"})
	if err != nil {
		log.Fatal(err)
	}

	log.Println(response.GetMessage())
}
