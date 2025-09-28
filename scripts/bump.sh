#!/bin/bash

# bump.sh - Version bumping script for vfox
# Usage: ./scripts/bump.sh <new_version>
# Example: ./scripts/bump.sh 0.7.1

set -e

# Check if version argument is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 <new_version>"
    echo "Example: $0 0.7.1"
    exit 1
fi

NEW_VERSION="$1"
VERSION_FILE="internal/version.go"

# Validate version format (basic semver check)
if ! echo "$NEW_VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$'; then
    echo "Error: Invalid version format. Please use semantic versioning (e.g., 1.2.3)"
    exit 1
fi

# Check if version file exists
if [ ! -f "$VERSION_FILE" ]; then
    echo "Error: Version file $VERSION_FILE not found"
    exit 1
fi

# Get current version
CURRENT_VERSION=$(grep 'const RuntimeVersion' "$VERSION_FILE" | sed 's/.*"\(.*\)".*/\1/')
echo "Current version: $CURRENT_VERSION"
echo "New version: $NEW_VERSION"

# Confirm the change
read -p "Do you want to proceed with version bump? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Version bump cancelled"
    exit 0
fi

# Update version in the file
echo "Updating version in $VERSION_FILE..."
sed -i.bak "s/const RuntimeVersion = \".*\"/const RuntimeVersion = \"$NEW_VERSION\"/" "$VERSION_FILE"

# Remove backup file
rm "${VERSION_FILE}.bak"

# Verify the change
NEW_VERSION_CHECK=$(grep 'const RuntimeVersion' "$VERSION_FILE" | sed 's/.*"\(.*\)".*/\1/')
if [ "$NEW_VERSION_CHECK" != "$NEW_VERSION" ]; then
    echo "Error: Version update failed. Expected $NEW_VERSION, got $NEW_VERSION_CHECK"
    exit 1
fi

echo "Version updated successfully to $NEW_VERSION"

# Stage the version file
echo "Staging version file..."
git add "$VERSION_FILE"

# Commit the change
echo "Creating commit..."
git commit -m "bump version to $NEW_VERSION"

# Create and push tag
echo "Creating git tag v$NEW_VERSION..."
git tag "v$NEW_VERSION"

echo "Version bump completed successfully!"
echo "Don't forget to push the changes and tag:"
echo "  git push origin main"
echo "  git push origin v$NEW_VERSION"