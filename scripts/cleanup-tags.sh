#!/bin/bash
# scripts/cleanup-tags.sh
# Удаляет временные теги (dev, beta, all) для указанного пакета.
# Использование:
#   cleanup-tags.sh <package> <type> [branch-pattern]
#   type: dev, beta, all
#   branch-pattern: требуется только для dev (sanitized branch name)

set -e

PKG="$1"
TYPE="$2"
BRANCH_PATTERN="$3"

if [ -z "$PKG" ] || [ -z "$TYPE" ]; then
    echo "Usage: cleanup-tags.sh <package> <type> [branch-pattern]"
    exit 1
fi

case "$TYPE" in
    dev)
        if [ -z "$BRANCH_PATTERN" ]; then
            echo "Error: branch-pattern required for dev tags"
            exit 1
        fi
        PATTERN="${PKG}/v.*-dev-${BRANCH_PATTERN}\\.[0-9]+"
        ;;
    beta)
        PATTERN="${PKG}/v.*-beta\\.[0-9]+"
        ;;
    all)
        PATTERN="${PKG}/v.*-(dev-.*|beta\\.[0-9]+)"
        ;;
    *)
        echo "Unknown type: $TYPE (must be dev, beta, or all)"
        exit 1
        ;;
esac

TAGS=$(git tag -l | grep -E "^${PATTERN}$" || true)

if [ -n "$TAGS" ]; then
    for TAG in $TAGS; do
        echo "Deleting tag: $TAG"
        git push origin ":refs/tags/$TAG" || true
    done
    echo "Tags deleted successfully."
else
    echo "No matching tags found."
fi
