#!/usr/bin/env bash
# check-action-pinning.sh — enforce SHA pinning for all GitHub Actions references.
#
# Every `uses:` entry under .github/workflows/ must reference an action or
# workflow by its full 40-character commit SHA, followed by a comment that
# records the release version that SHA represents:
#
#   uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
#
# Exit codes:
#   0 — all references are SHA-pinned with a version comment
#   1 — one or more references violate the policy
#   2 — no workflow files found (invocation error)

set -euo pipefail

WORKFLOWS_DIR=".github/workflows"
PATTERN='uses:[[:space:]]+[A-Za-z0-9_.-]+/([A-Za-z0-9_.-]+/)*[A-Za-z0-9_.-]+@[0-9a-f]{40}[[:space:]]*#[[:space:]]*v[A-Za-z0-9.]+'

workflow_files=$(find "$WORKFLOWS_DIR" -maxdepth 1 \( -name '*.yml' -o -name '*.yaml' \) 2>/dev/null | sort)
if [ -z "$workflow_files" ]; then
  echo "check-action-pinning: no workflow files found in $WORKFLOWS_DIR" >&2
  exit 2
fi

violations=0
while IFS= read -r match; do
  file="${match%%:*}"
  line="${match#*:}"
  line="${line%%:*}"
  content="$(sed -n "${line}p" "$file")"
  if ! printf '%s\n' "$content" | grep -Eq "$PATTERN"; then
    echo "check-action-pinning: ${file}:${line}: not SHA-pinned with a version comment:" >&2
    printf '    %s\n' "$content" >&2
    violations=$((violations + 1))
  fi
done < <(grep -rnE 'uses:[[:space:]]' $workflow_files)

if [ "$violations" -gt 0 ]; then
  echo "check-action-pinning: ${violations} unpinned action reference(s) found" >&2
  exit 1
fi
echo "check-action-pinning: all action references are SHA-pinned with version comments"
