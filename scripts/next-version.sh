#!/bin/bash
# scripts/next-version.sh
# Usage: ./next-version.sh <package> <base-ref> <head-ref>

set -e

PKG="$1"
BASE_REF="$2"
HEAD_REF="$3"

# Find latest stable tag for this package
LAST_STABLE_TAG=$(git tag -l "${PKG}/v*" | grep -E "^${PKG}/v[0-9]+\.[0-9]+\.[0-9]+$" | sort -V | tail -n1)

if [ -z "$LAST_STABLE_TAG" ]; then
  CURRENT_VERSION="0.0.0"
else
  CURRENT_VERSION="${LAST_STABLE_TAG#${PKG}/v}"
fi

# Get commits between base and head that affect the package
COMMITS=$(git log "${BASE_REF}..${HEAD_REF}" --format="%s" -- "${PKG}/" 2>/dev/null || true)

BUMP_TYPE="none"
if echo "$COMMITS" | grep -qiE "major\(${PKG}\)|BREAKING CHANGE"; then
  BUMP_TYPE="major"
elif echo "$COMMITS" | grep -qiE "(feat|minor)\(${PKG}\)"; then
  BUMP_TYPE="minor"
elif echo "$COMMITS" | grep -qiE "fix\(${PKG}\)"; then
  BUMP_TYPE="patch"
fi

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
