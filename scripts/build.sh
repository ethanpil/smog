#!/bin/bash

# Exit on error
set -e

# Get the target OS from the first argument
TARGET_OS=$1
if [ -z "$TARGET_OS" ]; then
    echo "Usage: $0 <linux|darwin|windows>"
    exit 1
fi

# Get the project's root directory
ROOT_DIR=$(git rev-parse --show-toplevel)
cd "$ROOT_DIR"

# Set the package to build
PACKAGE="github.com/ethanpil/smog/cmd/smog"

# Create the output directory
OUTPUT_DIR="$ROOT_DIR/dist"
mkdir -p "$OUTPUT_DIR"

# Function to build for a specific os/arch
build() {
    local os=$1
    local arch=$2
    local output_name="smog-${os}-${arch}"
    if [ "$os" = "windows" ]; then
        output_name+=".exe"
    fi

    echo "Building for $os/$arch..."
    GOOS=$os GOARCH=$arch go build -o "$OUTPUT_DIR/$output_name" -v $PACKAGE
    echo "Successfully built $OUTPUT_DIR/$output_name"
}

# Build for the specified OS
case "$TARGET_OS" in
    linux)
        build "linux" "amd64"
        build "linux" "arm64"
        ;;
    darwin)
        build "darwin" "amd64"
        build "darwin" "arm64"
        ;;
    windows)
        build "windows" "amd64"
        build "windows" "arm64"
        ;;
    *)
        echo "Unsupported OS: $TARGET_OS"
        exit 1
        ;;
esac

echo "All builds for $TARGET_OS completed successfully."

# Archive the binaries
cd "$OUTPUT_DIR"
ARCHIVE_NAME="smog-$TARGET_OS"

if [ "$TARGET_OS" = "windows" ]; then
    ARCHIVE_NAME+=".zip"
    zip "$ARCHIVE_NAME" smog-windows-*.exe
else
    ARCHIVE_NAME+=".tar.gz"
    tar -czf "$ARCHIVE_NAME" smog-$TARGET_OS-*
fi

cd "$ROOT_DIR"

echo "Successfully created archive: $OUTPUT_DIR/$ARCHIVE_NAME"
ls -l "$OUTPUT_DIR"
