package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/redwoodjs/rw-cli/cli/cmd"
	"github.com/redwoodjs/rw-cli/cli/config"
	"github.com/redwoodjs/rw-cli/cli/files"
	"github.com/redwoodjs/rw-cli/cli/logging"
	"github.com/redwoodjs/rw-cli/cli/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

// NOTE: These variables are set at compile time via ldflags by goreleaser
var (
	version = "unknown"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Forward the version information to the cmd package
	cmd.BuildVersion = version
	cmd.BuildCommit = commit
	cmd.BuildDate = date
	cmd.BuildDate = date

	// TODO(jgmw): Support disabling telemetry via an environment variable
	// telemetryEnabled := os.Getenv("REDWOOD_DISABLE_TELEMETRY") == ""

	// Handle SIGINT gracefully for the telemetry
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Set up OpenTelemetry
	otelShutdown, err := telemetry.SetupOTelSDK(ctx)
	if err != nil {
		slog.Error("Error setting up telemetry", slog.String("error", err.Error()))
	}
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
		if err != nil {
			slog.Error("Error during telemetry shutdown", slog.String("error", err.Error()))
		}
	}()

	// Start the root span
	tracer := otel.GetTracerProvider().Tracer("rw-cli")
	rootSpanCtx, span := tracer.Start(ctx, "main")
	defer span.End()
	config.RootSpanCtx = &rootSpanCtx

	// We require a .rw directory to exist in the user's home directory
	err = files.EnsureDotRWExists()
	if err != nil {
		fmt.Println(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
		os.Exit(1)
	}

	// We have a debug logger which writes detail logs to a local file. The logs
	// are highly detailed and unredacted so are only kept locally on the user's machine.
	err = logging.SetupDebugLogger()
	if err != nil {
		fmt.Println(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
		os.Exit(1)
	}
	// NOTE: We may wish to reenable this. It was disabled because the OTel error handler needed to log
	// to the debug logger, but the debug logger was being closed before the OTel error handler was called.
	// defer logging.TeardownDebugLogger()
	defer func() {
		slog.Debug("End")
	}()

	// The root command handles errors itself, so we don't need to do anything
	cmd.Execute()
}
