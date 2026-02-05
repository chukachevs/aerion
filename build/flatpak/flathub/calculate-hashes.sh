#!/bin/bash
# Calculate SHA256 hashes and sizes for Flathub extra-data manifest
# Usage: ./calculate-hashes.sh v0.1.13

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.1.13"
    exit 1
fi

VERSION="$1"
REPO="https://github.com/hkdb/aerion"

echo "=========================================="
echo "Flathub Manifest Hash Calculator"
echo "=========================================="
echo "Version: $VERSION"
echo "Repository: $REPO"
echo ""
echo "Downloading and calculating..."
echo ""

# Create temp directory
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

# Download and calculate for x86_64
echo "📦 x86_64 tarball..."
if wget -q "${REPO}/releases/download/${VERSION}/aerion-${VERSION}-linux-x86_64.tar.gz"; then
    X86_64_SHA256=$(sha256sum aerion-${VERSION}-linux-x86_64.tar.gz | awk '{print $1}')
    X86_64_SIZE=$(stat -c%s aerion-${VERSION}-linux-x86_64.tar.gz)
    echo "   URL: ${REPO}/releases/download/${VERSION}/aerion-${VERSION}-linux-x86_64.tar.gz"
    echo "   SHA256: $X86_64_SHA256"
    echo "   Size: $X86_64_SIZE bytes"
    echo ""
else
    echo "   ❌ ERROR: Could not download x86_64 tarball"
    echo ""
    X86_64_SHA256="ERROR_FILE_NOT_FOUND"
    X86_64_SIZE="0"
fi

# Download and calculate for aarch64
echo "📦 aarch64 tarball..."
if wget -q "${REPO}/releases/download/${VERSION}/aerion-${VERSION}-linux-aarch64.tar.gz"; then
    AARCH64_SHA256=$(sha256sum aerion-${VERSION}-linux-aarch64.tar.gz | awk '{print $1}')
    AARCH64_SIZE=$(stat -c%s aerion-${VERSION}-linux-aarch64.tar.gz)
    echo "   URL: ${REPO}/releases/download/${VERSION}/aerion-${VERSION}-linux-aarch64.tar.gz"
    echo "   SHA256: $AARCH64_SHA256"
    echo "   Size: $AARCH64_SIZE bytes"
    echo ""
else
    echo "   ❌ ERROR: Could not download aarch64 tarball"
    echo ""
    AARCH64_SHA256="ERROR_FILE_NOT_FOUND"
    AARCH64_SIZE="0"
fi

# Note: Desktop file, icon, and metainfo are included directly in the Flathub repo
# No need to calculate hashes for them

# Cleanup
cd - > /dev/null
rm -rf "$TEMP_DIR"

echo "=========================================="
echo "Updating manifest file..."
echo "=========================================="

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
MANIFEST="${SCRIPT_DIR}/com.github.hkdb.Aerion-extradata.yml"

if [ ! -f "$MANIFEST" ]; then
    echo "❌ ERROR: Manifest file not found: $MANIFEST"
    exit 1
fi

# Create backup
cp "$MANIFEST" "${MANIFEST}.backup"

# Update x86_64 URL
sed -i "s|url: https://github.com/hkdb/aerion/releases/download/v[0-9.]\+/aerion-v[0-9.]\+-linux-x86_64.tar.gz|url: ${REPO}/releases/download/${VERSION}/aerion-${VERSION}-linux-x86_64.tar.gz|" "$MANIFEST"

# Update x86_64 sha256 (find the sha256 line after x86_64 URL)
awk -v sha="$X86_64_SHA256" '
/url:.*x86_64\.tar\.gz/ { found_x86=1; print; next }
found_x86 && /sha256:/ {
    match($0, /^[ \t]*/);
    spaces=substr($0, 1, RLENGTH);
    print spaces "sha256: " sha;
    found_x86=0;
    next
}
{ print }
' "$MANIFEST" > "${MANIFEST}.tmp" && mv "${MANIFEST}.tmp" "$MANIFEST"

# Update x86_64 size
awk -v size="$X86_64_SIZE" '
/url:.*x86_64\.tar\.gz/ { found_x86=1; print; next }
found_x86 && /size:/ {
    match($0, /^[ \t]*/);
    spaces=substr($0, 1, RLENGTH);
    print spaces "size: " size;
    found_x86=0;
    next
}
{ print }
' "$MANIFEST" > "${MANIFEST}.tmp" && mv "${MANIFEST}.tmp" "$MANIFEST"

# Update aarch64 URL
sed -i "s|url: https://github.com/hkdb/aerion/releases/download/v[0-9.]\+/aerion-v[0-9.]\+-linux-aarch64.tar.gz|url: ${REPO}/releases/download/${VERSION}/aerion-${VERSION}-linux-aarch64.tar.gz|" "$MANIFEST"

# Update aarch64 sha256
awk -v sha="$AARCH64_SHA256" '
/url:.*aarch64\.tar\.gz/ { found_arm=1; print; next }
found_arm && /sha256:/ {
    match($0, /^[ \t]*/);
    spaces=substr($0, 1, RLENGTH);
    print spaces "sha256: " sha;
    found_arm=0;
    next
}
{ print }
' "$MANIFEST" > "${MANIFEST}.tmp" && mv "${MANIFEST}.tmp" "$MANIFEST"

# Update aarch64 size
awk -v size="$AARCH64_SIZE" '
/url:.*aarch64\.tar\.gz/ { found_arm=1; print; next }
found_arm && /size:/ {
    match($0, /^[ \t]*/);
    spaces=substr($0, 1, RLENGTH);
    print spaces "size: " size;
    found_arm=0;
    next
}
{ print }
' "$MANIFEST" > "${MANIFEST}.tmp" && mv "${MANIFEST}.tmp" "$MANIFEST"

echo ""
echo "✅ Manifest updated successfully!"
echo "   Backup saved: ${MANIFEST}.backup"
echo ""
echo "=========================================="
echo "Summary:"
echo "=========================================="
echo ""
echo "x86_64 tarball:"
echo "  url: ${REPO}/releases/download/${VERSION}/aerion-${VERSION}-linux-x86_64.tar.gz"
echo "  sha256: $X86_64_SHA256"
echo "  size: $X86_64_SIZE"
echo ""
echo "aarch64 tarball:"
echo "  url: ${REPO}/releases/download/${VERSION}/aerion-${VERSION}-linux-aarch64.tar.gz"
echo "  sha256: $AARCH64_SHA256"
echo "  size: $AARCH64_SIZE"
echo ""
echo "=========================================="
echo "Next steps:"
echo "1. ✅ Manifest updated with ${VERSION} hashes"
echo "2. Review changes: git diff com.github.hkdb.Aerion-extradata.yml"
echo "3. Copy updated desktop/icon/metainfo to Flathub repo if changed"
echo "4. Test build: flatpak-builder --force-clean build-dir com.github.hkdb.Aerion-extradata.yml"
echo "=========================================="
