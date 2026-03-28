#!/usr/bin/env bash
# bump-version.sh — Update version across ALL project files atomically.
#
# Usage:
#   ./scripts/bump-version.sh 0.1.0
#
# Updates:
#   1. kubilitics-frontend/package.json
#   2. kubilitics-desktop/src-tauri/tauri.conf.json
#   3. kubilitics-desktop/src-tauri/Cargo.toml
#   4. deploy/helm/kubilitics/Chart.yaml (version + appVersion)
#   5. deploy/helm/kubilitics/values.yaml (image.tag)

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

if [ $# -ne 1 ]; then
  echo "Usage: $0 <new-version>"
  echo "  Example: $0 0.1.0"
  exit 1
fi

NEW_VERSION="$1"

# Validate semver format (with optional pre-release suffix)
if ! echo "$NEW_VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9._-]+)?$'; then
  echo -e "${RED}ERROR:${NC} Invalid version '$NEW_VERSION' — expected semver (e.g., 0.1.0 or 0.1.0-beta.1)"
  exit 1
fi

# Ensure we're at repo root
if [ ! -f "kubilitics-frontend/package.json" ]; then
  echo "ERROR: Run this script from the repository root."
  exit 1
fi

echo "Bumping all version files to $NEW_VERSION"
echo ""

# 1. kubilitics-frontend/package.json
FILE="kubilitics-frontend/package.json"
jq --arg v "$NEW_VERSION" '.version = $v' "$FILE" > "${FILE}.tmp" && mv "${FILE}.tmp" "$FILE"
echo -e "${GREEN}  ✓${NC} $FILE"

# 2. kubilitics-desktop/src-tauri/tauri.conf.json
FILE="kubilitics-desktop/src-tauri/tauri.conf.json"
jq --arg v "$NEW_VERSION" '.version = $v' "$FILE" > "${FILE}.tmp" && mv "${FILE}.tmp" "$FILE"
echo -e "${GREEN}  ✓${NC} $FILE"

# 3. kubilitics-desktop/src-tauri/Cargo.toml
FILE="kubilitics-desktop/src-tauri/Cargo.toml"
sed -i.bak "s/^version = \".*\"/version = \"$NEW_VERSION\"/" "$FILE"
rm -f "${FILE}.bak"
echo -e "${GREEN}  ✓${NC} $FILE"

# 4. deploy/helm/kubilitics/Chart.yaml (version + appVersion)
FILE="deploy/helm/kubilitics/Chart.yaml"
sed -i.bak "s/^version: .*/version: $NEW_VERSION/" "$FILE"
sed -i.bak "s/^appVersion: .*/appVersion: \"$NEW_VERSION\"/" "$FILE"
rm -f "${FILE}.bak"
echo -e "${GREEN}  ✓${NC} $FILE (version + appVersion)"

# 5. deploy/helm/kubilitics/values.yaml (image.tag)
FILE="deploy/helm/kubilitics/values.yaml"
sed -i.bak "s/^  tag: \".*\"/  tag: \"$NEW_VERSION\"/" "$FILE"
rm -f "${FILE}.bak"
echo -e "${GREEN}  ✓${NC} $FILE (image.tag)"

# Update Cargo.lock if it exists
if [ -f "kubilitics-desktop/src-tauri/Cargo.lock" ]; then
  (cd kubilitics-desktop/src-tauri && cargo generate-lockfile 2>/dev/null) || true
  echo -e "${GREEN}  ✓${NC} kubilitics-desktop/src-tauri/Cargo.lock"
fi

echo ""
echo "Verify with: ./scripts/pre-release-check.sh $NEW_VERSION"
echo "Then commit: git add -A && git commit -m 'chore: bump version to $NEW_VERSION'"
