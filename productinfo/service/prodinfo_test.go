package main

import (
	"context"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"log"
	"net"
	pb "productinfo/service/ecommerce"
	"productinfo/service/internal"
	"testing"
)

func TestServer_AddProduct(t *testing.T) {
	grpcServer := initGRPCServerHTTP2()
	defer grpcServer.Stop()

	conn, err := grpc.Dial(":50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewProductInfoClient(conn)
	name := "Apple iPhone 11"
	description := `Meet Apple iPhone 11. All-new dual-camera system with Ultra Wide and Night mode.`
	price := float32(1000.0)
	ctx := context.Background()

	// AddProduct
	r, err := client.AddProduct(ctx, &pb.Product{Name: name, Description: description, Price: price})
	if err != nil {
		t.Fatalf("Could not add product: %v", err)
	}
	assert.NotEmpty(t, r.Value)
}

// Copied from https://github.com/grpc-up-and-running/samples/blob/master/ch07/grpc-continous-integration/go/server/prodinfo_test.go
func initGRPCServerHTTP2() *grpc.Server {
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	srv := &internal.Server{}
	internal.InitServer(srv)
	pb.RegisterProductInfoServer(grpcServer, srv)
	pb.RegisterOrderManagementServer(grpcServer, srv)

	go func() {
		log.Printf("Starting gRPC Server on port %s", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	return grpcServer
}
