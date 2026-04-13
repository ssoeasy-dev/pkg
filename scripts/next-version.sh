#!/bin/bash
set -e

PKG="$1"
BASE_REF="$2"
HEAD_REF="$3"

# 1. Получаем коммиты, которые есть в PR, но ещё не в develop
COMMITS=$(git log "${BASE_REF}..${HEAD_REF}" --format="%s" -- "${PKG}/" 2>/dev/null || true)

# 2. Находим последний стабильный тег пакета
LAST_STABLE=$(git tag -l "${PKG}/v*" | grep -E "^${PKG}/v[0-9]+\.[0-9]+\.[0-9]+$" | sort -V | tail -n1)

if [ -z "$LAST_STABLE" ]; then
  CURRENT_VERSION="0.0.0"
else
  CURRENT_VERSION="${LAST_STABLE#${PKG}/v}"
fi

# 3. Если коммитов нет — версия не меняется
if [ -z "$COMMITS" ]; then
  echo "$CURRENT_VERSION"
  exit 0
fi

# 4. Определяем тип bump по сообщениям коммитов
BUMP_TYPE="patch"

if echo "$COMMITS" | grep -qiE "major\(${PKG}\)|BREAKING CHANGE"; then
  BUMP_TYPE="major"
elif echo "$COMMITS" | grep -qiE "(feat|minor)\(${PKG}\)"; then
  BUMP_TYPE="minor"
elif echo "$COMMITS" | grep -qi "fix(${PKG})"; then
  BUMP_TYPE="patch"
fi

# 5. Вычисляем следующую версию
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
