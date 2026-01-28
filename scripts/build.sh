#!/bin/sh

set -e

# Change to repo root
cd "$(dirname "$0")/../"

OS=$(uname -s)

# Determine output binary name
case "$OS" in
  *CYGWIN*|*MINGW*|*MSYS*)
    OUTPUT="./build/divinesense.exe"
    ;;
  *)
    OUTPUT="./build/divinesense"
    ;;
esac

echo "Building for $OS..."

# Ensure build directories exist and configure a writable Go build cache
mkdir -p ./build/.gocache ./build/.gomodcache
export GOCACHE="$(pwd)/build/.gocache"
export GOMODCACHE="$(pwd)/build/.gomodcache"

# Build the executable
go build -o "$OUTPUT" ./cmd/divinesense

echo "Build successful!"
echo "To run the application, execute the following command:"
echo "$OUTPUT --mode dev"
