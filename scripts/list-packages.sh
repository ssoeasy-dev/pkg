#!/bin/bash
# scripts/list-packages.sh
# Выводит список всех Go-пакетов в монорепозитории (директории первого уровня с go.mod)
# Использование: list-packages.sh [--json]

set -e

cd "$(git rev-parse --show-toplevel)" || exit 1

packages=()
for dir in */; do
    pkg="${dir%/}"
    if [[ -f "${pkg}/go.mod" ]]; then
        packages+=("$pkg")
    fi
done

if [[ "$1" == "--json" ]]; then
    printf '%s\n' "${packages[@]}" | jq -R . | jq -s -c .
else
    printf '%s\n' "${packages[@]}"
fi
