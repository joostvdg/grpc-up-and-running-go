package internal

import (
	"google.golang.org/grpc"
	"log"
	"time"
)

type WrappedStream struct {
	grpc.ServerStream
}

func (w *WrappedStream) RecvMsg(m interface{}) error {
	log.Printf("====== [Server Stream Interceptor Wrapper] Receive a message (Type: %T) at %s", m, time.Now().Format(time.RFC3339))
	return w.ServerStream.RecvMsg(m)
}

func (w *WrappedStream) SendMsg(m interface{}) error {
	log.Printf("====== [Server Stream Interceptor Wrapper] Send a message (Type: %T) at %s", m, time.Now().Format(time.RFC3339))
	return w.ServerStream.SendMsg(m)
}

func NewWrappedStream(s grpc.ServerStream) grpc.ServerStream {
	return &WrappedStream{s}
}

func OrderServerStreamInterceptor(srv interface{}, serverStream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Println("====== [Server Stream Interceptor] ", info.FullMethod)
	err := handler(srv, NewWrappedStream(serverStream))
	if err != nil {
		log.Printf("RPC failed with error %v", err)
	}
	return err
}
