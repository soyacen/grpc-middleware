package grpcoteltrace

import (
	"context"
	"io"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func startSpan(ctx context.Context,
	tracer trace.Tracer,
	propagator propagation.TextMapPropagator,
	method string,
	kind trace.SpanKind,
) (context.Context, trace.Span) {
	if kind == trace.SpanKindServer {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.Pairs()
			ctx = metadata.NewIncomingContext(ctx, md)
		}
		ctx = propagator.Extract(ctx, MetaDataCarrier(md))
	}
	ctx, span := tracer.Start(
		ctx,
		method,
		trace.WithSpanKind(kind),
		trace.WithAttributes(
			semconv.RPCSystemKey.String("grpc"),
			semconv.RPCMethodKey.String(method),
		),
	)
	if kind == trace.SpanKindClient {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.Pairs()
			ctx = metadata.NewOutgoingContext(ctx, md)
		}
		propagator.Inject(ctx, MetaDataCarrier(md))
		md, ok = metadata.FromOutgoingContext(ctx)
	}
	return ctx, span
}

func endSpan(err error, span trace.Span) {
	if err != nil && err != io.EOF {
		span.RecordError(err)
		span.SetStatus(codes.Error, status.Code(err).String())
	} else {
		span.SetStatus(codes.Ok, "OK")
	}
	span.End()
}
