package monitoring

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogFormat represents the output format for logs
type LogFormat int

const (
	FormatJSON LogFormat = iota
	FormatText
	FormatConsole
)

// StructuredLogger provides enhanced logging capabilities for production use
type StructuredLogger struct {
	logger    *slog.Logger
	level     LogLevel
	format    LogFormat
	fields    map[string]any
	component string
}

// LoggerConfig configures the structured logger
type LoggerConfig struct {
	Level     LogLevel
	Format    LogFormat
	Output    io.Writer
	Component string
	Fields    map[string]any
}

// NewStructuredLogger creates a new structured logger with the given configuration
func NewStructuredLogger(config LoggerConfig) *StructuredLogger {
	if config.Output == nil {
		config.Output = os.Stdout
	}

	if config.Fields == nil {
		config.Fields = make(map[string]any)
	}

	var handler slog.Handler

	// Configure log level
	var slogLevel slog.Level
	switch config.Level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	case LevelFatal:
		slogLevel = slog.LevelError + 4 // Custom fatal level
	}

	opts := &slog.HandlerOptions{
		Level:     slogLevel,
		AddSource: config.Level == LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339Nano))
			}
			return a
		},
	}

	// Configure output format
	switch config.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(config.Output, opts)
	case FormatText:
		handler = slog.NewTextHandler(config.Output, opts)
	case FormatConsole:
		handler = NewConsoleHandler(config.Output, opts)
	default:
		handler = slog.NewJSONHandler(config.Output, opts)
	}

	logger := slog.New(handler)

	// Add default fields
	if config.Component != "" {
		config.Fields["component"] = config.Component
	}
	config.Fields["service"] = "encx"
	config.Fields["version"] = "1.0.0" // TODO: Get from build info

	return &StructuredLogger{
		logger:    logger,
		level:     config.Level,
		format:    config.Format,
		fields:    config.Fields,
		component: config.Component,
	}
}

// WithFields returns a new logger with additional fields
func (l *StructuredLogger) WithFields(fields map[string]any) *StructuredLogger {
	newFields := make(map[string]any)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &StructuredLogger{
		logger:    l.logger,
		level:     l.level,
		format:    l.format,
		fields:    newFields,
		component: l.component,
	}
}

// WithContext returns a new logger with context information
func (l *StructuredLogger) WithContext(ctx context.Context) *StructuredLogger {
	fields := make(map[string]any)
	for k, v := range l.fields {
		fields[k] = v
	}

	// Extract common context values
	if traceID := ctx.Value("trace_id"); traceID != nil {
		fields["trace_id"] = traceID
	}
	if spanID := ctx.Value("span_id"); spanID != nil {
		fields["span_id"] = spanID
	}
	if userID := ctx.Value("user_id"); userID != nil {
		fields["user_id"] = userID
	}
	if requestID := ctx.Value("request_id"); requestID != nil {
		fields["request_id"] = requestID
	}

	return l.WithFields(fields)
}

// Debug logs a debug level message
func (l *StructuredLogger) Debug(msg string, args ...any) {
	if l.level > LevelDebug {
		return
	}
	l.log(context.Background(), LevelDebug, msg, args...)
}

// Info logs an info level message
func (l *StructuredLogger) Info(msg string, args ...any) {
	if l.level > LevelInfo {
		return
	}
	l.log(context.Background(), LevelInfo, msg, args...)
}

// Warn logs a warning level message
func (l *StructuredLogger) Warn(msg string, args ...any) {
	if l.level > LevelWarn {
		return
	}
	l.log(context.Background(), LevelWarn, msg, args...)
}

// Error logs an error level message
func (l *StructuredLogger) Error(msg string, args ...any) {
	if l.level > LevelError {
		return
	}
	l.log(context.Background(), LevelError, msg, args...)
}

// Fatal logs a fatal level message and exits
func (l *StructuredLogger) Fatal(msg string, args ...any) {
	l.log(context.Background(), LevelFatal, msg, args...)
	os.Exit(1)
}

// log is the internal logging method
func (l *StructuredLogger) log(ctx context.Context, level LogLevel, msg string, args ...any) {
	// Convert to slog level
	var slogLevel slog.Level
	switch level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	case LevelFatal:
		slogLevel = slog.LevelError + 4
	}

	// Create logger with fields
	logger := l.logger
	for k, v := range l.fields {
		logger = logger.With(k, v)
	}

	// Add caller information for errors and above
	if level >= LevelError {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			logger = logger.With("caller", fmt.Sprintf("%s:%d", filepath.Base(file), line))
		}
	}

	// Log the message
	logger.Log(ctx, slogLevel, fmt.Sprintf(msg, args...))
}

