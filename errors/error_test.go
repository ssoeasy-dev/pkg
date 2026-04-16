package errors

import (
	stderrors "errors"
	"fmt"
	"testing"
)

// Вспомогательные sentinel-ошибки для тестов.
var (
	errTestKind  = stderrors.New("test kind")
	errOtherKind = stderrors.New("other kind")
)

// TestNew проверяет создание корневой ошибки через New.
func TestNew(t *testing.T) {
	msg := "technical details"
	err := New(errTestKind, msg)

	// Error() должен вернуть только строку вида.
	if got := err.Error(); got != errTestKind.Error() {
		t.Errorf("Error() = %q, want %q", got, errTestKind.Error())
	}

	// FullError() должен вернуть техническое сообщение.
	if got := FullError(err); got != msg {
		t.Errorf("FullError() = %q, want %q", got, msg)
	}

	// Kind должен быть равен переданному sentinel.
	if got := Kind(err); got != errTestKind {
		t.Errorf("Kind() = %v, want %v", got, errTestKind)
	}
}

// TestNewf проверяет форматированное создание корневой ошибки.
func TestNewf(t *testing.T) {
	err := Newf(errTestKind, "user %d not found", 42)
	wantFull := "user 42 not found"

	if FullError(err) != wantFull {
		t.Errorf("FullError() = %q, want %q", FullError(err), wantFull)
	}
	if err.Error() != errTestKind.Error() {
		t.Errorf("Error() = %q, want %q", err.Error(), errTestKind.Error())
	}
}

// TestWrap проверяет оборачивание обычной ошибки с добавлением публичного сообщения.
func TestWrap(t *testing.T) {
	cause := stderrors.New("connection refused")
	wrapped := Wrap(cause, "failed to fetch user")

	// Error() показывает только публичное сообщение (без причины).
	wantPublic := "failed to fetch user"
	if got := wrapped.Error(); got != wantPublic {
		t.Errorf("Error() = %q, want %q", got, wantPublic)
	}

	// FullError() включает техническую причину.
	wantFull := "failed to fetch user: connection refused"
	if got := FullError(wrapped); got != wantFull {
		t.Errorf("FullError() = %q, want %q", got, wantFull)
	}

	// Kind обёрнутой ошибки должен быть ErrUnknown, так как причина не является *Error.
	if got := Kind(wrapped); got != ErrUnknown {
		t.Errorf("Kind() = %v, want %v", got, ErrUnknown)
	}

	// Unwrap возвращает исходную причину.
	if got := Unwrap(wrapped); got != cause {
		t.Errorf("Unwrap() = %v, want %v", got, cause)
	}
}

// TestWrap_Nil проверяет, что Wrap(nil, ...) возвращает nil.
func TestWrap_Nil(t *testing.T) {
	if err := Wrap(nil, "msg"); err != nil {
		t.Errorf("Wrap(nil) should return nil, got %v", err)
	}
}

// TestWrapf проверяет форматированное оборачивание.
func TestWrapf(t *testing.T) {
	cause := stderrors.New("timeout")
	wrapped := Wrapf(cause, "request to %s failed", "api.example.com")

	wantPublic := "request to api.example.com failed"
	if got := wrapped.Error(); got != wantPublic {
		t.Errorf("Error() = %q, want %q", got, wantPublic)
	}
}

// TestNewWrap проверяет создание ошибки с явным видом и причиной.
func TestNewWrap(t *testing.T) {
	cause := stderrors.New("db error")
	err := NewWrap(errTestKind, cause, "user creation failed")

	// Публичное сообщение.
	if got := err.Error(); got != "user creation failed" {
		t.Errorf("Error() = %q, want %q", got, "user creation failed")
	}

	// Полная ошибка включает причину.
	wantFull := "user creation failed: db error"
	if got := FullError(err); got != wantFull {
		t.Errorf("FullError() = %q, want %q", got, wantFull)
	}

	// Kind должен быть установлен явно.
	if got := Kind(err); got != errTestKind {
		t.Errorf("Kind() = %v, want %v", got, errTestKind)
	}

	// Is должен соответствовать виду.
	if !Is(err, errTestKind) {
		t.Error("Is(err, errTestKind) should be true")
	}
}

// TestNewWrapf проверяет форматированный NewWrap.
func TestNewWrapf(t *testing.T) {
	cause := stderrors.New("not found")
	err := NewWrapf(errTestKind, cause, "item %d missing", 123)

	wantPublic := "item 123 missing"
	if got := err.Error(); got != wantPublic {
		t.Errorf("Error() = %q, want %q", got, wantPublic)
	}
	if !Is(err, errTestKind) {
		t.Error("Is(err, errTestKind) should be true")
	}
}

// TestWithKind проверяет изменение вида ошибки.
func TestWithKind(t *testing.T) {
	t.Run("on *Error", func(t *testing.T) {
		original := New(errTestKind, "original msg")
		modified := WithKind(original, errOtherKind)

		if Kind(modified) != errOtherKind {
			t.Errorf("Kind not changed, got %v", Kind(modified))
		}
		// Публичное сообщение теперь содержит строку нового вида.
		if modified.Error() != errOtherKind.Error() {
			t.Errorf("Error() = %q, want %q", modified.Error(), errOtherKind.Error())
		}
		// FullError сохраняет исходное техническое сообщение.
		if FullError(modified) != "original msg" {
			t.Errorf("FullError() = %q, want %q", FullError(modified), "original msg")
		}
	})

	t.Run("on plain error", func(t *testing.T) {
		plain := stderrors.New("plain error")
		modified := WithKind(plain, errTestKind)

		if Kind(modified) != errTestKind {
			t.Errorf("Kind not set, got %v", Kind(modified))
		}
		if modified.Error() != errTestKind.Error() {
			t.Errorf("Error() = %q, want %q", modified.Error(), errTestKind.Error())
		}
		if FullError(modified) != "plain error" {
			t.Errorf("FullError() = %q, want %q", FullError(modified), "plain error")
		}
	})
}

