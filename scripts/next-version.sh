#!/bin/bash
set -e

PKG="$1"
BASE_REF="$2"
HEAD_REF="$3"

# Последний стабильный тег пакета (без пререлиза)
LAST_STABLE=$(git tag -l "${PKG}/v*" | grep -E "^${PKG}/v[0-9]+\.[0-9]+\.[0-9]+$" | sort -V | tail -n1)

if [ -z "$LAST_STABLE" ]; then
  CURRENT_VERSION="0.0.0"
  START_REF="$BASE_REF"
else
  CURRENT_VERSION="${LAST_STABLE#${PKG}/v}"
  # Определяем, что раньше: последний стабильный тег или BASE_REF (merge-base)
  if git merge-base --is-ancestor "$LAST_STABLE" "$BASE_REF" 2>/dev/null; then
    # LAST_STABLE предшествует BASE_REF → начинаем с BASE_REF
    START_REF="$BASE_REF"
  else
    # BASE_REF предшествует LAST_STABLE → начинаем с LAST_STABLE
    START_REF="$LAST_STABLE"
  fi
fi

# Коммиты от START_REF до HEAD, затрагивающие пакет
COMMITS=$(git log "${START_REF}..${HEAD_REF}" --format="%s" -- "${PKG}/" 2>/dev/null || true)

if [ -z "$COMMITS" ]; then
  echo "$CURRENT_VERSION"
  exit 0
fi

# Анализ маркеров
BUMP_TYPE="patch"  # по умолчанию patch, если есть изменения

if echo "$COMMITS" | grep -qiE "major\(${PKG}\)|BREAKING CHANGE"; then
  BUMP_TYPE="major"
elif echo "$COMMITS" | grep -qiE "(feat|minor)\(${PKG}\)"; then
  BUMP_TYPE="minor"
elif echo "$COMMITS" | grep -qi "fix(${PKG})"; then
  BUMP_TYPE="patch"
fi

# Вычисление следующей версии
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
