#!/bin/bash
set -e

# Generate changelog between two tags for a specific module
# Usage: ./scripts/generate-changelog.sh <current-tag> <module-path>

TAG="$1"
MODULE_PATH="${2:-.}"

if [ -z "$TAG" ]; then
    echo "Error: TAG is required"
    echo "Usage: $0 <tag> [module-path]"
    exit 1
fi

# Find previous tag for this module
if [ "$MODULE_PATH" = "." ]; then
    PREV_TAG=$(git tag -l 'v*' --sort=-v:refname | grep -v "^$TAG$" | head -n1)
else
    PREV_TAG=$(git tag -l "${MODULE_PATH}/v*" --sort=-v:refname | grep -v "^$TAG$" | head -n1)
fi

if [ -z "$PREV_TAG" ]; then
    echo "Initial release"
    exit 0
fi

# Generate changelog from commits
if [ "$MODULE_PATH" = "." ]; then
    # Main module: exclude thru/ directory
    CHANGELOG=$(git log --pretty=format:"- %s (%h)" "$PREV_TAG..$TAG" -- . ':!thru/' 2>/dev/null || echo "")
else
    # Submodule: only include that directory
    CHANGELOG=$(git log --pretty=format:"- %s (%h)" "$PREV_TAG..$TAG" -- "$MODULE_PATH/" 2>/dev/null || echo "")
fi

if [ -z "$CHANGELOG" ]; then
    echo "No changes in this module"
else
    echo "$CHANGELOG"
fi
