#!/usr/bin/env bash
set -e

# Fetch latest tags
git fetch --tags origin >/dev/null 2>&1 || true
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v1.4.0")

echo "========================================="
echo "   Automatic Mouse Mover Release Tool   "
echo "========================================="
echo "Current latest tag: ${LATEST_TAG}"
echo ""

# Prompt for version tag
read -p "Enter new version tag (e.g. v1.5.1): " NEW_VERSION
if [ -z "$NEW_VERSION" ]; then
    echo "Error: Version tag cannot be empty."
    exit 1
fi

# Ensure leading 'v'
if [[ ! "$NEW_VERSION" =~ ^v ]]; then
    NEW_VERSION="v$NEW_VERSION"
fi
CLEAN_VERSION="${NEW_VERSION#v}"

# Prompt for optional release title
read -p "Enter release title [Default: Release ${NEW_VERSION}]: " RELEASE_TITLE
if [ -z "$RELEASE_TITLE" ]; then
    RELEASE_TITLE="Release ${NEW_VERSION}"
fi

# Prompt for optional release notes
read -p "Enter release description notes (optional): " RELEASE_NOTES

echo ""
echo "Summary:"
echo "- Version Tag:  ${NEW_VERSION}"
echo "- Version Num:  ${CLEAN_VERSION}"
echo "- Title:        ${RELEASE_TITLE}"
echo "- Description:  ${RELEASE_NOTES:-'(Auto-generated from commits)'}"
echo ""
read -p "Do you want to proceed and publish this release? (y/N): " CONFIRM
if [[ ! "$CONFIRM" =~ ^[yY]$ ]]; then
    echo "Aborted."
    exit 0
fi

# 1. Update Info.plist version
PLIST_FILE="appInfo/Info.plist"
if [ -f "$PLIST_FILE" ]; then
    /usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString ${CLEAN_VERSION}" "$PLIST_FILE" 2>/dev/null || true
fi

# 2. Run unit tests
echo "Running test suite..."
go test ./...

# 3. Commit version bump if dirty
git add appInfo/Info.plist
if ! git diff-index --quiet HEAD --; then
    git commit -m "chore(release): bump version to ${NEW_VERSION}"
    git push origin master
fi

# 4. Create annotated tag and push
echo "Creating tag ${NEW_VERSION}..."
TAG_MSG="${RELEASE_TITLE}"
if [ -n "$RELEASE_NOTES" ]; then
    TAG_MSG="${RELEASE_TITLE} - ${RELEASE_NOTES}"
fi
git tag -a "${NEW_VERSION}" -m "${TAG_MSG}"
git push origin "${NEW_VERSION}"

echo ""
echo "🎉 Tag ${NEW_VERSION} has been pushed to GitHub!"
echo "GitHub Actions is now compiling the Universal binary and publishing the release."
echo "Track status: https://github.com/samlara32/automatic-mouse-mover/actions"
