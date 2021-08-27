package utils

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func SetPropagator() {
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.Baggage{},
		propagation.TraceContext{},
	)
	otel.SetTextMapPropagator(propagator)
}

func SetTracerProvider(samplingRate float64, serviceName, serviceID, version string) error {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(os.Stdout),
	)
	if err != nil {
		return fmt.Errorf("failed new exporter, %w", err)
	}
	resource, err := newResource(serviceName, serviceID, version)
	if err != nil {
		return err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.ParentBased(newSample(samplingRate))),
		sdktrace.WithResource(resource),
	)
	otel.SetTracerProvider(tp)
	return nil
}

func newSample(samplingRate float64) sdktrace.Sampler {
	var sampler sdktrace.Sampler
	if samplingRate >= 1 {
		sampler = sdktrace.AlwaysSample()
	} else if samplingRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(samplingRate)
	}
	return sampler
}

func newResource(serviceName, serviceID, version string) (*resource.Resource, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed get hostname, %w", err)
	}
	resource, err := resource.New(
		context.Background(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithProcess(),
		resource.WithFromEnv(),
		resource.WithHost(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			attribute.Key("hostname").String(hostname),
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceInstanceIDKey.String(serviceID),
			semconv.ServiceVersionKey.String(version),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed new resource, %w", err)
	}
	return resource, nil
}
