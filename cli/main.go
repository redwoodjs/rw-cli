package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/redwoodjs/rw-cli/cli/cmd"
)

// NOTE: These variables are set at compile time via ldflags by goreleaser
var (
	version = "unknown"
	commit  = "unknown"
	date    = "unknown"
)

var logFile *os.File

const (
	RW_DIR_NAME = ".rw"
)

func main() {
	// Forward the version information to the cmd package
	cmd.BuildVersion = version
	cmd.BuildCommit = commit
	cmd.BuildDate = date

	err := ensureDotRWExists()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = setupDebugLogger()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer teardownDebugLogger()

	cmd.Execute()
}

func ensureDotRWExists() error {
	uDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dRWDir := filepath.Join(uDir, RW_DIR_NAME)
	if _, err := os.Stat(dRWDir); os.IsNotExist(err) {
		err = os.MkdirAll(dRWDir, 0755)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func setupDebugLogger() error {
	uDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Open a file for logging
	logFile, err = os.OpenFile(filepath.Join(uDir, RW_DIR_NAME, "debug.json"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	// Configure the logger
	logger := slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	slog.Debug("Start", slog.Time("at", time.Now()))

	return nil
}

func teardownDebugLogger() error {
	slog.Debug("Stop", slog.Time("at", time.Now()))

	if logFile == nil {
		return nil
	}

	// If the file gets too big (>1MB), delete it
	fileInfo, err := logFile.Stat()
	if err != nil {
		return err
	}
	if fileInfo.Size() > 1_000_000 {
		err = os.Remove(logFile.Name())
		if err != nil {
			return err
		}
	}

	// Close the file
	err = logFile.Close()
	if err != nil {
		return err
	}

	return nil
}
