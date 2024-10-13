package logger

// watermill
import (
	"context"
	"log/slog"
	"os"

	"github.com/rs/zerolog"
)

// LogLevel is a type for log levels.
type LogLevel uint

const (

	// TraceLogLevel is the lowest log level.
	TraceLogLevel LogLevel = iota + 1

	// DebugLogLevel is a log level for debugging.
	DebugLogLevel

	// InfoLogLevel is a log level for informational messages.
	InfoLogLevel

	// ErrorLogLevel is a log level for errors.
	ErrorLogLevel
)

// LogFields is the logger's key-value list of fields.
type LogFields map[string]interface{}

// Add adds new fields to the list of LogFields.
func (l LogFields) Add(newFields LogFields) LogFields {
	resultFields := make(LogFields, len(l)+len(newFields))

	for field, value := range l {
		resultFields[field] = value
	}
	for field, value := range newFields {
		resultFields[field] = value
	}

	return resultFields
}

// Copy copies the LogFields.
func (l LogFields) Copy() LogFields {
	cpy := make(LogFields, len(l))
	for k, v := range l {
		cpy[k] = v
	}

	return cpy
}

// LoggerType represents the type of logger.
type LoggerType int

const (

	// ZapLogger represents the zap logger.
	ZapLogger LoggerType = iota

	// LogrusLogger represents the logrus logger.
	LogrusLogger

	// ZerologLogger represents the zerolog logger.
	ZerologLogger

	// StructedLogger represents the structured logger.
	StructedLogger

	// StdLogger represents the standard logger.
	StdLogger
)

// String returns the string representation of the LoggerType.
func (lt LoggerType) String() string {
	return [...]string{"zap", "logrus", "zerolog", "structured", "std"}[lt]
}

// LoggerAdapter is an interface, that you need to implement to support logging.
// You can use StdLoggerAdapter as a reference implementation.
type LoggerAdapter interface {
	Error(msg string, err error, fields LogFields)
	Info(msg string, fields LogFields)
	Debug(msg string, fields LogFields)
	Trace(msg string, fields LogFields)
	With(fields LogFields) LoggerAdapter
}

// NewLoggerAdapter creates a new logger wrapper.
func NewLoggerAdapter(loggerType LoggerType, ctx context.Context) LoggerAdapter {

	switch loggerType {
	case ZerologLogger:
		logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006/01/02 15:04:05"}).Level(zerolog.InfoLevel).With().Timestamp().Logger()
		adapter := NewZerologLoggerAdapter(logger)
		return adapter
	case StdLogger:
		logger := NewStdLoggerWithOut(os.Stderr, true, true)
		return logger
	case StructedLogger:
		logger := &slog.Logger{}
		adapter := NewSlogLogger(logger)
		return adapter
	default:
		return NopLogger{}
	}

}

// NopLogger is a logger which discards all logs.
type NopLogger struct{}

func (NopLogger) Error(msg string, err error, fields LogFields) {}
func (NopLogger) Info(msg string, fields LogFields)             {}
func (NopLogger) Debug(msg string, fields LogFields)            {}
func (NopLogger) Trace(msg string, fields LogFields)            {}
func (l NopLogger) With(fields LogFields) LoggerAdapter         { return l }
