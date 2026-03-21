package logger

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// capture возвращает logger, пишущий в буфер, и сам буфер.
func capture(env Environment) (*bytes.Buffer, Logger) {
	var buf bytes.Buffer
	return &buf, newLoggerWithWriter(env, "", &buf)
}

// ─── Environment ──────────────────────────────────────────────────────────────

func TestEnvironment_IsVerbose(t *testing.T) {
	tests := []struct {
		env      Environment
		expected bool
	}{
		{EnvironmentDevelopment, true},
		{EnvironmentLocal, true},
		{EnvironmentProduction, false},
		{EnvironmentTest, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.env), func(t *testing.T) {
			if got := tt.env.IsVerbose(); got != tt.expected {
				t.Errorf("IsVerbose() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ─── Info / Warn / Error всегда пишут вывод ───────────────────────────────────

func TestLogger_Info_WritesOutput(t *testing.T) {
	buf, log := capture(EnvironmentProduction)
	log.Info(context.Background(), "hello", nil)
	assertContains(t, buf.String(), "[INFO] hello")
}

func TestLogger_Warn_WritesOutput(t *testing.T) {
	buf, log := capture(EnvironmentProduction)
	log.Warn(context.Background(), "watch out", nil)
	assertContains(t, buf.String(), "[WARN] watch out")
}

func TestLogger_Error_WritesOutput(t *testing.T) {
	buf, log := capture(EnvironmentProduction)
	log.Error(context.Background(), "boom", nil)
	assertContains(t, buf.String(), "[ERROR] boom")
}

// ─── Debug: включён только в dev/local ───────────────────────────────────────

func TestLogger_Debug_WrittenInDevelopment(t *testing.T) {
	buf, log := capture(EnvironmentDevelopment)
	log.Debug(context.Background(), "dev detail", nil)
	assertContains(t, buf.String(), "[DEBUG] dev detail")
}

func TestLogger_Debug_WrittenInLocal(t *testing.T) {
	buf, log := capture(EnvironmentLocal)
	log.Debug(context.Background(), "local detail", nil)
	assertContains(t, buf.String(), "[DEBUG] local detail")
}

func TestLogger_Debug_SuppressedInProduction(t *testing.T) {
	buf, log := capture(EnvironmentProduction)
	log.Debug(context.Background(), "secret", nil)
	if buf.Len() != 0 {
		t.Errorf("expected no output in production, got: %q", buf.String())
	}
}

func TestLogger_Debug_SuppressedInTest(t *testing.T) {
	buf, log := capture(EnvironmentTest)
	log.Debug(context.Background(), "test detail", nil)
	if buf.Len() != 0 {
		t.Errorf("expected no output in test env, got: %q", buf.String())
	}
}

// ─── Контекст: enrichFromContext ──────────────────────────────────────────────

func TestLogger_Context_TraceID_Included(t *testing.T) {
	buf, log := capture(EnvironmentProduction)
	ctx := context.WithValue(context.Background(), TraceIDKey, "abc-123")
	log.Info(ctx, "msg", nil)
	assertContains(t, buf.String(), "trace_id=abc-123")
}

func TestLogger_Context_RequestID_Included(t *testing.T) {
	buf, log := capture(EnvironmentProduction)
	ctx := context.WithValue(context.Background(), RequestIDKey, "req-456")
	log.Info(ctx, "msg", nil)
	assertContains(t, buf.String(), "request_id=req-456")
}

func TestLogger_Context_BothIDs_IncludedAndOrdered(t *testing.T) {
	buf, log := capture(EnvironmentProduction)
	ctx := context.WithValue(context.Background(), TraceIDKey, "t1")
	ctx = context.WithValue(ctx, RequestIDKey, "r1")
	log.Info(ctx, "msg", nil)

	out := buf.String()
	assertContains(t, out, "trace_id=t1")
	assertContains(t, out, "request_id=r1")

	// trace_id всегда перед request_id
	tiPos := strings.Index(out, "trace_id")
	riPos := strings.Index(out, "request_id")
	if tiPos >= riPos {
		t.Errorf("expected trace_id before request_id, got: %q", out)
	}
}

func TestLogger_NilContext_DoesNotPanic(t *testing.T) {
	_, log := capture(EnvironmentProduction)
	// nil context не паникует
	log.Info(nil, "msg", nil) //nolint:staticcheck
}

func TestLogger_NilFields_DoesNotPanic(t *testing.T) {
	_, log := capture(EnvironmentProduction)
	log.Info(context.Background(), "msg", nil)
}

func TestLogger_EmptyContext_NoSeparator(t *testing.T) {
	buf, log := capture(EnvironmentProduction)
	log.Info(context.Background(), "clean msg", nil)
	out := buf.String()
	if strings.Contains(out, " | ") {
		t.Errorf("expected no separator without fields, got: %q", out)
	}
}

// ─── Поля ─────────────────────────────────────────────────────────────────────

func TestLogger_Fields_IncludedInOutput(t *testing.T) {
	buf, log := capture(EnvironmentProduction)
	log.Info(context.Background(), "msg", map[string]any{"user_id": 42})
	assertContains(t, buf.String(), "user_id=42")
}

func TestLogger_Fields_SeparatorPresentWhenFieldsExist(t *testing.T) {
	buf, log := capture(EnvironmentProduction)
	log.Info(context.Background(), "msg", map[string]any{"x": 1})
	assertContains(t, buf.String(), " | ")
}

func TestLogger_Fields_DeterministicOrder(t *testing.T) {
	// Запускаем несколько раз — порядок полей должен быть стабильным.
	capture(EnvironmentProduction)
	fields := map[string]any{"z": 1, "a": 2, "m": 3}

	results := make([]string, 5)
	for i := range results {
		var buf bytes.Buffer
		l := newLoggerWithWriter(EnvironmentProduction, "", &buf)
		l.Info(context.Background(), "msg", fields)
		results[i] = buf.String()
	}
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("non-deterministic output: %q vs %q", results[0], results[i])
		}
	}
}

func TestLogger_Fields_SortedAfterTraceAndRequest(t *testing.T) {
	buf, log := capture(EnvironmentProduction)
	ctx := context.WithValue(context.Background(), TraceIDKey, "t")
	log.Info(ctx, "msg", map[string]any{"z_field": 1, "a_field": 2})

	out := buf.String()
	// trace_id идёт первым, затем a_field, затем z_field
	tiPos := strings.Index(out, "trace_id")
	aPos := strings.Index(out, "a_field")
	zPos := strings.Index(out, "z_field")

	if !(tiPos < aPos && aPos < zPos) {
		t.Errorf("unexpected field order: %q", out)
	}
}

// ─── enrichFromContext: не мутирует оригинальный map ─────────────────────────

func TestEnrichFromContext_DoesNotMutateOriginalMap(t *testing.T) {
	original := map[string]any{"key": "val"}
	ctx := context.WithValue(context.Background(), TraceIDKey, "t1")

	enriched := enrichFromContext(ctx, original)

	if _, ok := original["trace_id"]; ok {
		t.Error("enrichFromContext mutated the original map")
	}
	if enriched["trace_id"] != "t1" {
		t.Errorf("enriched map missing trace_id, got: %v", enriched)
	}
}

func TestEnrichFromContext_NoContextValues_ReturnsSameMap(t *testing.T) {
	original := map[string]any{"key": "val"}
	result := enrichFromContext(context.Background(), original)

	// Без контекстных значений возвращается тот же самый map (не копия).
	if &result == nil || result["key"] != "val" {
		t.Error("expected original map to be returned unchanged")
	}
}

func TestEnrichFromContext_NilContext_ReturnsSameMap(t *testing.T) {
	original := map[string]any{"key": "val"}
	result := enrichFromContext(nil, original) //nolint:staticcheck
	if result["key"] != "val" {
		t.Errorf("expected original map for nil ctx, got: %v", result)
	}
}

func TestEnrichFromContext_NilMap_WithContextValues(t *testing.T) {
	ctx := context.WithValue(context.Background(), TraceIDKey, "x")
	result := enrichFromContext(ctx, nil)
	if result["trace_id"] != "x" {
		t.Errorf("expected trace_id in enriched nil map, got: %v", result)
	}
}

// ─── formatFields ─────────────────────────────────────────────────────────────

func TestFormatFields_TraceAndRequestFirst(t *testing.T) {
	fields := map[string]any{
		"trace_id":   "t",
		"request_id": "r",
		"z":          1,
		"a":          2,
	}
	out := formatFields(fields)

	if !strings.HasPrefix(out, "trace_id=t request_id=r") {
		t.Errorf("expected trace_id and request_id first, got: %q", out)
	}
}

func TestFormatFields_OnlyTraceID(t *testing.T) {
	out := formatFields(map[string]any{"trace_id": "t", "foo": "bar"})
	if !strings.HasPrefix(out, "trace_id=t") {
		t.Errorf("expected trace_id first, got: %q", out)
	}
}

func TestFormatFields_OnlyRequestID(t *testing.T) {
	out := formatFields(map[string]any{"request_id": "r", "foo": "bar"})
	if !strings.HasPrefix(out, "request_id=r") {
		t.Errorf("expected request_id first, got: %q", out)
	}
}

func TestFormatFields_SortsRemainingKeys(t *testing.T) {
	out := formatFields(map[string]any{"z": 3, "a": 1, "m": 2})
	expected := "a=1 m=2 z=3"
	if out != expected {
		t.Errorf("got %q, want %q", out, expected)
	}
}

func TestFormatFields_EmptyMap(t *testing.T) {
	out := formatFields(map[string]any{})
	if out != "" {
		t.Errorf("expected empty string, got: %q", out)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}
