package logger

import (
	"github.com/rs/zerolog"
)

// ZerologLoggerAdapter is an adapter for the zerolog logger.
type ZerologLoggerAdapter struct {
	logger zerolog.Logger
}

func addFieldsData(event *zerolog.Event, fields LogFields) {
	for i, v := range fields {
		event.Interface(i, v)
	}
}

// Logs an error message.
func (loggerAdapter *ZerologLoggerAdapter) Error(msg string, err error, fields LogFields) {
	event := loggerAdapter.logger.Err(err)

	if fields != nil {
		addFieldsData(event, fields)
	}

	event.Msg(msg)
}

// Logs an info message.
func (loggerAdapter *ZerologLoggerAdapter) Info(msg string, fields LogFields) {
	event := loggerAdapter.logger.Info()

	if fields != nil {
		addFieldsData(event, fields)
	}

	event.Msg(msg)
}

// Logs a debug message.
func (loggerAdapter *ZerologLoggerAdapter) Debug(msg string, fields LogFields) {
	event := loggerAdapter.logger.Debug()

	if fields != nil {
		addFieldsData(event, fields)
	}

	event.Msg(msg)
}

// Logs a trace.
func (loggerAdapter *ZerologLoggerAdapter) Trace(msg string, fields LogFields) {
	event := loggerAdapter.logger.Trace()

	if fields != nil {
		addFieldsData(event, fields)
	}

	event.Msg(msg)
}

// Creates new adapter wiht the input fields as context.
func (loggerAdapter *ZerologLoggerAdapter) With(fields LogFields) LoggerAdapter {
	if fields == nil {
		return loggerAdapter
	}

	subLog := loggerAdapter.logger.With()

	for i, v := range fields {
		subLog = subLog.Interface(i, v)
	}

	return &ZerologLoggerAdapter{
		logger: subLog.Logger(),
	}
}

// Gets a new zerolog adapter for use in the watermill context.
// *ZerologLoggerAdapter
func NewZerologLoggerAdapter(logger zerolog.Logger) LoggerAdapter {
	return &ZerologLoggerAdapter{
		logger: logger,
	}
}