// LogCryptoOperation logs a crypto operation with standard fields
func (l *StructuredLogger) LogCryptoOperation(ctx context.Context, operation string, duration time.Duration, err error, metadata map[string]any) {
	fields := map[string]any{
		"operation":   operation,
		"duration":    duration.String(),
		"duration_ms": duration.Nanoseconds() / 1000000,
	}

	// Add metadata
	for k, v := range metadata {
		fields[k] = v
	}

	logger := l.WithContext(ctx).WithFields(fields)

	if err != nil {
		fields["error"] = err.Error()
		fields["error_type"] = fmt.Sprintf("%T", err)
		logger.Error("Crypto operation failed")
	} else {
		logger.Info("Crypto operation completed")
	}
}

// LogKeyOperation logs a key management operation
func (l *StructuredLogger) LogKeyOperation(ctx context.Context, operation string, keyAlias string, keyVersion int, metadata map[string]any) {
	fields := map[string]any{
		"operation":   operation,
		"key_alias":   keyAlias,
		"key_version": keyVersion,
	}

	// Add metadata
	for k, v := range metadata {
		fields[k] = v
	}

	logger := l.WithContext(ctx).WithFields(fields)
	logger.Info("Key operation performed")
}

// LogSecurityEvent logs a security-related event
func (l *StructuredLogger) LogSecurityEvent(ctx context.Context, event string, severity string, metadata map[string]any) {
	fields := map[string]any{
		"event":     event,
		"severity":  severity,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	}

	// Add metadata
	for k, v := range metadata {
		fields[k] = v
	}

	logger := l.WithContext(ctx).WithFields(fields)

	switch severity {
	case "critical", "high":
		logger.Error("Security event detected")
	case "medium":
		logger.Warn("Security event detected")
	default:
		logger.Info("Security event detected")
	}
}

// ConsoleHandler provides colorized console output
type ConsoleHandler struct {
	handler slog.Handler
	output  io.Writer
}

// NewConsoleHandler creates a new console handler
func NewConsoleHandler(output io.Writer, opts *slog.HandlerOptions) *ConsoleHandler {
	return &ConsoleHandler{
		handler: slog.NewTextHandler(output, opts),
		output:  output,
	}
}

func (h *ConsoleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *ConsoleHandler) Handle(ctx context.Context, record slog.Record) error {
	// Add colors for console output
	var levelStr string
	switch record.Level {
	case slog.LevelDebug:
		levelStr = "\033[36mDEBUG\033[0m" // Cyan
	case slog.LevelInfo:
		levelStr = "\033[32mINFO\033[0m" // Green
	case slog.LevelWarn:
		levelStr = "\033[33mWARN\033[0m" // Yellow
	case slog.LevelError:
		levelStr = "\033[31mERROR\033[0m" // Red
	default:
		if record.Level >= slog.LevelError+4 {
			levelStr = "\033[35mFATAL\033[0m" // Magenta
		} else {
			levelStr = record.Level.String()
		}
	}

	// Format timestamp
	timestamp := record.Time.Format("15:04:05.000")

	// Create custom record with colored level
	fmt.Fprintf(h.output, "%s [%s] %s", timestamp, levelStr, record.Message)

	// Add attributes
	record.Attrs(func(a slog.Attr) bool {
		if a.Key != slog.TimeKey && a.Key != slog.LevelKey {
			fmt.Fprintf(h.output, " %s=%s", a.Key, a.Value)
		}
		return true
	})

	fmt.Fprintln(h.output)
	return nil
}

func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ConsoleHandler{
		handler: h.handler.WithAttrs(attrs),
		output:  h.output,
	}
}

func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	return &ConsoleHandler{
		handler: h.handler.WithGroup(name),
		output:  h.output,
	}
}

// ProductionLogger creates a logger optimized for production use
func NewProductionLogger(component string) *StructuredLogger {
	level := LevelInfo
	if levelStr := os.Getenv("ENCX_LOG_LEVEL"); levelStr != "" {
		switch levelStr {
		case "debug":
			level = LevelDebug
		case "info":
			level = LevelInfo
		case "warn":
			level = LevelWarn
		case "error":
			level = LevelError
		}
	}

	format := FormatJSON
	if formatStr := os.Getenv("ENCX_LOG_FORMAT"); formatStr != "" {
		switch formatStr {
		case "json":
			format = FormatJSON
		case "text":
			format = FormatText
		case "console":
			format = FormatConsole
		}
	}

	return NewStructuredLogger(LoggerConfig{
		Level:     level,
		Format:    format,
		Component: component,
		Fields: map[string]any{
			"pid": os.Getpid(),
		},
	})
}

// DevelopmentLogger creates a logger optimized for development
func NewDevelopmentLogger(component string) *StructuredLogger {
	return NewStructuredLogger(LoggerConfig{
		Level:     LevelDebug,
		Format:    FormatConsole,
		Component: component,
		Fields: map[string]any{
			"pid": os.Getpid(),
		},
	})
}
