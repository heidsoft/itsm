package eventbus

import (
	"github.com/ThreeDotsLabs/watermill"
	"go.uber.org/zap"
)

// ZapLoggerAdapter adapts zap.SugaredLogger to watermill.LoggerAdapter interface
type ZapLoggerAdapter struct {
	logger *zap.SugaredLogger
	fields watermill.LogFields
}

// NewZapLoggerAdapter creates a new ZapLoggerAdapter
func NewZapLoggerAdapter(logger *zap.SugaredLogger) *ZapLoggerAdapter {
	return &ZapLoggerAdapter{
		logger: logger,
		fields: watermill.LogFields{},
	}
}

// Error logs an error message.
func (l *ZapLoggerAdapter) Error(msg string, err error, fields watermill.LogFields) {
	allFields := l.mergeFields(fields)
	if err != nil {
		allFields["error"] = err
	}
	l.logger.Errorw(msg, toKeyValueSlice(allFields)...)
}

// Info logs an info message
func (l *ZapLoggerAdapter) Info(msg string, fields watermill.LogFields) {
	allFields := l.mergeFields(fields)
	l.logger.Infow(msg, toKeyValueSlice(allFields)...)
}

// Debug logs a debug message
func (l *ZapLoggerAdapter) Debug(msg string, fields watermill.LogFields) {
	allFields := l.mergeFields(fields)
	l.logger.Debugw(msg, toKeyValueSlice(allFields)...)
}

// Trace logs a trace message (mapped to debug)
func (l *ZapLoggerAdapter) Trace(msg string, fields watermill.LogFields) {
	allFields := l.mergeFields(fields)
	l.logger.Debugw(msg, toKeyValueSlice(allFields)...)
}

// With returns a new logger with additional fields
func (l *ZapLoggerAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	newFields := l.mergeFields(fields)
	return &ZapLoggerAdapter{
		logger: l.logger,
		fields: newFields,
	}
}

// mergeFields merges the existing fields with new fields
func (l *ZapLoggerAdapter) mergeFields(newFields watermill.LogFields) watermill.LogFields {
	merged := make(watermill.LogFields, len(l.fields)+len(newFields))
	for k, v := range l.fields {
		merged[k] = v
	}
	for k, v := range newFields {
		merged[k] = v
	}
	return merged
}

// toKeyValueSlice converts LogFields to a slice of key-value pairs
func toKeyValueSlice(fields watermill.LogFields) []interface{} {
	kv := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		kv = append(kv, k, v)
	}
	return kv
}
