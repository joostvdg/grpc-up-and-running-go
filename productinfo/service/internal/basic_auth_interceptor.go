package internal

import (
	"context"
	"encoding/base64"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"strings"
)

var (
	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
	errInvalidToken    = status.Errorf(codes.Unauthenticated, "invalid credentials")
)

func AuthIsValid(authorization []string) bool {
	if len(authorization) < 1 {
		return false
	}

	token := strings.TrimPrefix(authorization[0], "Basic ")
	return token == base64.StdEncoding.EncodeToString([]byte("admin:admin"))
}

func EnsureValidBasicCredentials(ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {

	log.Println("unary interceptor: ", info.FullMethod)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Println("missing context metadata")
		return nil, errMissingMetadata
	}
	if !AuthIsValid(md["authorization"]) {
		log.Println("invalid basic credentials")
		return nil, errInvalidToken
	}
	log.Println("access granted")
	return handler(ctx, req)
}
