#!/bin/bash
set -e

# Extract module and version information from a git tag
# Usage: ./scripts/release-info.sh <tag>

TAG="${1:-${GITHUB_REF#refs/tags/}}"

if [ -z "$TAG" ]; then
    echo "Error: TAG is required"
    echo "Usage: $0 <tag>"
    exit 1
fi

# Parse tag format
if [[ "$TAG" =~ ^v([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
    # Main module tag (e.g., v1.0.0)
    MODULE="main"
    MODULE_PATH="."
    VERSION="${BASH_REMATCH[1]}"
    RELEASE_NAME="Sift v$VERSION"
elif [[ "$TAG" =~ ^(thru/[^/]+)/v([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
    # Submodule tag (e.g., thru/sql/v1.0.0)
    MODULE_PATH="${BASH_REMATCH[1]}"
    VERSION="${BASH_REMATCH[2]}"
    MODULE_NAME=$(echo "$MODULE_PATH" | sed 's/thru\///')
    MODULE="$MODULE_NAME"
    # Capitalize first letter (portable way)
    MODULE_NAME_CAPITALIZED=$(echo "$MODULE_NAME" | awk '{print toupper(substr($0,1,1)) tolower(substr($0,2))}')
    RELEASE_NAME="Sift $MODULE_NAME_CAPITALIZED v$VERSION"
else
    echo "Error: Invalid tag format: $TAG"
    echo "Expected: v1.0.0 or thru/module/v1.0.0"
    exit 1
fi

# Output in GitHub Actions format if GITHUB_OUTPUT is set
if [ -n "$GITHUB_OUTPUT" ]; then
    echo "tag=$TAG" >> "$GITHUB_OUTPUT"
    echo "module=$MODULE" >> "$GITHUB_OUTPUT"
    echo "module_path=$MODULE_PATH" >> "$GITHUB_OUTPUT"
    echo "version=$VERSION" >> "$GITHUB_OUTPUT"
    echo "release_name=$RELEASE_NAME" >> "$GITHUB_OUTPUT"
else
    # Output for local testing
    echo "TAG=$TAG"
    echo "MODULE=$MODULE"
    echo "MODULE_PATH=$MODULE_PATH"
    echo "VERSION=$VERSION"
    echo "RELEASE_NAME=$RELEASE_NAME"
fi
