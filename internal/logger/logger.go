package logger

import (
	"io"
	"log/slog"
	"os"
)

// Logger wraps slog.Logger to provide structured logging
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration
type Config struct {
	Level   slog.Level
	Format  string // "json" or "text"
	Output  io.Writer
	Service string // service name for structured logging
}

// NewLogger creates a new structured logger
func NewLogger(cfg Config) *Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	if cfg.Service == "" {
		cfg.Service = "github-stars-notify"
	}

	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: false,
	}

	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(cfg.Output, opts)
	default:
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	logger := slog.New(handler)

	// Add service name to all log entries
	logger = logger.With("service", cfg.Service)

	return &Logger{Logger: logger}
}

// Default creates a logger with default settings
func Default() *Logger {
	return NewLogger(Config{
		Level:   slog.LevelInfo,
		Format:  "text",
		Output:  os.Stdout,
		Service: "github-stars-notify",
	})
}

// WithContext creates a logger with context-specific attributes
func (l *Logger) WithContext(keyvals ...interface{}) *Logger {
	return &Logger{Logger: l.With(keyvals...)}
}

// WithRepository creates a logger with repository-specific attributes
func (l *Logger) WithRepository(owner, repo string) *Logger {
	return &Logger{Logger: l.With("repo_owner", owner, "repo_name", repo)}
}

// WithComponent creates a logger with component-specific attributes
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{Logger: l.With("component", component)}
}

// WithError creates a logger with error context
func (l *Logger) WithError(err error) *Logger {
	return &Logger{Logger: l.With("error", err)}
}
