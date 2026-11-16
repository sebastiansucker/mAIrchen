#!/usr/bin/env bash
# Installs the dev toolchain without root (Go, golangci-lint into ~/.local) and fetches
# project dependencies. Idempotent: safe to run on every session start.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PREFIX="$HOME/.local"
BIN="$PREFIX/bin"
mkdir -p "$BIN"
export PATH="$BIN:$PATH"

# Go version follows backend/go.mod so Renovate bumps carry over automatically.
GO_VERSION="$(awk '/^go /{print $2}' "$ROOT/backend/go.mod")"
if [ "$(go env GOVERSION 2>/dev/null)" != "go$GO_VERSION" ]; then
  case "$(uname -m)" in
    x86_64) ARCH=amd64 ;;
    aarch64 | arm64) ARCH=arm64 ;;
    *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;;
  esac
  echo "Installing Go $GO_VERSION ..."
  rm -rf "$PREFIX/go"
  curl -fsSL "https://go.dev/dl/go$GO_VERSION.linux-$ARCH.tar.gz" | tar -xz -C "$PREFIX"
  ln -sf "$PREFIX/go/bin/go" "$PREFIX/go/bin/gofmt" "$BIN/"
fi

if ! command -v golangci-lint >/dev/null; then
  echo "Installing golangci-lint ..."
  curl -fsSL https://golangci-lint.run/install.sh | sh -s -- -b "$BIN"
fi

(cd "$ROOT/backend" && go mod download)
(cd "$ROOT/tests" && npm install --no-audit --no-fund --silent)

echo "dev-setup done: $(go version), $(golangci-lint --version | head -1)"
