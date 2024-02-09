package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redwoodjs/rw-cli/cli/cmd"
	"github.com/redwoodjs/rw-cli/cli/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func SetupOTelSDK(ctx context.Context) (shutdown func(context.Context) error, err error) {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(inErr error) {
		slog.Error("otel error", slog.String("error", inErr.Error()))
	}))

	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	tracerProvider, err := newTraceProvider(ctx)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	return shutdown, err
}

func newTraceProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	// TODO(jgmw): Get the resource information from the environment, os, cpu, etc.
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("rw-cli"),
			semconv.ServiceVersion(cmd.BuildVersion),
			semconv.OSName(""),
			semconv.OSVersion(""),

			// 'shell.name': info.System?.Shell?.name,
			// 'node.version': info.Binaries?.Node?.version,
			// 'yarn.version': info.Binaries?.Yarn?.version,
			// 'npm.version': info.Binaries?.npm?.version,
			// 'vscode.version': info.IDEs?.VSCode?.version,
			// 'cpu.count': cpu.physicalCores,
			// 'memory.gb': Math.round(mem.total / 1073741824),
			// 'env.node_env': process.env.NODE_ENV || null,
			// 'ci.redwood': !!process.env.REDWOOD_CI,
			// 'ci.isci': ci.isCI,
			// 'dev.environment': developmentEnvironment,
			// uid: UID,
		),
	)
	if err != nil {
		return nil, err
	}

	// Trace exporter which sends telemetry to RedwoodJS
	remoteTE, err := otlptracehttp.New(ctx,
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		otlptracehttp.WithEndpointURL(config.RW_OTEL_ENDPOINT),
	)
	if err != nil {
		return nil, err
	}

	// Trace exporter which sends telemetry to the local debug log
	localTE := &SlogExporter{}

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(localTE),
		sdktrace.WithResource(r),
		sdktrace.WithBatcher(remoteTE, sdktrace.WithBatchTimeout(time.Second)),
	)
	return traceProvider, nil
}

// SlogExporter is a simple OpenTelemetry span exporter that logs spans to `slog.Debug`
type SlogExporter struct {
}

func (se *SlogExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	// We print out all spans, we specifically highlight some fields for visibility but include the
	// full span as `raw` anyway
	for _, span := range spans {
		slog.Debug("otel span",
			slog.String("name", span.Name()),
			slog.String("attributes", fmt.Sprintf("%+v", span.Attributes())),
			slog.String("spanId", fmt.Sprintf("%+v", span.SpanContext().SpanID())),
			slog.String("traceId", fmt.Sprintf("%+v", span.SpanContext().TraceID())),
			slog.String("parentSpanId", fmt.Sprintf("%+v", span.Parent().SpanID())),
			slog.String("status", fmt.Sprintf("%+v", span.Status())),
			slog.String("kind", fmt.Sprintf("%+v", span.SpanKind())),
			slog.String("startTime", fmt.Sprintf("%+v", span.StartTime())),
			slog.String("endTime", fmt.Sprintf("%+v", span.EndTime())),
			slog.String("events", fmt.Sprintf("%+v", span.Events())),
			slog.String("resource", fmt.Sprintf("%+v", span.Resource())),
			slog.String("raw", fmt.Sprintf("%+v", span)),
		)
	}
	return nil
}

func (se *SlogExporter) Shutdown(ctx context.Context) error {
	// We don't need to do anything to shut down the SlogExporter
	return nil
}
