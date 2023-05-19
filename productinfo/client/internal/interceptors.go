package internal

import (
	"context"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"log"
	"time"
)

func OrderUnaryClientInterceptor(
	ctx context.Context,
	method string,
	req,
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption) error {

	// Pre-processing logic
	log.Println("Method:", method, "Request:", req)

	newCtx, span := otel.Tracer("productinfo-client").Start(ctx, method)
	err := invoker(newCtx, method, req, reply, cc, opts...)

	if err != nil {
		log.Println("Error:", err)
	} else {
		log.Println("Response:", reply)
	}
	span.End()
	return err
}

func ClientStreamInterceptor(
	ctx context.Context,
	desc *grpc.StreamDesc,
	cc *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	opts ...grpc.CallOption) (grpc.ClientStream, error) {

	log.Println("======= [Client Interceptor] ", method)
	newCtx, span := otel.Tracer("productinfo-client").Start(ctx, method)
	s, err := streamer(newCtx, desc, cc, method, opts...)
	if err != nil {
		return nil, err
	}
	span.End()
	return NewWrappedStream(s), nil
}

type WrappedStream struct {
	grpc.ClientStream
}

func (w *WrappedStream) RecvMsg(m interface{}) error {
	log.Printf("====== [Client Stream Interceptor Wrapper] Receive a message (Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	err := w.ClientStream.RecvMsg(m)
	if err != nil {
		log.Printf("====== [Client Stream Interceptor Wrapper] error while receiving %v", err)
	}
	return w.ClientStream.RecvMsg(m)
}

func (w *WrappedStream) SendMsg(m interface{}) error {
	log.Printf("====== [Client Stream Interceptor Wrapper] Send a message (Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	err := w.ClientStream.SendMsg(m)
	if err != nil {
		log.Printf("====== [Client Stream Interceptor Wrapper] error while sending %v", err)
	}
	return w.ClientStream.SendMsg(m)
}

func NewWrappedStream(s grpc.ClientStream) grpc.ClientStream {
	return &WrappedStream{s}
}
