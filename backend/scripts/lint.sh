#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
command -v golangci-lint >/dev/null || { echo 'Install golangci-lint 2.13.2 (see README.md).' >&2; exit 1; }
[[ $(golangci-lint version) == *'version 2.13.2 '* ]] || { echo 'golangci-lint 2.13.2 is required.' >&2; exit 1; }
golangci-lint fmt --diff
golangci-lint run ./...
