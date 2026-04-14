#!/bin/bash
# scripts/build-release-plan.sh
# Принимает список изменённых пакетов, парсит go.mod, строит граф зависимостей,
# выполняет топологическую сортировку и возвращает уровни в JSON.

set -e

cd "$(git rev-parse --show-toplevel)" || exit 1

CHANGED_PKGS=("$@")
PREFIX="github.com/ssoeasy-dev/pkg/"

# Получаем список всех пакетов через отдельный скрипт
ALL_PKGS=($(bash scripts/list-packages.sh))

# 2. Фильтруем изменённые пакеты (оставляем только те, что есть в ALL_PKGS)
declare -A is_changed
for pkg in "${CHANGED_PKGS[@]}"; do
    if [[ " ${ALL_PKGS[*]} " == *" $pkg "* ]]; then
        is_changed[$pkg]=1
    fi
done

# 3. Парсим go.mod каждого пакета и строим граф зависимостей среди изменённых пакетов
declare -A deps      # deps[pkg] = список внутренних зависимостей
declare -A indegree  # indegree[pkg] = количество входящих рёбер от других изменённых пакетов
declare -A adj       # adj[from] = список пакетов, зависящих от from (обратный граф)

for pkg in "${ALL_PKGS[@]}"; do
    modfile="${pkg}/go.mod"
    # Извлекаем все строки, начинающиеся с префикса нашего монорепозитория
    internal_deps=$(grep -E "^[[:space:]]*${PREFIX}" "$modfile" | \
                    sed -E "s|.*${PREFIX}([^[:space:]]+).*|\1|" | \
                    sort -u)
    deps[$pkg]="$internal_deps"
done

# 4. Строим граф только между изменёнными пакетами
for pkg in "${!is_changed[@]}"; do
    for dep in ${deps[$pkg]}; do
        if [[ -n "${is_changed[$dep]}" ]]; then
            # pkg зависит от dep (стрелка dep -> pkg)
            adj[$dep]="${adj[$dep]} $pkg"
            indegree[$pkg]=$((indegree[$pkg] + 1))
        fi
    done
done

# 5. Топологическая сортировка (алгоритм Кана)
queue=()
for pkg in "${!is_changed[@]}"; do
    if [[ ${indegree[$pkg]:-0} -eq 0 ]]; then
        queue+=("$pkg")
    fi
done

levels=()
while [[ ${#queue[@]} -gt 0 ]]; do
    level_pkgs=("${queue[@]}")
    # Сохраняем уровень как JSON-массив
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

# 6. Выводим результат
if [[ ${#levels[@]} -eq 0 ]]; then
    echo "[]"
else
    printf '%s\n' "${levels[@]}" | jq -s .
fi
