package internal

import (
	"context"
	epb "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"io"
	"log"
	pb "productinfo/service/ecommerce"
	"strings"
	"time"
)

func (s *Server) GetOrder(ctx context.Context, in *wrapperspb.StringValue) (*pb.Order, error) {

	orderId := in.Value
	order, ok := s.orderMap[orderId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "Order with ID %s not found", orderId)
	}
	return order, nil
}

func (s *Server) UpdateOrders(stream pb.OrderManagement_UpdateOrdersServer) error {

	ordersStr := "Updated Order IDs : "
	for {
		order, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&wrapperspb.StringValue{Value: "Orders processed " + ordersStr})
		}
		if err != nil {
			return err
		}

		s.orderMap[order.Id] = order

		ordersStr += order.Id + ", "
		log.Printf("Order ID %s updated\n", order.Id)
	}
}

func (s *Server) SearchOrders(searchQuery *wrapperspb.StringValue, stream pb.OrderManagement_SearchOrdersServer) error {

	md, ok := metadata.FromIncomingContext(stream.Context())
	if ok {
		log.Printf("SearchOrders function invoked with metadata %+v\n", md)
	} else {
		log.Printf("SearchOrders function invoked with no metadata\n")
	}

	header := metadata.Pairs("key1", "val1")
	stream.SendHeader(header)
	trailer := metadata.Pairs("key2", "val2")
	stream.SetTrailer(trailer)

	for key, order := range s.orderMap {
		log.Printf("key: %s, value: %v\n", key, order)
		for _, itemStr := range order.Items {
			log.Printf("itemStr: %s\n", itemStr)
			if strings.Contains(itemStr, searchQuery.Value) {
				err := stream.Send(order)
				if err != nil {
					return err
				}
				log.Printf("Matching Order Found: " + order.Id)
				break
			}
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

func (s *Server) ProcessOrders(stream pb.OrderManagement_ProcessOrdersServer) error {
	batchMarker := 0
	const orderBatchSize = 3

	for {
		orderId, err := stream.Recv()
		if err == io.EOF {
			for _, combinedShipment := range s.combinedShipmentMap {
				err = stream.Send(combinedShipment)
				if err != nil {
					return err
				}
			}
			return nil
		}

		if err != nil {
			return err
		}

		// Process order
		log.Printf("Processing order ID : %s", orderId)
		combinedShipment := &pb.CombinedShipment{Id: "cmb-" + orderId.Value}
		s.combinedShipmentMap[orderId.Value] = combinedShipment

		// Process order and send combined shipment
		if batchMarker == orderBatchSize {
			// Stream combined orders to the client in batches
			for _, combinedShipment := range s.combinedShipmentMap {
				err = stream.Send(combinedShipment)
				if err != nil {
					return err
				}
			}
			batchMarker = 0
			s.combinedShipmentMap = make(map[string]*pb.CombinedShipment)
		} else {
			batchMarker++
		}
	}
}

func (s *Server) AddOrder(ctx context.Context, orderReq *pb.Order) (*wrapperspb.StringValue, error) {

	sleepDuration := 5
	log.Printf("Sleeping for : %d seconds", sleepDuration)
	time.Sleep(time.Duration(sleepDuration) * time.Second)

	if ctx.Err() == context.Canceled {
		log.Printf("Client cancelled, abandoning.")
		return nil, status.Error(codes.Canceled, "Client cancelled, abandoning.")
	}

	if ctx.Err() == context.DeadlineExceeded {
		log.Printf("Client deadline exceeded, abandoning.")
		return nil, status.Error(codes.DeadlineExceeded, "Client deadline exceeded, abandoning.")
	}

	if orderReq.Id == "-1" {
		log.Printf("Invalid Order ID: %s", orderReq.Id)

		errorStatus := status.New(codes.InvalidArgument, "Invalid Order ID: "+orderReq.Id)
		ds, err := errorStatus.WithDetails(
			&epb.BadRequest_FieldViolation{
				Field:       "ID",
				Description: "Order ID cannot be negative",
			},
		)
		if err != nil {
			return nil, errorStatus.Err()
		}
		return nil, ds.Err()
	}

	s.orderMap[orderReq.Id] = orderReq
	log.Printf("Order Added %v\n", orderReq)

	return &wrapperspb.StringValue{Value: "Order Added: " + orderReq.Id}, status.New(codes.OK, "").Err()
}
