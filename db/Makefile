.PHONY: test test-unit test-integration

# Unit тесты (без БД, без тега integration)
test-unit:
	go test -v -race ./...

# Интеграционные тесты (требуют Docker)
test-integration:
	go test -v -race -tags=integration ./...

# Всё вместе
test: test-unit test-integration