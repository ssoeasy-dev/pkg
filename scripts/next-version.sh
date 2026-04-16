#!/bin/bash
set -e

PKG="$1"

# Находим последний стабильный тег пакета
LAST_STABLE=$(git tag -l "${PKG}/v*" | grep -E "^${PKG}/v[0-9]+\.[0-9]+\.[0-9]+$" | sort -V | tail -n1)

if [ -z "$LAST_STABLE" ]; then
  CURRENT_VERSION="0.0.0"
  START_REF=$(git rev-list --max-parents=0 HEAD)
else
  CURRENT_VERSION="${LAST_STABLE#"$PKG"/v}"
  START_REF="$LAST_STABLE"
fi

# Коммиты от стабильного тега до текущего HEAD, затрагивающие пакет
COMMITS=$(git log "${START_REF}..HEAD" --format="%s" -- "${PKG}/" 2>/dev/null || true)

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

# Ручной расчёт следующей версии
IFS='.' read -r MAJOR MINOR PATCH <<<"$CURRENT_VERSION"
case $BUMP_TYPE in
  major)
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
    ;;
  minor)
    MINOR=$((MINOR + 1))
    PATCH=0
    ;;
  patch)
    PATCH=$((PATCH + 1))
    ;;
esac

NEXT_VERSION="${MAJOR}.${MINOR}.${PATCH}"
echo "$NEXT_VERSION"
