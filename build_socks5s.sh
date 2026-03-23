#!/bin/bash

# Build script for socks5s

set -e

# Get the directory where this script is located
PROJECT_ROOT="."

# Read version from .version file
VERSION_FILE=".version"
if [ ! -f "$VERSION_FILE" ]; then
    echo "Error: .version file not found at $VERSION_FILE"
    exit 1
fi

VERSION=$(cat "$VERSION_FILE" | tr -d '[:space:]')
if [ -z "$VERSION" ]; then
    echo "Error: Version is empty in $VERSION_FILE"
    exit 1
fi

echo "Building socks5s version: $VERSION"

# Get current date in ISO 8601 format
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Get commit hash (if in git repository)
if git -C "$PROJECT_ROOT" rev-parse --git-dir > /dev/null 2>&1; then
    COMMIT_HASH=$(git -C "$PROJECT_ROOT" rev-parse HEAD)
    if [ -z "$COMMIT_HASH" ]; then
        COMMIT_HASH="unknown"
    fi
else
    COMMIT_HASH="unknown"
fi

# Get Go version
GO_VERSION=$(go version | awk '{print $3}')

# Get platform
PLATFORM="$(go env GOOS)/$(go env GOARCH)"

# Build output directory
OUTPUT_DIR="build"
BINARY_NAME="socks5s"

# Build with linker flags
echo "Building with:"
echo "  Version:     $VERSION"
echo "  Build Date:  $BUILD_DATE"
echo "  Commit Hash: $COMMIT_HASH"
echo "  Go Version:  $GO_VERSION"
echo "  Platform:    $PLATFORM"

cd "$PROJECT_ROOT"

go build -v \
    -ldflags "\
        -X 'github.com/HT4w5/socks5/cmd/socks5s/meta.BuildDate=$BUILD_DATE' \
        -X 'github.com/HT4w5/socks5/cmd/socks5s/meta.CommitHash=$COMMIT_HASH' \
        -X 'github.com/HT4w5/socks5/cmd/socks5s/meta.Version=$VERSION' \
        -X 'github.com/HT4w5/socks5/cmd/socks5s/meta.Platform=$PLATFORM' \
        -X 'github.com/HT4w5/socks5/cmd/socks5s/meta.GoVersion=$GO_VERSION'" \
    -o "$OUTPUT_DIR/$BINARY_NAME" \
    cmd/socks5s/main.go

if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo "Binary created at: $OUTPUT_DIR/$BINARY_NAME"
    
    # Display version info from built binary
    echo ""
    echo "Version info from built binary:"
    "$OUTPUT_DIR/$BINARY_NAME" --help 2>&1 | head -1
else
    echo "Build failed!"
    exit 1
fi
