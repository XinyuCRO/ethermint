package keeper

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "ethermint"
var JaegerCollectorURL = "http://localhost:14278/api/traces"

var Tracer trace.Tracer

func NewTracer() (trace.Tracer, func()) {
	aTracer, shutdown, err := InitTracer(serviceName, "instance")
	if err != nil {
		log.Fatal(err)
	}

	Tracer = aTracer

	return Tracer, shutdown
}

func InitTracer(serviceName, instanceName string) (trace.Tracer, func(), error) {

	tp, err := newTracerProvider(serviceName, instanceName)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't initialize tracer provider: %w", err)
	}

	otel.SetTracerProvider(tp)

	// Cleanly shutdown and flush telemetry when the application exits.
	shutdown := func() {
		// Do not make the application hang when it is shutdown.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}

	return tp.Tracer(serviceName), shutdown, err
}

func newTracerProvider(serviceName, instanceName string) (*tracesdk.TracerProvider, error) {
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(JaegerCollectorURL)))
	if err != nil {
		return nil, err
	}
	return tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in a Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("service-instance", instanceName),
		)),
	), nil
}
