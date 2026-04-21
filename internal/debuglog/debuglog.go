package debuglog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/toto/whoopy/internal/paths"
)

var (
	mu      sync.RWMutex
	logger  = discardLogger()
	logFile *os.File
	enabled bool
)

// Configure enables or disables structured debug logging.
func Configure(on bool) error {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}

	enabled = on
	if !on {
		logger = discardLogger()
		return nil
	}

	path, err := paths.DebugLogFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("prepare debug log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open debug log file: %w", err)
	}

	logFile = file
	logger = slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return nil
}

// Enabled reports whether debug logging is active.
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

// Path returns the configured debug log path.
func Path() (string, error) {
	return paths.DebugLogFile()
}

// Debug writes a debug-level entry.
func Debug(msg string, args ...any) {
	log(context.Background(), slog.LevelDebug, msg, args...)
}

// Info writes an info-level entry.
func Info(msg string, args ...any) {
	log(context.Background(), slog.LevelInfo, msg, args...)
}

// Warn writes a warn-level entry.
func Warn(msg string, args ...any) {
	log(context.Background(), slog.LevelWarn, msg, args...)
}

// Error writes an error-level entry.
func Error(msg string, args ...any) {
	log(context.Background(), slog.LevelError, msg, args...)
}

func log(ctx context.Context, level slog.Level, msg string, args ...any) {
	mu.RLock()
	current := logger
	mu.RUnlock()
	current.Log(ctx, level, msg, args...)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}
