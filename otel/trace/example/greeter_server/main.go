/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a server for Greeter service.
package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"

	"go.opentelemetry.io/otel/trace"

	grpccontext "github.com/soyacen/grpc-middleware/context"
	grpcoteltrace "github.com/soyacen/grpc-middleware/otel/trace"
	"github.com/soyacen/grpc-middleware/otel/trace/example/helloworld"
	"github.com/soyacen/grpc-middleware/otel/trace/example/utils"
)

const (
	port = ":50051"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	helloworld.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &helloworld.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	err = utils.SetTracerProvider(1, "test.trace", "test.trace.1", "v0.0.0")
	if err != nil {
		panic(err)
	}
	utils.SetPropagator()

	tracer := otel.Tracer("test.trace")
	propagator := otel.GetTextMapPropagator()

	unary, stream := grpcoteltrace.ServerInterceptor(grpcoteltrace.WithTracer(tracer), grpcoteltrace.WithPropagator(propagator))

	unary2, stream2 := grpccontext.ServerInterceptor(grpccontext.WithContextFunc(func(ctx context.Context) context.Context {
		spanContext := trace.SpanContextFromContext(ctx)
		if spanContext.HasTraceID() {
			fmt.Println(spanContext.TraceID().String())
		}
		return ctx
	}))
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(unary, unary2), grpc.ChainStreamInterceptor(stream, stream2))
	helloworld.RegisterGreeterServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
