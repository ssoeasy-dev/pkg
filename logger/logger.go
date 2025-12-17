package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
)

type ctxKey string

const (
	TraceIDKey ctxKey = "trace_id"
	RequestIDKey ctxKey = "request_id"
)

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
	EnvironmentTest        Environment = "test"
	EnvironmentLocal       Environment = "local"
)

type Logger struct {
	*log.Logger
	environment Environment
}

func NewLogger(environment Environment, prefix string) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, prefix, log.LstdFlags|log.Lshortfile),
		environment:  environment,
	}
}

func extractFromContext(ctx context.Context, fields map[string]any) map[string]any {
	if fields == nil {
		fields = make(map[string]any)
	}

	if ctx != nil {
		if traceID := ctx.Value(TraceIDKey); traceID != nil {
			fields["trace_id"] = traceID
		}
		if requestID := ctx.Value(RequestIDKey); requestID != nil {
			fields["request_id"] = requestID
		}
	}

	return fields
}

func (l *Logger) Debug(ctx context.Context, msg string, fields map[string]any) {
	if l.environment == EnvironmentDevelopment || l.environment == EnvironmentLocal {
		l.logWithFields("DEBUG", msg, extractFromContext(ctx, fields))
	}
}

func (l *Logger) Info(ctx context.Context, msg string, fields map[string]any) {
	l.logWithFields("INFO", msg, extractFromContext(ctx, fields))
}

func (l *Logger) Warn(ctx context.Context, msg string, fields map[string]any) {
	l.logWithFields("WARN", msg, extractFromContext(ctx, fields))
}

func (l *Logger) Error(ctx context.Context, msg string, fields map[string]any) {
	l.logWithFields("ERROR", msg, extractFromContext(ctx, fields))
}

func (l *Logger) logWithFields(level, msg string, fields map[string]any) {
	output := fmt.Sprintf("[%s] %s", level, msg)
	if len(fields) > 0 {
		output += " | " + formatFields(fields)
	}
	_ = l.Output(3, output)
}

func formatFields(fields map[string]any) string {
	pairs := make([]string, 0, len(fields))

	if v, ok := fields["trace_id"]; ok {
		pairs = append(pairs, fmt.Sprintf("trace_id=%v", v))
	}

	if v, ok := fields["request_id"]; ok {
		pairs = append(pairs, fmt.Sprintf("request_id=%v", v))
	}

	for k, v := range fields {
		if k == "trace_id" || k == "request_id" {
			continue
		}
		pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
	}

	return strings.Join(pairs, " ")
}
