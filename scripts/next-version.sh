#!/bin/bash
set -e

PKG="$1"
HEAD_REF="$2"

# Находим последний стабильный тег пакета
LAST_STABLE=$(git tag -l "${PKG}/v*" | grep -E "^${PKG}/v[0-9]+\.[0-9]+\.[0-9]+$" | sort -V | tail -n1)

if [ -z "$LAST_STABLE" ]; then
  CURRENT_VERSION="0.0.0"
  START_REF=$(git rev-list --max-parents=0 HEAD)
else
  CURRENT_VERSION="${LAST_STABLE#${PKG}/v}"
  # Находим общего предка между стабильным тегом и HEAD
  START_REF=$(git merge-base "$LAST_STABLE" "$HEAD_REF" 2>/dev/null || echo "$LAST_STABLE")
fi

# Коммиты от START_REF до HEAD, затрагивающие пакет
COMMITS=$(git log "${START_REF}..${HEAD_REF}" --format="%s" -- "${PKG}/" 2>/dev/null || true)

if [ -z "$COMMITS" ]; then
  echo "$CURRENT_VERSION"
  exit 0
fi

# Определяем тип bump
BUMP_TYPE="patch"
if echo "$COMMITS" | grep -qiE "major\(${PKG}\)|BREAKING CHANGE"; then
  BUMP_TYPE="major"
elif echo "$COMMITS" | grep -qiE "(feat|minor)\(${PKG}\)"; then
  BUMP_TYPE="minor"
elif echo "$COMMITS" | grep -qi "fix(${PKG})"; then
  BUMP_TYPE="patch"
fi

# Вычисляем следующую версию
if command -v semver &>/dev/null; then
  NEXT_VERSION=$(semver bump "$BUMP_TYPE" "$CURRENT_VERSION")
else
  IFS='.' read -r MAJOR MINOR PATCH <<<"$CURRENT_VERSION"
  case $BUMP_TYPE in
    major) NEXT_VERSION="$((MAJOR+1)).0.0" ;;
    minor) NEXT_VERSION="${MAJOR}.$((MINOR+1)).0" ;;
    patch) NEXT_VERSION="${MAJOR}.${MINOR}.$((PATCH+1))" ;;
    *)     NEXT_VERSION="$CURRENT_VERSION" ;;
  esac
fi

echo "$NEXT_VERSION"
