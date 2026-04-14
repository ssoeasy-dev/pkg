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

ALL_PKGS=($(bash scripts/list-packages.sh))
PREFIX="github.com/ssoeasy-dev/pkg/"

if [[ "$MODE" == "beta" ]]; then
    echo "=== Режим beta: замена ВСЕХ dev-версий на beta ==="

    # Собираем все dev-зависимости во всех go.mod
    declare -A dev_deps
    for mod in "${ALL_PKGS[@]}"; do
        modfile="${mod}/go.mod"
        if [[ ! -f "$modfile" ]]; then continue; fi

        grep -E "^[[:space:]]*${PREFIX}[^[:space:]]+ .*-dev-" "$modfile" | while read -r line; do
            pkg=$(echo "$line" | sed -E "s|.*${PREFIX}([^[:space:]]+).*|\1|")
            dev_deps[$pkg]="${dev_deps[$pkg]} $mod"
        done
    done

    for pkg in "${!dev_deps[@]}"; do
        beta_tag=$(git tag -l "${pkg}/v*-beta.*" | sort -V | tail -n1)
        if [[ -z "$beta_tag" ]]; then
            echo "⚠️  Для $pkg нет beta-тега, пропускаем"
            continue
        fi
        version="${beta_tag#${pkg}/v}"
        module_path="${PREFIX}${pkg}"
        echo "Обновление $module_path до $version"

        for mod in ${dev_deps[$pkg]}; do
            echo "  В модуле $mod"
            (cd "$mod" && go mod edit -require "${module_path}@v${version}")
        done
    done
else
    echo "=== Режим stable: замена beta-версий указанных пакетов на stable ==="
    while [[ $# -gt 0 ]]; do
        pkg="$1"
        version="$2"
        shift 2

        module_path="${PREFIX}${pkg}"
        echo "Обновление $module_path до стабильной $version"

        for mod in "${ALL_PKGS[@]}"; do
            modfile="${mod}/go.mod"
            if [[ ! -f "$modfile" ]]; then continue; fi
            if grep -qE "${module_path}[[:space:]]+.*-beta\\." "$modfile"; then
                echo "  В модуле $mod"
                (cd "$mod" && go mod edit -require "${module_path}@v${version}")
            fi
        done
    done
fi

# Очищаем go.sum и заново генерируем для всех модулей
for mod in "${ALL_PKGS[@]}"; do
    if [[ -f "${mod}/go.mod" ]]; then
        echo "  Очистка и обновление go.sum для $mod"
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