// TestKind проверяет извлечение вида из различных ошибок.
func TestKind(t *testing.T) {
	t.Run("root *Error", func(t *testing.T) {
		err := New(errTestKind, "msg")
		if Kind(err) != errTestKind {
			t.Errorf("expected %v, got %v", errTestKind, Kind(err))
		}
	})

	t.Run("wrapped *Error", func(t *testing.T) {
		root := New(errTestKind, "msg")
		wrapped := Wrap(root, "outer")
		if Kind(wrapped) != errTestKind {
			t.Errorf("expected %v, got %v", errTestKind, Kind(wrapped))
		}
	})

	t.Run("plain error", func(t *testing.T) {
		plain := stderrors.New("plain")
		if Kind(plain) != ErrUnknown {
			t.Errorf("expected ErrUnknown, got %v", Kind(plain))
		}
	})

	t.Run("nil", func(t *testing.T) {
		if Kind(nil) != ErrUnknown {
			t.Errorf("expected ErrUnknown, got %v", Kind(nil))
		}
	})
}

// TestIs проверяет работу Is с видами ошибок.
func TestIs(t *testing.T) {
	err := New(errTestKind, "msg")
	if !Is(err, errTestKind) {
		t.Error("Is should be true")
	}
	if Is(err, errOtherKind) {
		t.Error("Is should be false")
	}
}

// TestAs проверяет извлечение *Error через As.
func TestAs(t *testing.T) {
	err := New(errTestKind, "msg")
	var e *Error
	if !As(err, &e) {
		t.Error("As should succeed")
	}
	if e.Kind() != errTestKind {
		t.Errorf("extracted kind = %v, want %v", e.Kind(), errTestKind)
	}
}

// TestUnwrap проверяет разворачивание цепочки ошибок.
func TestUnwrap(t *testing.T) {
	root := New(errTestKind, "root")
	outer := Wrap(root, "outer")
	outermost := Wrap(outer, "outermost")

	unwrapped := Unwrap(outermost)
	if unwrapped != outer {
		t.Errorf("Unwrap(outermost) should return outer")
	}
	unwrapped = Unwrap(unwrapped)
	if unwrapped != root {
		t.Errorf("second Unwrap should return root")
	}
	unwrapped = Unwrap(unwrapped)
	if unwrapped != nil {
		t.Errorf("third Unwrap should return nil")
	}
}

// TestFullError проверяет получение полного сообщения для разных типов ошибок.
func TestFullError(t *testing.T) {
	t.Run("plain error", func(t *testing.T) {
		plain := stderrors.New("boom")
		if FullError(plain) != "boom" {
			t.Errorf("FullError(plain) = %q, want %q", FullError(plain), "boom")
		}
	})

	t.Run("nil", func(t *testing.T) {
		// FullError(nil) вызывает панику, так как err.Error() для nil недопустим.
		// FullError(nil) возвращает пустую строку для безопасного логирования.
		if got := FullError(nil); got != "" {
			t.Errorf("FullError(nil) = %q, want empty string", got)
		}
	
	})
}

// TestError_Chain_PublicMessages проверяет цепочку публичных сообщений.
func TestError_Chain_PublicMessages(t *testing.T) {
	root := New(errTestKind, "technical root cause")
	level1 := Wrap(root, "failed to process request")
	level2 := Wrap(level1, "user service unavailable")

	// Ожидаемая публичная цепочка: "user service unavailable: failed to process request: test kind"
	want := "user service unavailable: failed to process request: " + errTestKind.Error()
	if got := level2.Error(); got != want {
		t.Errorf("Error() chain = %q, want %q", got, want)
	}

	// Полная ошибка включает техническую причину.
	wantFull := "user service unavailable: failed to process request: technical root cause"
	if got := FullError(level2); got != wantFull {
		t.Errorf("FullError() chain = %q, want %q", got, wantFull)
	}
}

// TestError_Chain_PlainCause проверяет цепочку, где причина — обычная ошибка.
func TestError_Chain_PlainCause(t *testing.T) {
	cause := stderrors.New("sql: connection closed")
	wrapped := Wrap(cause, "database query failed")

	// Публичное сообщение останавливается на обычной ошибке.
	want := "database query failed"
	if got := wrapped.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	// FullError включает текст обычной ошибки.
	wantFull := "database query failed: sql: connection closed"
	if got := FullError(wrapped); got != wantFull {
		t.Errorf("FullError() = %q, want %q", got, wantFull)
	}
}

// Примеры использования функций (также используются в тестах как Example-функции).

func ExampleNew() {
	err := New(ErrNotFound, "user with id 42 not found in database")
	fmt.Println(err.Error())
	fmt.Println(FullError(err))
	// Output:
	// not found
	// user with id 42 not found in database
}

func ExampleWrap() {
	cause := stderrors.New("connection refused")
	err := Wrap(cause, "failed to fetch user")
	fmt.Println(err.Error())
	fmt.Println(FullError(err))
	// Output:
	// failed to fetch user
	// failed to fetch user: connection refused
}

func ExampleNewWrap() {
	cause := stderrors.New("duplicate key value violates unique constraint")
	err := NewWrap(ErrConflict, cause, "user already exists")
	fmt.Println(err.Error())
	fmt.Println(FullError(err))
	fmt.Println(Is(err, ErrConflict))
	// Output:
	// user already exists
	// user already exists: duplicate key value violates unique constraint
	// true
}
