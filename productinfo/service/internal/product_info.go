package internal

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "productinfo/service/ecommerce"

	"github.com/gofrs/uuid"
)

func (s *server) AddProduct(ctx context.Context, product *pb.Product) (*pb.ProductID, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	product.Id = id.String()

	if s.productMap == nil {
		s.productMap = make(map[string]*pb.Product)
	}

	s.productMap[product.Id] = product

	return &pb.ProductID{Value: product.Id}, status.New(codes.OK, "").Err()
}

func (s *server) GetProduct(ctx context.Context, id *pb.ProductID) (*pb.Product, error) {
	if s.productMap == nil {
		return nil, nil
	}

	product, ok := s.productMap[id.Value]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "Could not find product with id %s", id.Value)
	}

	return product, status.New(codes.OK, "").Err()
}
