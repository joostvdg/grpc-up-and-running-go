package main

import (
	"context"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	epb "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"sync"

	"log"
	"time"

	"google.golang.org/grpc"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
	pb "productinfo/client/ecommerce"
	"productinfo/client/internal"
)

const (
	address = "localhost:50051"
)

var (
	resource          *sdkresource.Resource
	initResourcesOnce sync.Once
)

func main() {
	tp := initTracerProvider()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Tracer Provider Shutdown: %v", err)
		}
	}()

	mp := initMeterProvider()
	defer func() {
		if err := mp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Error shutting down meter provider: %v", err)
		}
	}()

	defer log.Println("-----------------------------------------")
	defer log.Println("Done.")
	log.Println("Talking to the ProductInfo server... (w/o auth)")
	interactWithProductInfoServer()
	log.Println("-----------------------------------------")

	log.Println("Talking to the ProductInfo server... (w/ auth)")
	interactWithProductInfoServerWithAuth()
	log.Println("-----------------------------------------")

	log.Println("Talking to the ProductInfo server... (withInvalidOrderId)")
	withInvalidOrderId()
	log.Println("-----------------------------------------")

	log.Println("Talking to the ProductInfo server... (with cancel)")
	withCancel()
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

func withInvalidOrderId() {
	grpcOpts := createGrpcOptions()
	conn, err := grpc.Dial(address, grpcOpts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	orderManagementClient := pb.NewOrderManagementClient(conn)

	order1 := pb.Order{Id: "-1", Items: []string{"Google Pixel 3A", "Mac Book Pro"}, Destination: "Mountain View, CA", Price: 1800.00}
	res, err := orderManagementClient.AddOrder(ctx, &order1)
	if err != nil {
		errorCode := status.Code(err)
		if errorCode == codes.InvalidArgument {
			log.Printf("Invalid Argument Error : %s", errorCode)
			errorStatus := status.Convert(err)
			for _, d := range errorStatus.Details() {
				switch info := d.(type) {
				case *epb.BadRequest_FieldViolation:
					log.Printf("Request Field Invalid: %s", info)
				default:
					log.Printf("Unexpected error type: %s", info)
				}
			}
		} else {
			log.Printf("Unhandled error : %s", errorCode)
		}
	} else {
		log.Printf("Order Added : %v", res.Value)
	}

}

func withCancel() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcOpts := createGrpcOptions()
	conn, err := grpc.Dial(address, grpcOpts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewOrderManagementClient(conn)
	streamProcOrder, _ := client.ProcessOrders(ctx)
	_ = streamProcOrder.Send(&wrapperspb.StringValue{Value: "101"})
	_ = streamProcOrder.Send(&wrapperspb.StringValue{Value: "102"})
	_ = streamProcOrder.Send(&wrapperspb.StringValue{Value: "103"})

	channel := make(chan bool, 1)

	go asncClientBidirectionalRPCWithBoolChan(streamProcOrder, channel)
	time.Sleep(time.Millisecond * 100)

	cancel()
	log.Println("Cancelled the context: RPC status", status.Code(ctx.Err()), ctx.Err())
	_ = streamProcOrder.Send(&wrapperspb.StringValue{Value: "104"})
	_ = streamProcOrder.CloseSend()
	<-channel
}

func asncClientBidirectionalRPCWithBoolChan(order pb.OrderManagement_ProcessOrdersClient, c chan bool) {
	for {
		combinedShipment, errProcOrder := order.Recv()
		if errProcOrder != nil {
			log.Printf("Error Receiving messages %v", errProcOrder)
			break
		} else {
			if errProcOrder == io.EOF {
				break
			}
			log.Printf("Combined shipment : %s", combinedShipment.OrderLists)
		}
	}
	c <- true
}

func OrdersProcessingWithOrderManagementServer() {
	grpcOpts := createGrpcOptions()
	conn, err := grpc.Dial(address, grpcOpts...)
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
	grpcOpts := createGrpcOptions()
	conn, err := grpc.Dial(address, grpcOpts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	clientDeadline := time.Now().Add(time.Duration(2 * time.Second))
	ctx, cancel := context.WithDeadline(context.Background(), clientDeadline)
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
	grpcOpts := createGrpcOptions()
	conn, err := grpc.Dial(address, grpcOpts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	orderManagementClient := pb.NewOrderManagementClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Option 1
	//md := metadata.New(map[string]string{"location": "NL"})

	// Options 2
	md := metadata.Pairs("timestamp", time.Now().Format(time.Stamp), "kn", "vn")
	ctx = metadata.NewOutgoingContext(ctx, md)

	searchStream, err := orderManagementClient.SearchOrders(ctx, &wrapperspb.StringValue{Value: "Apple"})
	if err != nil {
		log.Fatalf("SearchOrders failed: %v", err)
	}

	for {
		var header, trailer metadata.MD
		order, err := searchStream.Recv()
		if err == io.EOF {
			log.Println("End of SearchOrders")
			break
		}
		if err != nil {
			log.Fatalf("%v.SearchOrders(_) = _, %v", orderManagementClient, err)
		}
		log.Printf("Search Result: %v\n", order)
		header, err = searchStream.Header()
		if err != nil {
			log.Printf("Failed to get header: %v", err)
		}
		log.Println("Header: ", header)

		trailer = searchStream.Trailer()
		log.Println("Trailer: ", trailer)
	}
}

func interactWithOrderManagementServer() {
	grpcOpts := createGrpcOptions()
	conn, err := grpc.Dial(address, grpcOpts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	orderManagementClient := pb.NewOrderManagementClient(conn)

	clientDeadline := time.Now().Add(time.Duration(2 * time.Second))
	ctx, cancel := context.WithDeadline(context.Background(), clientDeadline)
	defer cancel()

	orderId := wrapperspb.String("106")
	order, err := orderManagementClient.GetOrder(ctx, orderId)
	if err != nil {
		log.Fatalf("could not get order: %v", err)
	}
	log.Printf("Order: %s", order.String())

	order1 := pb.Order{Id: "106", Items: []string{"iPhone XS", "Mac Book Pro"}, Destination: "San Jose, CA", Price: 2300.00}
	res, addErr := orderManagementClient.AddOrder(ctx, &order1)
	if addErr != nil {
		log.Fatalf("could not add order: %v", addErr)
	} else {
		log.Printf("AddOrder Response -> %s", res.Value)
	}
}

func interactWithProductInfoServer() {
	grpcOpts := createGrpcOptions()
	conn, err := grpc.Dial(address, grpcOpts...)
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
		log.Printf("could not add product: %v", err)
		return
	}

	log.Printf("Product ID: %s added successfully", response.Value)

	product, err := client.GetProduct(ctx, &pb.ProductID{Value: response.Value})
	if err != nil {
		log.Printf("could not get product: %v", err)
		return
	}
	log.Printf("Product: %s", product.String())
}

func interactWithProductInfoServerWithAuth() {
	grpcOpts := createGrpcOptionsWithAuth()
	conn, err := grpc.Dial(address, grpcOpts...)
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
		log.Printf("could not add product: %v", err)
	}

	log.Printf("Product ID: %s added successfully", response.Value)

	product, err := client.GetProduct(ctx, &pb.ProductID{Value: response.Value})
	if err != nil {
		log.Printf("could not get product: %v", err)
	}
	log.Printf("Product: %s", product.String())
}

func createGrpcOptions() []grpc.DialOption {
	noCreds := insecure.NewCredentials()
	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(noCreds),
		grpc.WithChainUnaryInterceptor(internal.OrderUnaryClientInterceptor, otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	}
	return grpcOpts
}

func createGrpcOptionsWithAuth() []grpc.DialOption {
	auth := internal.BasicAuth{
		Username: "admin",
		Password: "admin",
	}
	grpcOpts := createGrpcOptions()
	grpcOpts = append(grpcOpts, grpc.WithPerRPCCredentials(&auth))
	return grpcOpts
}

func initResource() *sdkresource.Resource {
	initResourcesOnce.Do(func() {
		extraResources, _ := sdkresource.New(
			context.Background(),
			sdkresource.WithOS(),
			sdkresource.WithProcess(),
			sdkresource.WithContainer(),
			sdkresource.WithHost(),
		)
		resource, _ = sdkresource.Merge(
			sdkresource.Default(),
			extraResources,
		)
	})
	return resource
}

func initTracerProvider() *sdktrace.TracerProvider {
	ctx := context.Background()

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint("localhost:4317"),
	)
	if err != nil {
		log.Fatalf("OTLP Trace gRPC Creation: %v", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(initResource()),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp
}

func initMeterProvider() *sdkmetric.MeterProvider {
	ctx := context.Background()

	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure(), otlpmetricgrpc.WithEndpoint("localhost:4317"))
	if err != nil {
		log.Fatalf("new otlp metric grpc exporter failed: %v", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(initResource()),
	)
	global.SetMeterProvider(mp)
	return mp
}
