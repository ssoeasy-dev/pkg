#!/bin/bash
# scripts/update-deps.sh
# Обновляет внутренние зависимости в go.mod всех модулей монорепозитория.
# Использование:
#   update-deps.sh --mode=(beta|stable) [pkg1 version1] [pkg2 version2] ...
#
# Режимы:
#   beta   - заменяет ВСЕ dev-версии на соответствующие beta-версии.
#   stable - заменяет beta-версии указанных пакетов на стабильные.

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

# Настройка Go для приватных модулей
go env -w GOPRIVATE=github.com/ssoeasy-dev/*
go env -w GONOPROXY=github.com/ssoeasy-dev/*
go env -w GONOSUMDB=github.com/ssoeasy-dev/*

ALL_PKGS=($(bash scripts/list-packages.sh))
PREFIX="github.com/ssoeasy-dev/pkg/"

update_pkg() {
    local pkg="$1"
    local version="$2"
    local module_path="${PREFIX}${pkg}"
    local pattern

    if [[ "$MODE" == "beta" ]]; then
        pattern="v[0-9]+\.[0-9]+\.[0-9]+-dev-[^[:space:]]+"
        echo "Обновление $module_path до $version (dev → beta)"
    else
        pattern="v[0-9]+\.[0-9]+\.[0-9]+-beta\.[0-9]+"
        echo "Обновление $module_path до стабильной $version (beta → stable)"
    fi

    for mod in "${ALL_PKGS[@]}"; do
        modfile="${mod}/go.mod"
        [[ ! -f "$modfile" ]] && continue
        if grep -qE "${module_path}[[:space:]]+${pattern}" "$modfile"; then
            echo "  В модуле $mod"
            sed -i -E "s|(${module_path}[[:space:]]+)${pattern}|\1${version}|g" "$modfile"
        fi
    done
}

if [[ "$MODE" == "beta" ]]; then
    echo "=== Режим beta: замена ВСЕХ dev-версий на beta ==="
    # В режиме beta мы обновляем все переданные пакеты до указанных beta-версий
    while [[ $# -ge 2 ]]; do
        update_pkg "$1" "$2"
        shift 2
    done
    if [[ $# -eq 1 ]]; then
        echo "Ошибка: пропущена версия для пакета $1" >&2
        exit 1
    fi
else
    echo "=== Режим stable: замена beta-версий указанных пакетов на stable ==="
    while [[ $# -ge 2 ]]; do
        update_pkg "$1" "$2"
        shift 2
    done
    if [[ $# -eq 1 ]]; then
        echo "Ошибка: пропущена версия для пакета $1" >&2
        exit 1
    fi
fi

# Очищаем go.sum, кэш модулей и генерируем заново
for mod in "${ALL_PKGS[@]}"; do
    if [[ -f "${mod}/go.mod" ]]; then
        echo "  Полная очистка и обновление $mod"
        (cd "$mod" && rm -f go.sum && go mod tidy)
        git add "${mod}/go.mod" "${mod}/go.sum" 2>/dev/null || true
    fi
done

if ! git diff --cached --quiet; then
    if [[ "$MODE" == "beta" ]]; then
        git commit -m "chore(deps): update all dev dependencies to beta versions"
    else
        git commit -m "chore(deps): update beta dependencies to stable versions"
    fi
    echo "Изменения закоммичены."
else
    echo "Обновлений зависимостей не требуется."
fi
