package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// Init configures the global OpenTelemetry TracerProvider. When OTEL_ENABLED is
// not "true" the noop provider is installed, keeping zero runtime cost.
// Returns a shutdown function that callers must invoke before exit.
func Init(ctx context.Context, serviceName string, enabled bool, logger *slog.Logger) (func(context.Context) error, error) {
	if !enabled {
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
		logger.Info("otel tracer set to noop", "service", serviceName)
		return func(context.Context) error { return nil }, nil
	}

	exp, err := stdouttrace.New(stdouttrace.WithWriter(os.Stdout), stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))),
	)
	otel.SetTracerProvider(tp)
	logger.Info("otel tracer enabled", "service", serviceName, "exporter", "stdout")
	return tp.Shutdown, nil
}

func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
