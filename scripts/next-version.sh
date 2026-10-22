#!/usr/bin/env bash
# Prints the next release tag for MAJOR.MINOR (from VERSION), reading existing tags on stdin.
# Usage: git tag | scripts/next-version.sh "$(cat VERSION)"
set -euo pipefail
base=$(tr -d '[:space:]' <<<"${1:-}")
if [[ ! $base =~ ^[0-9]+\.[0-9]+$ ]]; then
  echo "VERSION must be MAJOR.MINOR, got '$base'" >&2
  exit 2
fi

next=0
while read -r tag; do
  if [[ $tag =~ ^v${base//./\\.}\.([0-9]+)$ ]] && ((BASH_REMATCH[1] >= next)); then
    next=$((BASH_REMATCH[1] + 1))
  fi
done
echo "v$base.$next"
