#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
output_dir=${CHECK_OUTPUT_DIR:-/tmp/ai-buddy-check}
mkdir -p "$output_dir"
mapfile -t sources < <(find cmd internal -name '*.go' -type f | sort)
unformatted=$(gofmt -l "${sources[@]}")
if [[ -n "$unformatted" ]]; then
  echo "Run gofmt on:" >&2
  echo "$unformatted" >&2
  exit 1
fi
go vet ./...
go build -o "$output_dir/bin/" ./cmd/...
for group in tracker ticket wiki platform; do
  if [[ "$group" == platform ]]; then
    packages=$(go list ./internal/platform/... ./cmd/migrate ./cmd/seed | paste -sd,)
  elif [[ "$group" == wiki ]]; then
    packages=$(go list ./internal/wiki/... ./cmd/wiki ./cmd/wiki-index | paste -sd,)
  else
    packages=$(go list "./internal/$group/..." "./cmd/$group" | paste -sd,)
  fi
  # Instrument only this group. All tests may exercise it, but no other group's
  # statements or test helpers can contribute to its numerator or denominator.
  go test -race -coverpkg="$packages" -coverprofile="$output_dir/$group.cover" ./... > "$output_dir/$group-tests.log" 2>&1 || {
    cat "$output_dir/$group-tests.log" >&2
    exit 1
  }
  go tool cover -func="$output_dir/$group.cover" > "$output_dir/$group-coverage.txt"
  # Merge identical blocks across test binaries and compare statement counts.
  # A rounded 80.0% must never pass the gate.
  awk -v group="$group" 'NR>1 {statements[$1]=$2; if ($3>0) hit[$1]=1} END {
    for (block in statements) {total+=statements[block]; if (hit[block]) covered+=statements[block]}
    if(total==0) {print group ": no coverage blocks"; exit 1}
    printf "%s: %.2f%% (%d/%d statements)\n",group,100*covered/total,covered,total
    if(100*covered<=80*total) exit 1
  }' "$output_dir/$group.cover"
done
if [[ -z "${TEST_DATABASE_URL:-}" ]]; then
  echo 'PostgreSQL integration tests skipped: TEST_DATABASE_URL is unset.'
fi
if command -v docker >/dev/null; then
  for service in tracker ticket wiki ollama; do
    docker compose -f "deploy/$service/compose.yaml" --env-file "deploy/$service/.env.example" config --quiet
  done
else
  echo 'Compose validation skipped: Docker CLI unavailable.'
fi
