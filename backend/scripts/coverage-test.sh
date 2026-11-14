#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
fixture=$(mktemp)
trap 'rm -f "$fixture"' EXIT
cat > "$fixture" <<'PROFILE'
mode: atomic
ai-buddy/internal/tracker/tasks/a.go:1.1,2.1 4 1
ai-buddy/internal/tracker/tasks/a.go:3.1,4.1 1 0
ai-buddy/internal/ticket/tickets/a.go:1.1,2.1 100 1
ai-buddy/internal/testutil/a.go:1.1,2.1 100 1
ai-buddy/internal/platform/httpx/a.go:1.1,2.1 100 1
PROFILE
if awk -v group=tracker -f scripts/coverage.awk "$fixture" >/dev/null 2>&1; then
  echo 'Exactly 80% incorrectly passed' >&2; exit 1
fi
# Other services, platform and helpers must never rescue Tracker's exact 80%.
echo 'ai-buddy/internal/tracker/tasks/a.go:3.1,4.1 1 1' >> "$fixture"
output=$(awk -v group=tracker -f scripts/coverage.awk "$fixture" 2>/dev/null)
[[ "$output" != *'/ticket/'* && "$output" != *'/platform/'* && "$output" != *'/testutil/'* ]]
[[ $(wc -l <<< "$output") -eq 3 ]]
for group in buddy wiki; do
  if awk -v group="$group" -f scripts/coverage.awk "$fixture" >/dev/null 2>&1; then
    echo 'Missing group incorrectly passed' >&2; exit 1
  fi
done
# Check every membership rule, including command packages and prefix collisions.
cat > "$fixture" <<'PROFILE'
mode: atomic
ai-buddy/internal/tracker/tasks/a.go:1.1,2.1 1 1
ai-buddy/cmd/tracker/main.go:1.1,2.1 1 1
ai-buddy/internal/ticket/tickets/a.go:1.1,2.1 1 1
ai-buddy/cmd/ticket/main.go:1.1,2.1 1 1
ai-buddy/internal/wiki/http/a.go:1.1,2.1 1 1
ai-buddy/cmd/wiki/main.go:1.1,2.1 1 1
ai-buddy/cmd/wiki-index/main.go:1.1,2.1 1 1
ai-buddy/internal/buddy/http/a.go:1.1,2.1 1 1
ai-buddy/cmd/buddy/main.go:1.1,2.1 1 1
ai-buddy/internal/platform/httpx/a.go:1.1,2.1 1 1
ai-buddy/cmd/migrate/main.go:1.1,2.1 1 1
ai-buddy/cmd/seed/main.go:1.1,2.1 1 1
ai-buddy/internal/testutil/a.go:1.1,2.1 100 0
ai-buddy/internal/buddy-other/a.go:1.1,2.1 100 0
PROFILE
for group in tracker ticket wiki buddy platform; do
  output=$(awk -v group="$group" -f scripts/coverage.awk "$fixture" 2>/dev/null)
  expected=3
  [[ "$group" != wiki && "$group" != platform ]] || expected=4
  [[ $(wc -l <<< "$output") -eq "$expected" ]]
done
# 80.00001% must pass despite rendering as 80.00%.
cat > "$fixture" <<'PROFILE'
mode: atomic
ai-buddy/internal/tracker/a.go:1.1,2.1 8000001 1
ai-buddy/internal/tracker/a.go:3.1,4.1 1999999 0
PROFILE
awk -v group=tracker -f scripts/coverage.awk "$fixture" >/dev/null 2>&1
