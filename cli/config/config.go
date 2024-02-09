package config

import (
	"context"
	"os"
)

const (
	RW_DIR_NAME = ".rw"

	RW_OTEL_ENDPOINT = "https://quark.quantumparticle.io/v1/traces"
)

// ----------------------------------------------------------------

// NOTE: These variables are reflect the values in `main` which are injected at compile time
// via ldflags by goreleaser
var (
	Version = "unknown"
	Commit  = "unknown"
	Date    = "unknown"
)

// LogFile is the file to which debug logs are written
var LogFile *os.File

// RootSpanCtx is the root OTel span context for the main func
var RootSpanCtx *context.Context
