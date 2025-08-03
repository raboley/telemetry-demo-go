package logging

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

type ContextLogger struct {
	*logrus.Logger
}

func NewLogger() *ContextLogger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	return &ContextLogger{Logger: logger}
}

func (l *ContextLogger) WithTracing(ctx context.Context) *logrus.Entry {
	entry := l.WithContext(ctx)
	
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		entry = entry.WithFields(logrus.Fields{
			"trace_id": spanCtx.TraceID().String(),
			"span_id":  spanCtx.SpanID().String(),
		})
	}
	
	return entry
}

func (l *ContextLogger) InfoWithTracing(ctx context.Context, msg string, fields logrus.Fields) {
	entry := l.WithTracing(ctx)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Info(msg)
}

func (l *ContextLogger) ErrorWithTracing(ctx context.Context, msg string, err error, fields logrus.Fields) {
	entry := l.WithTracing(ctx)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	if err != nil {
		entry = entry.WithError(err)
	}
	entry.Error(msg)
}

func (l *ContextLogger) WarnWithTracing(ctx context.Context, msg string, fields logrus.Fields) {
	entry := l.WithTracing(ctx)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Warn(msg)
}

func (l *ContextLogger) DebugWithTracing(ctx context.Context, msg string, fields logrus.Fields) {
	entry := l.WithTracing(ctx)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Debug(msg)
}