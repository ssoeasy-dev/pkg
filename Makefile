.PHONY: test test-unit test-integration lint

# Переменная PKG — относительный путь к модулю (например, ./s3)
# По умолчанию пустая — тогда тестируются все модули.
PKG ?=

# Функция для поиска всех директорий, содержащих go.mod
define find_modules
	@find . -maxdepth 2 -name go.mod -exec dirname {} \; | sed 's|^\./||' | sort
endef

# Функция для выполнения команды в каждом модуле
define run_in_modules
	@modules="$(call find_modules)"; \
	if [ -z "$$modules" ]; then \
		echo "No Go modules found"; \
		exit 1; \
	fi; \
	for mod in $$modules; do \
		echo "Running: $(1) in $$mod"; \
		(cd $$mod && $(2)) || exit 1; \
	done
endef

# Unit-тесты
test-unit:
	@if [ -n "$(PKG)" ]; then \
		echo "Running unit tests in $(PKG)"; \
		cd $(PKG) && go test -v -race ./...; \
	else \
		$(call run_in_modules,unit tests,go test -v -race ./...); \
	fi

# Интеграционные тесты (требуют Docker)
test-integration:
	@if [ -n "$(PKG)" ]; then \
		echo "Running integration tests in $(PKG)"; \
		cd $(PKG) && go test -v -race -tags=integration ./...; \
	else \
		$(call run_in_modules,integration tests,go test -v -race -tags=integration ./...); \
	fi

# Все тесты
test: test-unit test-integration

# Линтер
lint:
	@if [ -n "$(PKG)" ]; then \
		echo "Running linter in $(PKG)"; \
		cd $(PKG) && golangci-lint run ./...; \
	else \
		$(call run_in_modules,linter,golangci-lint run ./...); \
	fi