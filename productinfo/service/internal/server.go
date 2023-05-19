package internal

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
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	pb "productinfo/service/ecommerce"
	"sync"
)

var (
	resource          *sdkresource.Resource
	initResourcesOnce sync.Once
)

// Server is used to implement ecommerce.ProductInfoServer.
type Server struct {
	productMap          map[string]*pb.Product
	orderMap            map[string]*pb.Order
	combinedShipmentMap map[string]*pb.CombinedShipment
}

func (s *Server) initMaps() {
	s.productMap = make(map[string]*pb.Product)
	s.orderMap = make(map[string]*pb.Order)
	s.combinedShipmentMap = make(map[string]*pb.CombinedShipment)
}
func (s *Server) initOrders() {
	order1 := &pb.Order{
		Id:    "106",
		Items: []string{"Google Pixel 3A", "Mac Book Pro"},
	}
	s.orderMap[order1.Id] = order1

	order2 := &pb.Order{
		Id:    "107",
		Items: []string{"Apple Watch S4"},
	}
	s.orderMap[order2.Id] = order2

	order3 := &pb.Order{
		Id:    "108",
		Items: []string{"Apple", "Orange", "Banana"},
	}
	s.orderMap[order3.Id] = order3
}

const name = "productinfo"

func InitServer(s *Server) {
	s.initMaps()
	s.initOrders()
}

func StartGRPCServer(port string) {
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

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.ChainStreamInterceptor(OrderServerStreamInterceptor, otelgrpc.StreamServerInterceptor()),
		grpc.ChainUnaryInterceptor(orderUnaryInterceptor, EnsureValidBasicCredentials, otelgrpc.UnaryServerInterceptor()),
	)
	srv := &Server{}
	srv.initMaps()
	srv.initOrders()
	pb.RegisterProductInfoServer(s, srv)
	pb.RegisterOrderManagementServer(s, srv)
	reflection.Register(s)

	log.Printf("Starting gRPC Server on port %s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func orderUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Println("======= [Server Interceptor] ", info.FullMethod)

	newCtx, span := otel.Tracer(name).Start(ctx, info.FullMethod)
	m, err := handler(newCtx, req)
	if err != nil {
		log.Printf("RPC failed with error %v\n", err)
	}
	log.Printf("RPC returned with response %v\n", m)
	span.End()
	return m, err
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

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure(), otlptracegrpc.WithEndpoint("localhost:4317"))
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
