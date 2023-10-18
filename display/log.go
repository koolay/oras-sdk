package display

import (
	"context"
	"os"

	"golang.org/x/exp/slog"
)

type contextKey int

// loggerKey is the associated key type for logger entry in context.
const loggerKey contextKey = iota

// NewLogger returns a logger.
func NewLogger(
	ctx context.Context,
	debug bool,
) (context.Context, *slog.Logger) {
	var logLevel slog.Level
	if debug {
		logLevel = slog.LevelDebug
	} else {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	return context.WithValue(ctx, loggerKey, logger), logger
}

// Logger return the logger attached to context or the standard one.
func Logger(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerKey).(*slog.Logger)
	if !ok {
		panic("no logger attached to context")
	}
	return logger
}
