package internal

import (
	"google.golang.org/grpc"
	"log"
	"net"
	pb "productinfo/service/ecommerce"
)

// server is used to implement ecommerce.ProductInfoServer.
type server struct {
	productMap          map[string]*pb.Product
	orderMap            map[string]*pb.Order
	combinedShipmentMap map[string]*pb.CombinedShipment
}

func (s *server) initMaps() {
	s.productMap = make(map[string]*pb.Product)
	s.orderMap = make(map[string]*pb.Order)
	s.combinedShipmentMap = make(map[string]*pb.CombinedShipment)
}
func (s *server) initOrders() {
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

func StartGRPCServer(port string) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	srv := &server{}
	srv.initMaps()
	srv.initOrders()
	pb.RegisterProductInfoServer(s, srv)
	pb.RegisterOrderManagementServer(s, srv)

	log.Printf("Starting gRPC server on port %s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
