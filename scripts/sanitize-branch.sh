#!/bin/bash
# scripts/sanitize-branch.sh
# Usage: ./sanitize-branch.sh <branch-name>

BRANCH="$1"
# Replace all non-alphanumeric characters with '-', then collapse multiple '-'
echo "$BRANCH" | sed 's/[^a-zA-Z0-9]/-/g' | sed 's/--*/-/g' | sed 's/^-//;s/-$//'
