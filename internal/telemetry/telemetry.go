package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type TracerProvider struct {
	provider *trace.TracerProvider
}

func NewTracerProvider(serviceName, otlpEndpoint string, sampleRate float64) (*TracerProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otlpEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP gRPC exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(sampleRate)),
	)

	otel.SetTracerProvider(tp)

	return &TracerProvider{provider: tp}, nil
}

func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	return tp.provider.Shutdown(ctx)
}

func (tp *TracerProvider) GetTracerProvider() *trace.TracerProvider {
	return tp.provider
}
