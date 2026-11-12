package main

import (
	"fmt"
	"grpc-course/gen/hello"

	"google.golang.org/protobuf/proto"
)

// main демонстрирует сериализацию и десериализацию Protobuf-сообщения.
func main() {
	request := &hello.HelloRequest{
		Name: "Marat",
	}

	data, err := proto.Marshal(request)
	if err != nil {
		panic(err)
	}

	fmt.Println("Исходное сообщение:", request)
	fmt.Println("Bytes", data)

	decoded := &hello.HelloRequest{}

	err = proto.Unmarshal(data, decoded)
	if err != nil {
		panic(err)
	}

	fmt.Println("After deserialisation:", decoded)
	fmt.Println("Name:", decoded.Name)

}
