#!/bin/bash
# scripts/next-version.sh
# Usage: ./next-version.sh <package> [base-ref] <head-ref>
# If base-ref is empty, uses the latest stable tag for the package.

set -e

PKG="$1"
BASE_REF="$2"
HEAD_REF="$3"

# Find the latest stable tag for this package
LAST_STABLE=$(git tag -l "${PKG}/v*" | grep -E "^${PKG}/v[0-9]+\.[0-9]+\.[0-9]+$" | sort -V | tail -n1)

if [ -z "$LAST_STABLE" ]; then
  CURRENT_VERSION="0.0.0"
else
  CURRENT_VERSION="${LAST_STABLE#${PKG}/v}"
fi

# Determine base reference for commit log
if [ -z "$BASE_REF" ]; then
  if [ -n "$LAST_STABLE" ]; then
    BASE_REF="$LAST_STABLE"
  else
    BASE_REF=$(git rev-list --max-parents=0 HEAD)
  fi
fi

# Get commits affecting the package since BASE_REF
COMMITS=$(git log "${BASE_REF}..${HEAD_REF}" --format="%s" -- "${PKG}/" 2>/dev/null || true)

# Default bump type: patch if there are any commits, else none
if [ -z "$COMMITS" ]; then
  echo "$CURRENT_VERSION"
  exit 0
fi

BUMP_TYPE="patch"

if echo "$COMMITS" | grep -qiE "major\(${PKG}\)|BREAKING CHANGE"; then
  BUMP_TYPE="major"
elif echo "$COMMITS" | grep -qiE "(feat|minor)\(${PKG}\)"; then
  BUMP_TYPE="minor"
elif echo "$COMMITS" | grep -qi "fix(${PKG})"; then
  BUMP_TYPE="patch"
fi

# Compute next version
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
