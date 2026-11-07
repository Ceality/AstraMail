#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")/.."

VERSION="$(sed -n 's/^[[:space:]]*Version = "\(.*\)"/\1/p' internal/version/version.go)"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo nogit)"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LDFLAGS="-s -w -X Astramail/internal/version.Commit=${COMMIT} -X Astramail/internal/version.BuildDate=${BUILD_DATE}"

mkdir -p dist

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="${LDFLAGS}" -o "dist/astramail-${VERSION}-windows-amd64.exe" .
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags="${LDFLAGS}" -o "dist/astramail-${VERSION}-linux-amd64" .
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags="${LDFLAGS}" -o "dist/astramail-${VERSION}-darwin-arm64" .

cd dist
sha256sum astramail-"${VERSION}"-* > checksums.txt
cat checksums.txt
