#!/bin/bash

# Quasar Go Agent - Installer Script
# Usage: curl -sL https://get.gravito.dev/quasar-go | bash

set -e

REPO="gravito-framework/quasar-go"
BINARY_NAME="quasar-go"
GITHUB_API="https://api.github.com/repos/$REPO/releases/latest"

# 1. Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
    darwin)  OS="darwin" ;;
    linux)   OS="linux" ;;
    *) echo "❌ Unsupported OS: $OS"; exit 1 ;;
esac

# 2. Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

# 3. Get latest version from GitHub
echo "🔍 Finding latest version of Quasar Go Agent..."
VERSION=$(curl -s $GITHUB_API | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo "❌ Could not find latest version. Please check GitHub: https://github.com/$REPO/releases"
    exit 1
fi

# 4. Construct Download URL
# Pattern: quasar-go-darwin-arm64
FILENAME="${BINARY_NAME}-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

echo "🚀 Downloading $BINARY_NAME $VERSION for $OS/$ARCH..."
curl -L -o "$BINARY_NAME" "$DOWNLOAD_URL"
chmod +x "$BINARY_NAME"

# 5. Move to local bin if possible
if [ -w "/usr/local/bin" ]; then
    mv "$BINARY_NAME" "/usr/local/bin/$BINARY_NAME"
    echo "✅ Successfully installed to /usr/local/bin/$BINARY_NAME"
else
    echo "⚠️  /usr/local/bin is not writable. $BINARY_NAME has been downloaded to current directory."
    echo "💡 Run 'sudo mv $BINARY_NAME /usr/local/bin/' to finish installation."
fi

echo "✨ Done! You can now run: $BINARY_NAME --help"
