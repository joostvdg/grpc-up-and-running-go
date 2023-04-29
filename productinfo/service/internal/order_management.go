package internal

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"io"
	"log"
	pb "productinfo/service/ecommerce"
	"strings"
	"time"
)

func (s *server) GetOrder(ctx context.Context, in *wrapperspb.StringValue) (*pb.Order, error) {

	orderId := in.Value
	order, ok := s.orderMap[orderId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "Order with ID %s not found", orderId)
	}
	return order, nil
}

func (s *server) UpdateOrders(stream pb.OrderManagement_UpdateOrdersServer) error {

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

func (s *server) AddOrder(ctx context.Context, orderReq *pb.Order) (*wrapperspb.StringValue, error) {

	orderId := orderReq.Id
	if s.orderMap == nil {
		s.orderMap = make(map[string]*pb.Order)
	}

	s.orderMap[orderId] = orderReq

	return &wrapperspb.StringValue{Value: orderId}, status.New(codes.OK, "").Err()
}

func (s *server) SearchOrders(searchQuery *wrapperspb.StringValue, stream pb.OrderManagement_SearchOrdersServer) error {

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

func (s *server) ProcessOrders(stream pb.OrderManagement_ProcessOrdersServer) error {
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
