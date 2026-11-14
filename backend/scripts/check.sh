#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mode=${1:-full}
[[ $# -le 1 && ( "$mode" == full || "$mode" == --unit ) ]] || { echo 'Usage: scripts/check.sh [--unit]' >&2; exit 1; }
if [[ "$mode" == full ]]; then
  [[ -n "${TEST_DATABASE_URL:-}" ]] || { echo 'TEST_DATABASE_URL is required for the full check.' >&2; exit 1; }
  command -v docker >/dev/null || { echo 'Docker CLI with Compose is required.' >&2; exit 1; }
  docker compose version >/dev/null
else
  unset TEST_DATABASE_URL
fi
output_dir=${CHECK_OUTPUT_DIR:-/tmp/ai-buddy-check}
mkdir -p "$output_dir"
exec > >(tee "$output_dir/check.log") 2>&1
scripts/lint.sh
go vet ./...
go build -o "$output_dir/bin/" ./cmd/...
if [[ "$mode" == --unit ]]; then
  go test -race ./...
  exit
fi
packages=$(go list ./internal/tracker/... ./internal/ticket/... ./internal/wiki/... ./internal/buddy/... ./internal/platform/... ./cmd/... | paste -sd,)
# Run every race test once, instrumenting the union of all five groups.
result=0
go test -race -coverpkg="$packages" -coverprofile="$output_dir/all.cover" ./... 2>&1 | tee "$output_dir/tests.log" || result=1
for group in tracker ticket wiki buddy platform; do
  awk -v group="$group" -f scripts/coverage.awk "$output_dir/all.cover" > "$output_dir/$group.cover" || result=1
  go tool cover -func="$output_dir/$group.cover" > "$output_dir/$group-coverage.txt"
done
for service in tracker ticket wiki buddy ollama; do
  docker compose -f "deploy/$service/compose.yaml" --env-file "deploy/$service/.env.example" config --quiet
done
exit "$result"
