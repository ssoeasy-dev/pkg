package errors

import (
	"context"
	"errors"
)

// Универсальные виды ошибок, покрывающие как семантику HTTP, так и gRPC.
// При маппинге на конкретный протокол выбирается наиболее подходящий код;
// менее детализированные протоколы схлопывают несколько видов в один.
var (
	// ErrCanceled — операция отменена (обычно клиентом).
	ErrCanceled = context.Canceled

	// ErrUnknown — неизвестная ошибка.
	ErrUnknown = errors.New("unknown error")

	// ErrInvalidArgument — некорректные аргументы.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrDeadlineExceeded — превышен дедлайн.
	ErrDeadlineExceeded = context.DeadlineExceeded

	// ErrNotFound — ресурс не найден.
	ErrNotFound = errors.New("not found")

	// ErrPermissionDenied — недостаточно прав.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrResourceExhausted — исчерпаны ресурсы.
	ErrResourceExhausted = errors.New("resource exhausted")

	// ErrFailedPrecondition — нарушено предусловие.
	ErrFailedPrecondition = errors.New("failed precondition")

	// ErrUnimplemented — не реализовано.
	ErrUnimplemented = errors.New("unimplemented")

	// ErrInternal — внутренняя ошибка сервера.
	ErrInternal = errors.New("internal error")

	// ErrUnavailable — сервис недоступен.
	ErrUnavailable = errors.New("unavailable")

	// ErrUnauthenticated — отсутствует аутентификация.
	ErrUnauthenticated = errors.New("unauthenticated")

	// ErrAlreadyExists — ресурс уже существует.
	ErrAlreadyExists = errors.New("already exists")

	// ErrAborted — операция прервана.
	ErrAborted = errors.New("aborted")

	// ErrDataLoss — потеря данных.
	ErrDataLoss = errors.New("data loss")

	// ErrPaymentRequired — требуется оплата.
	ErrPaymentRequired = errors.New("payment required")

	// ErrMethodNotAllowed — метод не поддерживается.
	ErrMethodNotAllowed = errors.New("method not allowed")

	// ErrNotAcceptable — не удаётся удовлетворить Accept.
	ErrNotAcceptable = errors.New("not acceptable")

	// ErrRequestTimeout — клиент не отправил запрос вовремя.
	ErrRequestTimeout = errors.New("request timeout")

	// ErrConflict — конфликт состояния.
	ErrConflict = errors.New("conflict")

	// ErrGone — ресурс удалён навсегда.
	ErrGone = errors.New("gone")

	// ErrPayloadTooLarge — тело запроса слишком большое.
	ErrPayloadTooLarge = errors.New("payload too large")

	// ErrURITooLong — URI слишком длинный.
	ErrURITooLong = errors.New("uri too long")

	// ErrUnsupportedMediaType — Content-Type не поддерживается.
	ErrUnsupportedMediaType = errors.New("unsupported media type")

	// ErrRangeNotSatisfiable — запрошенный диапазон невыполним.
	ErrRangeNotSatisfiable = errors.New("range not satisfiable")

	// ErrExpectationFailed — ожидание из Expect не выполнено.
	ErrExpectationFailed = errors.New("expectation failed")

	// ErrUnprocessableEntity — семантическая ошибка валидации.
	ErrUnprocessableEntity = errors.New("unprocessable entity")

	// ErrLocked — ресурс заблокирован.
	ErrLocked = errors.New("locked")

	// ErrFailedDependency — зависимый запрос не выполнен.
	ErrFailedDependency = errors.New("failed dependency")

	// ErrTooEarly — сервер не готов обработать запрос.
	ErrTooEarly = errors.New("too early")

	// ErrUpgradeRequired — требуется обновление протокола.
	ErrUpgradeRequired = errors.New("upgrade required")

	// ErrPreconditionRequired — требуется заголовок If-Match.
	ErrPreconditionRequired = errors.New("precondition required")

	// ErrTooManyRequests — превышен лимит запросов.
	ErrTooManyRequests = errors.New("too many requests")

	// ErrRequestHeaderFieldsTooLarge — заголовки слишком большие.
	ErrRequestHeaderFieldsTooLarge = errors.New("request header fields too large")

	// ErrUnavailableForLegalReasons — недоступно по юридическим причинам.
	ErrUnavailableForLegalReasons = errors.New("unavailable for legal reasons")

	// ErrBadGateway — ошибка шлюза.
	ErrBadGateway = errors.New("bad gateway")

	// ErrGatewayTimeout — таймаут шлюза.
	ErrGatewayTimeout = errors.New("gateway timeout")

	// ErrHTTPVersionNotSupported — версия HTTP не поддерживается.
	ErrHTTPVersionNotSupported = errors.New("http version not supported")

	// ErrVariantAlsoNegotiates — ошибка согласования контента.
	ErrVariantAlsoNegotiates = errors.New("variant also negotiates")

	// ErrInsufficientStorage — недостаточно места.
	ErrInsufficientStorage = errors.New("insufficient storage")

	// ErrLoopDetected — обнаружен цикл.
	ErrLoopDetected = errors.New("loop detected")

	// ErrNotExtended — требуется расширение.
	ErrNotExtended = errors.New("not extended")

	// ErrNetworkAuthenticationRequired — нужна сетевая аутентификация.
	ErrNetworkAuthenticationRequired = errors.New("network authentication required")
)
