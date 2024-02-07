package logging

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/redwoodjs/rw-cli/cli/config"
)

func SetupDebugLogger() error {
	uDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Open a file for logging
	logFile, err := os.OpenFile(filepath.Join(uDir, config.RW_DIR_NAME, "debug.json"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	config.LogFile = logFile

	// Configure the logger
	logger := slog.New(slog.NewJSONHandler(config.LogFile, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	slog.Debug("Start", slog.Time("at", time.Now()))

	return nil
}

func TeardownDebugLogger() error {
	fmt.Println("TeardownDebugLogger")
	slog.Debug("Stop", slog.Time("at", time.Now()))

	if config.LogFile == nil {
		return nil
	}

	// If the file gets too big (>1MB), delete it
	fileInfo, err := config.LogFile.Stat()
	if err != nil {
		return err
	}
	if fileInfo.Size() > 1_000_000 {
		err = os.Remove(config.LogFile.Name())
		if err != nil {
			return err
		}
	}

	// Close the file
	err = config.LogFile.Close()
	if err != nil {
		return err
	}

	return nil
}
