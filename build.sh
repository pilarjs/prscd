#!/usr/bin/env bash

set -e

# Environment variable options:
#   - PLATFORMS: Platforms to build for (e.g. "windows/amd64,linux/amd64,darwin/amd64")

export CLI_VERSION=$(git describe --tags 2>/dev/null || git rev-parse --short HEAD)

export LC_ALL=C
export LC_DATE=C

# make_ldflags() {
#     local ldflags="-s -w" #-X 'github.com/pilarjs/prscd/cli.Version=$CLI_VERSION'"
# }

build_for_platform() {
    local platform="$1"
    local ldflags="$2"

    local GOOS="${platform%/*}"
    local GOARCH="${platform#*/}"
    if [[ -z "$GOOS" || -z "$GOARCH" ]]; then
        echo "Invalid platform $platform" >&2
        return 1
    fi
    echo "Building $GOOS/$GOARCH"
    local output="prscd"
    if [[ "$GOOS" = "windows" ]]; then
        output="$output.exe"
    fi
    # compress to .tar.gz file
    local binfile="build/prscd-$GOARCH-$GOOS.tar.gz"
    local exit_val=0
    GOOS=$GOOS GOARCH=$GOARCH go build -o "build/$output" -ldflags "$ldflags" -gcflags=-l -trimpath ./cmd/prscd || exit_val=$?
    if [[ "$exit_val" -ne 0 ]]; then
        echo "Error: failed to build $GOOS/$GOARCH" >&2
        return $exit_val
    fi
    # compress compiled binary
    tar -C build -czvf "$binfile" "$output"
    rm -rf "build/$output"
}

if [ -z "$PLATFORMS" ]; then
    PLATFORMS="$(go env GOOS)/$(go env GOARCH)"
fi

platforms=(${PLATFORMS//,/ })
ldflags="-s -w" #-X 'github.com/pilarjs/prscd/cli.Version=$CLI_VERSION'"

mkdir -p build
rm -rf build/*

echo "Starting build..."

for platform in "${platforms[@]}"; do
    build_for_platform "$platform" "$ldflags"
done

echo "Build complete."

ls -lh build/ | awk '{print $9, $5}'
