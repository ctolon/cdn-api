package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
)

// StdLoggerAdapter is a logger implementation, which sends all logs to provided standard output.
type StdLoggerAdapter struct {
	ErrorLogger *log.Logger
	InfoLogger  *log.Logger
	DebugLogger *log.Logger
	TraceLogger *log.Logger

	fields LogFields
}

// NewStdLogger creates StdLoggerAdapter which sends all logs to stderr.
func NewStdLogger(debug, trace bool) LoggerAdapter {
	return NewStdLoggerWithOut(os.Stderr, debug, trace)
}

// NewStdLoggerWithOut creates StdLoggerAdapter which sends all logs to provided io.Writer.
func NewStdLoggerWithOut(out io.Writer, debug bool, trace bool) LoggerAdapter {
	l := log.New(out, "[watermill] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	a := &StdLoggerAdapter{InfoLogger: l, ErrorLogger: l}

	if debug {
		a.DebugLogger = l
	}
	if trace {
		a.TraceLogger = l
	}

	return a
}

func (l *StdLoggerAdapter) Error(msg string, err error, fields LogFields) {
	l.log(l.ErrorLogger, "ERROR", msg, fields.Add(LogFields{"err": err}))
}

func (l *StdLoggerAdapter) Info(msg string, fields LogFields) {
	l.log(l.InfoLogger, "INFO ", msg, fields)
}

func (l *StdLoggerAdapter) Debug(msg string, fields LogFields) {
	l.log(l.DebugLogger, "DEBUG", msg, fields)
}

func (l *StdLoggerAdapter) Trace(msg string, fields LogFields) {
	l.log(l.TraceLogger, "TRACE", msg, fields)
}

func (l *StdLoggerAdapter) With(fields LogFields) LoggerAdapter {
	return &StdLoggerAdapter{
		ErrorLogger: l.ErrorLogger,
		InfoLogger:  l.InfoLogger,
		DebugLogger: l.DebugLogger,
		TraceLogger: l.TraceLogger,
		fields:      l.fields.Add(fields),
	}
}

func (l *StdLoggerAdapter) log(logger *log.Logger, level string, msg string, fields LogFields) {
	if logger == nil {
		return
	}

	fieldsStr := ""

	allFields := l.fields.Add(fields)

	keys := make([]string, len(allFields))
	i := 0
	for field := range allFields {
		keys[i] = field
		i++
	}

	sort.Strings(keys)

	for _, key := range keys {
		var valueStr string
		value := allFields[key]

		if stringer, ok := value.(fmt.Stringer); ok {
			valueStr = stringer.String()
		} else {
			valueStr = fmt.Sprintf("%v", value)
		}

		if strings.Contains(valueStr, " ") {
			valueStr = `"` + valueStr + `"`
		}

		fieldsStr += key + "=" + valueStr + " "
	}

	_ = logger.Output(3, fmt.Sprintf("\t"+`level=%s msg="%s" %s`, level, msg, fieldsStr))
}
