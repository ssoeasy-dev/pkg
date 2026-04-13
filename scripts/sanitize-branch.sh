#!/bin/bash
# scripts/sanitize-branch.sh
# Usage: ./sanitize-branch.sh <branch-name>

BRANCH="$1"
# Replace '/' and '_' with '-', convert to lowercase, keep only alphanumeric and '-'
echo "$BRANCH" | sed 's/[\/_]/-/g' | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]//g'
