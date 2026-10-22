#!/usr/bin/env bash
# Tests next-version.sh: run with `bash scripts/next-version_test.sh`.
set -euo pipefail
cd "$(dirname "$0")"
fails=0

expect() { # expect <want> <base> <tags...>
  local want=$1 base=$2 got
  shift 2
  got=$(printf '%s\n' "$@" | ./next-version.sh "$base")
  if [[ $got != "$want" ]]; then
    echo "FAIL: base $base, tags [$*]: want $want, got $got"
    fails=$((fails + 1))
  fi
}

expect v0.1.0 0.1
expect v0.1.0 0.1 v0.0.4 v1.0.0
expect v0.1.3 0.1 v0.1.0 v0.1.2 v0.1.1
expect v0.1.11 0.1 v0.1.9 v0.1.10
expect v0.2.0 0.2 v0.1.7
expect v1.0.1 1.0 v1.0.0 v1.0.0-rc1 v1.0x.5 notes
expect v0.1.0 " 0.1 "

if ./next-version.sh 0.1.2 </dev/null >/dev/null 2>&1; then
  echo "FAIL: accepted a base with a patch number"
  fails=$((fails + 1))
fi

[[ $fails == 0 ]] && echo "next-version: ok"
exit "$fails"
