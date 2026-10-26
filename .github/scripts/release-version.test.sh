#!/usr/bin/env bash
# Tests release-version.sh against a throwaway git repository. CI runs it.
#
# Usage: .github/scripts/release-version.test.sh
set -euo pipefail

script="$(cd "$(dirname "$0")" && pwd)/release-version.sh"
repo=$(mktemp -d)
trap 'rm -rf "$repo"' EXIT
git -C "$repo" init -q

failures=0

commit() {
  git -C "$repo" -c user.name=test -c user.email=test@example.com \
    commit -q --allow-empty -m "$1"
}

# expect "<tag argument, or empty>" "<expected output on one line>"
expect() {
  local actual args=()
  [[ -n $1 ]] && args=("$1")
  actual=$(cd "$repo" && "$script" "${args[@]}" 2>&1 | tr '\n' ' ' | sed 's/ $//' || true)
  if [[ $actual == "$2" ]]; then
    echo "ok   '$1' -> $actual"
  else
    echo "FAIL '$1': expected '$2', got '$actual'"
    failures=$((failures + 1))
  fi
}

commit "fix: first"
expect "" "tag=v0.0.1 new=true"
commit "feat(events): add reminders"
expect "" "tag=v0.1.0 new=true"

git -C "$repo" tag v1.2.3
expect "" "tag=v1.2.3 new=false"

commit "fix(auth): b"
expect "" "tag=v1.2.4 new=true"

commit "docs: c

* feat!: a branch commit listed in the squash body"
expect "" "tag=v1.2.4 new=true"

commit "feat(api)!: drop the v1 routes"
expect "" "tag=v2.0.0 new=true"

git -C "$repo" tag v2.0.0
commit "chore: d

BREAKING CHANGE: config moved"
expect "" "tag=v3.0.0 new=true"

expect "v1.2.3" "tag=v1.2.3 new=false"
expect "v9.9.9" "::error::No tag v9.9.9"
git -C "$repo" tag 'v1.0;x'
expect "v1.0;x" "::error::v1.0;x is not a vX.Y.Z tag"

if ((failures > 0)); then
  echo "$failures test(s) failed"
  exit 1
fi
