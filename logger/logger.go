package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
)

type ctxKey string

const (
	TraceIDKey   ctxKey = "trace_id"
	RequestIDKey ctxKey = "request_id"
)

// Environment определяет окружение запуска приложения.
type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
	EnvironmentTest        Environment = "test"
	EnvironmentLocal       Environment = "local"
)

// IsVerbose сообщает, включён ли Debug-вывод для данного окружения.
func (e Environment) IsVerbose() bool {
	return e == EnvironmentDevelopment || e == EnvironmentLocal
}

// Logger — интерфейс логгера для Go-микросервисов SSO Easy.
// Все методы принимают context для автоматического обогащения trace_id / request_id.
type Logger interface {
	Debug(ctx context.Context, msg string, fields map[string]any)
	Info(ctx context.Context, msg string, fields map[string]any)
	Warn(ctx context.Context, msg string, fields map[string]any)
	Error(ctx context.Context, msg string, fields map[string]any)
}

type logger struct {
	log         *log.Logger
	environment Environment
}

// NewLogger создаёт Logger, пишущий в stdout.
// prefix добавляется к каждой строке (например, "auth.svc").
// Debug выводится только для EnvironmentDevelopment и EnvironmentLocal.
func NewLogger(environment Environment, prefix string) Logger {
	return newLoggerWithWriter(environment, prefix, os.Stdout)
}

// newLoggerWithWriter создаёт logger с произвольным io.Writer.
// Используется в тестах для перехвата вывода.
func newLoggerWithWriter(environment Environment, prefix string, w io.Writer) Logger {
	p := prefix
	if p != "" {
		p += " "
	}
	return &logger{
		log:         log.New(w, p, log.LstdFlags|log.Lshortfile),
		environment: environment,
	}
}

func (l *logger) Debug(ctx context.Context, msg string, fields map[string]any) {
	if l.environment.IsVerbose() {
		l.write("DEBUG", msg, enrichFromContext(ctx, fields))
	}
}

func (l *logger) Info(ctx context.Context, msg string, fields map[string]any) {
	l.write("INFO", msg, enrichFromContext(ctx, fields))
}

func (l *logger) Warn(ctx context.Context, msg string, fields map[string]any) {
	l.write("WARN", msg, enrichFromContext(ctx, fields))
}

func (l *logger) Error(ctx context.Context, msg string, fields map[string]any) {
	l.write("ERROR", msg, enrichFromContext(ctx, fields))
}

func (l *logger) write(level, msg string, fields map[string]any) {
	output := fmt.Sprintf("[%s] %s", level, msg)
	if len(fields) > 0 {
		output += " | " + formatFields(fields)
	}
	_ = l.log.Output(3, output)
}

// enrichFromContext копирует fields и добавляет trace_id / request_id из ctx,
// если они присутствуют. Не мутирует оригинальный map.
// Если в ctx нет нужных значений — возвращает оригинальный map без копирования.
func enrichFromContext(ctx context.Context, fields map[string]any) map[string]any {
	if ctx == nil {
		return fields
	}

	traceID := ctx.Value(TraceIDKey)
	requestID := ctx.Value(RequestIDKey)

	if traceID == nil && requestID == nil {
		return fields
	}

	enriched := make(map[string]any, len(fields)+2)
	for k, v := range fields {
		enriched[k] = v
	}
	if traceID != nil {
		enriched["trace_id"] = traceID
	}
	if requestID != nil {
		enriched["request_id"] = requestID
	}
	return enriched
}

// formatFields сериализует поля в "k=v" пары.
// trace_id и request_id всегда идут первыми; остальные ключи сортируются.
func formatFields(fields map[string]any) string {
	pairs := make([]string, 0, len(fields))

	if v, ok := fields["trace_id"]; ok {
		pairs = append(pairs, fmt.Sprintf("trace_id=%v", v))
	}
	if v, ok := fields["request_id"]; ok {
		pairs = append(pairs, fmt.Sprintf("request_id=%v", v))
	}

	rest := make([]string, 0, len(fields))
	for k := range fields {
		if k != "trace_id" && k != "request_id" {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range rest {
		pairs = append(pairs, fmt.Sprintf("%s=%v", k, fields[k]))
	}

	return strings.Join(pairs, " ")
}
