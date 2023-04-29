package main

import (
	"context"
	"io"

	"log"
	"time"

	"google.golang.org/grpc"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
	pb "productinfo/client/ecommerce"
)

const (
	address = "localhost:50051"
)

func main() {
	defer log.Println("-----------------------------------------")
	defer log.Println("Done.")
	log.Println("Talking to the ProductInfo server...")
	interactWithProductInfoServer()
	log.Println("-----------------------------------------")

	log.Println("Doing a Streaming Search with the OrderManagement server...")
	searchOrdersWithOrderManagementServer()
	log.Println("-----------------------------------------")

	log.Println("Doing a Streaming OrdersUpdate with the OrderManagement server...")
	OrdersUpdateWithOrderManagementServer()
	log.Println("-----------------------------------------")

	log.Println("Doing a Streaming OrdersProcessing with the OrderManagement server...")
	OrdersProcessingWithOrderManagementServer()
	log.Println("-----------------------------------------")

	log.Println("Talking to the OrderManagement server...")
	interactWithOrderManagementServer()
}

func OrdersProcessingWithOrderManagementServer() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	orderManagementClient := pb.NewOrderManagementClient(conn)
	streamProcessOrder, err := orderManagementClient.ProcessOrders(ctx)
	if err != nil {
		log.Fatalf("%v.ProcessOrders(_) = _, %v", orderManagementClient, err)
	}

	order1 := &wrapperspb.StringValue{Value: "201"}
	order2 := &wrapperspb.StringValue{Value: "202"}
	order3 := &wrapperspb.StringValue{Value: "203"}
	firstBatchOfOrders := []*wrapperspb.StringValue{order1, order2, order3}

	for _, order := range firstBatchOfOrders {
		if err := streamProcessOrder.Send(order); err != nil {
			log.Fatalf("%v.Send(%v) = %v", streamProcessOrder, order, err)
		}
		time.Sleep(time.Second)
	}

	channel := make(chan struct{})
	go asncClientBidirectionalRPC(streamProcessOrder, channel)
	time.Sleep(time.Millisecond * 1000)

	order4 := &wrapperspb.StringValue{Value: "204"}
	order5 := &wrapperspb.StringValue{Value: "205"}
	secondBatchOfOrders := []*wrapperspb.StringValue{order4, order5}

	for _, order := range secondBatchOfOrders {
		if err := streamProcessOrder.Send(order); err != nil {
			log.Fatalf("%v.Send(%v) = %v", streamProcessOrder, order, err)
		}
		time.Sleep(time.Second)
	}

	if err := streamProcessOrder.CloseSend(); err != nil {
		log.Fatal(err)
	}

	<-channel
}

func asncClientBidirectionalRPC(streamProcessOrder pb.OrderManagement_ProcessOrdersClient, channel chan struct{}) {
	for {
		combinedShipment, errProcOrder := streamProcessOrder.Recv()
		if errProcOrder == io.EOF {
			break
		}
		log.Printf("Combined shipment : %v", combinedShipment.OrderLists)
	}
	<-channel
}

func OrdersUpdateWithOrderManagementServer() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	orderManagementClient := pb.NewOrderManagementClient(conn)
	stream, err := orderManagementClient.UpdateOrders(ctx)
	if err != nil {
		log.Fatalf("%v.UpdateOrders(_) = _, %v", orderManagementClient, err)
	}

	order1 := pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Mac Book Pro"}, Destination: "Mountain View, CA", Price: 1800.00}
	order2 := pb.Order{Id: "103", Items: []string{"Apple Watch S4"}, Destination: "San Jose, CA", Price: 400.00}
	order3 := pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub"}, Destination: "Mountain View, CA", Price: 400.00}
	order4 := pb.Order{Id: "105", Items: []string{"Amazon Echo"}, Destination: "San Jose, CA", Price: 30.00}

	orders := []*pb.Order{&order1, &order2, &order3, &order4}
	for _, o := range orders {
		if err := stream.Send(o); err != nil {
			log.Fatalf("%v.Send(%v) = %v", stream, o, err)
		}
		time.Sleep(3 * time.Second)
	}

	updateRes, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
	}
	log.Printf("Update Orders Res : %s", updateRes.Value)
}

func searchOrdersWithOrderManagementServer() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	orderManagementClient := pb.NewOrderManagementClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	searchStream, err := orderManagementClient.SearchOrders(ctx, &wrapperspb.StringValue{Value: "Apple"})
	if err != nil {
		log.Fatalf("SearchOrders failed: %v", err)
	}

	for {
		order, err := searchStream.Recv()
		if err == io.EOF {
			log.Println("End of SearchOrders")
			break
		}
		if err != nil {
			log.Fatalf("%v.SearchOrders(_) = _, %v", orderManagementClient, err)
		}
		log.Printf("Search Result: %v\n", order)
	}
}

func interactWithOrderManagementServer() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	orderManagementClient := pb.NewOrderManagementClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	orderId := wrapperspb.String("106")
	order, err := orderManagementClient.GetOrder(ctx, orderId)
	if err != nil {
		log.Fatalf("could not get order: %v", err)
	}
	log.Printf("Order: %s", order.String())
}

func interactWithProductInfoServer() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewProductInfoClient(conn)

	name := "test"
	description := "we are testing"
	price := float32(9239.23)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	response, err := client.AddProduct(ctx, &pb.Product{Name: name, Description: description, Price: price})
	if err != nil {
		log.Fatalf("could not add product: %v", err)
	}
	log.Printf("Product ID: %s added successfully", response.Value)

	product, err := client.GetProduct(ctx, &pb.ProductID{Value: response.Value})
	if err != nil {
		log.Fatalf("could not get product: %v", err)
	}
	log.Printf("Product: %s", product.String())
}
