package rmq

import (
	"errors"
	"fmt"
)

// Sentinel errors для errors.Is. Оригинальная причина доступна через errors.Unwrap.
var (
	// ErrInvalidConfig возвращается при неполной или некорректной конфигурации.
	ErrInvalidConfig = errors.New("rmq: invalid configuration")
	// ErrConnect возвращается при невозможности установить или восстановить соединение.
	ErrConnect = errors.New("rmq: connection failed")
	// ErrPublish возвращается при ошибке публикации сообщения в exchange.
	ErrPublish = errors.New("rmq: publish failed")
	// ErrStopped возвращается при попытке операции на остановленном consumer-е.
	ErrStopped = errors.New("rmq: consumer stopped")
)

// rmqError — минимальная обёртка: sentinel для errors.Is, причина для errors.Unwrap.
//
//	errors.Is(err, ErrConnect)  → true (через sentinel)
//	errors.Unwrap(err)          → оригинальная amqp-ошибка (для логирования)
type rmqError struct {
	msg      string
	sentinel error
	cause    error
}

func (e *rmqError) Error() string        { return e.msg }
func (e *rmqError) Is(target error) bool { return target == e.sentinel }
func (e *rmqError) Unwrap() error        { return e.cause }

// newError создаёт rmqError. Если cause != nil, его текст включается в сообщение.
func newError(sentinel error, msg string, cause error) error {
	if cause != nil {
		return &rmqError{
			msg:      fmt.Sprintf("%s: %v", msg, cause),
			sentinel: sentinel,
			cause:    cause,
		}
	}
	return &rmqError{msg: msg, sentinel: sentinel}
}
