#!/bin/bash
# scripts/update-deps.sh
# Обновляет внутренние зависимости в go.mod всех модулей монорепозитория.
# Использование:
#   update-deps.sh --mode=(beta|stable) [pkg1 version1] [pkg2 version2] ...
#
# Режимы:
#   beta   - заменяет dev-версии (содержат "-dev-") на указанную beta-версию.
#   stable - заменяет beta-версии (содержат "-beta.") на указанную стабильную версию.

set -e

MODE=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --mode=*)
            MODE="${1#*=}"
            shift
            ;;
        --mode)
            MODE="$2"
            shift 2
            ;;
        *)
            break
            ;;
    esac
done

if [[ "$MODE" != "beta" && "$MODE" != "stable" ]]; then
    echo "Ошибка: --mode должен быть 'beta' или 'stable'"
    exit 1
fi

# Получаем список всех пакетов
ALL_PKGS=($(bash scripts/list-packages.sh))

update_pkg() {
    local pkg="$1"
    local version="$2"
    local module_path="github.com/ssoeasy-dev/pkg/${pkg}"
    local pattern

    if [[ "$MODE" == "beta" ]]; then
        pattern=".*-dev-"
        echo "Обновление ${module_path} до ${version} (dev → beta)"
    else
        pattern=".*-beta\\."
        echo "Обновление ${module_path} до стабильной ${version} (beta → stable)"
    fi

    for mod in "${ALL_PKGS[@]}"; do
        if [[ ! -f "${mod}/go.mod" ]]; then
            continue
        fi
        # Проверяем, что модуль зависит от пакета и версия соответствует шаблону
        if grep -qE "${module_path}[[:space:]]+${pattern}" "${mod}/go.mod"; then
            echo "  Обновление ${mod}"
            (cd "$mod" && go get "${module_path}@${version}")
        fi
    done
}

# Обрабатываем аргументы парами (пакет версия)
while [[ $# -gt 0 ]]; do
    update_pkg "$1" "$2"
    shift 2
done

# Выполняем go mod tidy во всех модулях
for mod in "${ALL_PKGS[@]}"; do
    if [[ -f "${mod}/go.mod" ]]; then
        (cd "$mod" && go mod tidy)
        git add "${mod}/go.mod" "${mod}/go.sum" 2>/dev/null || true
    fi
done

# Коммитим изменения, если они есть
if ! git diff --cached --quiet; then
    if [[ "$MODE" == "beta" ]]; then
        git commit -m "chore(deps): update dev dependencies to beta versions"
    else
        git commit -m "chore(deps): update beta dependencies to stable versions"
    fi
    echo "Изменения закоммичены."
else
    echo "Обновлений зависимостей не требуется."
fi
