#!/bin/bash
# scripts/build-release-plan.sh
# Принимает список изменённых пакетов, парсит go.mod, строит граф зависимостей,
# выполняет топологическую сортировку и возвращает уровни в JSON.

set -e

cd "$(git rev-parse --show-toplevel)" || exit 1

CHANGED_PKGS=("$@")
PREFIX="github.com/ssoeasy-dev/pkg/"

# Получаем список всех пакетов через отдельный скрипт
mapfile -t ALL_PKGS < <(bash scripts/list-packages.sh)

# Фильтруем изменённые пакеты
declare -A is_changed
for pkg in "${CHANGED_PKGS[@]}"; do
    if [[ " ${ALL_PKGS[*]} " == *" $pkg "* ]]; then
        is_changed[$pkg]=1
    fi
done

# Парсим go.mod и строим граф
declare -A deps indegree adj

for pkg in "${ALL_PKGS[@]}"; do
    modfile="${pkg}/go.mod"
    internal_deps=$(sed -n '/^require/,/^)/p' "$modfile" | \
            grep -oE "${PREFIX}[^[:space:]]*" | \
            sed "s|${PREFIX}||" | sort -u)
    deps[$pkg]="$internal_deps"
done

# Строим граф только между изменёнными пакетами
for pkg in "${!is_changed[@]}"; do
    for dep in ${deps[$pkg]}; do
        if [[ -n "${is_changed[$dep]}" ]]; then
            adj[$dep]="${adj[$dep]} $pkg"
            indegree[$pkg]=$((indegree[$pkg] + 1))
        fi
    done
done

# Топологическая сортировка
queue=()
for pkg in "${!is_changed[@]}"; do
    if [[ ${indegree[$pkg]:-0} -eq 0 ]]; then
        queue+=("$pkg")
    fi
done

levels=()
while [[ ${#queue[@]} -gt 0 ]]; do
    level_pkgs=("${queue[@]}")
    level_json=$(printf '%s\n' "${level_pkgs[@]}" | jq -R . | jq -s -c .)
    levels+=("$level_json")

    new_queue=()
    for pkg in "${queue[@]}"; do
        for neighbor in ${adj[$pkg]}; do
            indegree[$neighbor]=$((indegree[$neighbor] - 1))
            if [[ ${indegree[$neighbor]} -eq 0 ]]; then
                new_queue+=("$neighbor")
            fi
        done
    done
    queue=("${new_queue[@]}")
done

# Вывод компактного JSON
if [[ ${#levels[@]} -eq 0 ]]; then
    echo "[]"
else
    printf '%s\n' "${levels[@]}" | jq -s -c .
fi
