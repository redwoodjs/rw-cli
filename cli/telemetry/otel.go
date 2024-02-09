package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/redwoodjs/rw-cli/cli/cmd"
	"github.com/redwoodjs/rw-cli/cli/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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

	// TODO(jgmw): Get shell name
	// TODO(jgmw): Get node version
	// TODO(jgmw): Get yarn version
	// TODO(jgmw): Get npm version
	// TODO(jgmw): Get vscode version
	// TODO(jgmw): Get memory size
	// TODO(jgmw): Get UID

	// Detecting environment information
	osName := runtime.GOOS
	osArch := runtime.GOARCH
	osVersion := "unknown"
	cpuCount := runtime.NumCPU()
	nodeEnv := os.Getenv("NODE_ENV")
	ciRedwood := os.Getenv("REDWOOD_CI") != ""
	ciIs := isCI()
	devEnv := getDevelopmentEnvironment()

	// TODO(jgmw): Get the resource information from the environment, os, cpu, etc.
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("rw-cli"),
			semconv.ServiceVersion(cmd.BuildVersion),
			semconv.OSName(osName),
			attribute.String("os.arch", osArch),
			attribute.String("os.version", osVersion),
			attribute.String("shell.name", "unknown"),
			attribute.String("node.version", "unknown"),
			attribute.String("yarn.version", "unknown"),
			attribute.String("npm.version", "unknown"),
			attribute.String("vscode.version", "unknown"),
			attribute.Int("cpu.count", cpuCount),
			attribute.Int("memory.gb", 0),
			attribute.String("env.node_env", nodeEnv),
			attribute.Bool("ci.redwood", ciRedwood),
			attribute.Bool("ci.isci", ciIs),
			attribute.String("dev.environment", devEnv),
			attribute.String("uid", "unknown"),
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

// Inspired from: https://github.com/watson/ci-info
func isCI() bool {
	// Explicitly check for "false" to bail out early
	if os.Getenv("CI") == "false" {
		return false
	}

	// Loop through a list of known CI flags
	flags := []string{
		// Generic flags
		"BUILD_ID",
		"BUILD_NUMBER",
		"CI",
		"CI_APP_ID",
		"CI_BUILD_ID",
		"CI_BUILD_NUMBER",
		"CI_NAME",
		"CONTINUOUS_INTEGRATION",
		"RUN_ID",
		// Specific flags
		"REDWOOD_CI",
		"AGOLA",
		"APPCIRCLE",
		"APPVEYOR",
		"CODEBUILD",
		"AZURE_PIPELINES",
		"BAMBOO",
		"BITBUCKET",
		"BITRISE",
		"BUDDY",
		"BUILDKITE",
		"CIRCLE",
		"CIRRUS",
		"CODEFRESH",
		"CODESHIP",
		"DRONE",
		"DSARI",
		"EARTHLY",
		"EAS",
		"GERRIT",
		"GITHUB_ACTIONS",
		"GITLAB",
		"GITEA_ACTIONS",
		"GOCD",
		"GOOGLE_CLOUD_BUILD",
		"HARNESS",
		"HEROKU",
		"HUDSON",
		"JENKINS",
		"LAYERCI",
		"MAGNUM",
		"NETLIFY",
		"NEVERCODE",
		"PROW",
		"RELEASEHUB",
		"RENDER",
		"SAIL",
		"SCREWDRIVER",
		"SEMAPHORE",
		"SOURCEHUT",
		"STRIDER",
		"TASKCLUSTER",
		"TEAMCITY",
		"TRAVIS",
		"VELA",
		"VERCEL",
		"APPCENTER",
		"WOODPECKER",
	}

	for _, flag := range flags {
		v := strings.ToLower(os.Getenv(flag))
		// We assume any value other than "false" indicates the flag is set
		if v != "" && v != "false" {
			return true
		}
	}

	// No known CI flag was found
	return false
}

func getDevelopmentEnvironment() string {
	// Check through all the env var keys
	for _, e := range os.Environ() {
		key := strings.SplitN(e, "=", 2)[0]

		// Gitpod
		if strings.HasPrefix(key, "GITPOD_") {
			return "gitpod"
		}
	}

	// No known development environment was found
	return ""
}
