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

// Package main implements a client for Greeter service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	grpccontext "github.com/soyacen/grpc-middleware/context"
	grpcoteltrace "github.com/soyacen/grpc-middleware/otel/trace"
	"github.com/soyacen/grpc-middleware/otel/trace/example/helloworld"
	"github.com/soyacen/grpc-middleware/otel/trace/example/utils"
)

const (
	address     = "localhost:50051"
	defaultName = "world"
)

func main() {
	// Set up a connection to the server.
	err := utils.SetTracerProvider(1, "test.trace", "test.trace.1", "v0.0.0")
	if err != nil {
		panic(err)
	}
	utils.SetPropagator()
	tracer := otel.Tracer("test.trace")
	propagator := otel.GetTextMapPropagator()
	unary, stream := grpcoteltrace.ClientInterceptor(grpcoteltrace.WithTracer(tracer), grpcoteltrace.WithPropagator(propagator))
	unary2, stream2 := grpccontext.ClientInterceptor(grpccontext.WithContextFunc(func(ctx context.Context) context.Context {
		spanContext := trace.SpanContextFromContext(ctx)
		if spanContext.HasTraceID() {
			fmt.Println(spanContext.TraceID().String())
		}
		return ctx
	}))
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithChainUnaryInterceptor(unary, unary2), grpc.WithChainStreamInterceptor(stream, stream2))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := helloworld.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	r, err := c.SayHello(context.Background(), &helloworld.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
}
