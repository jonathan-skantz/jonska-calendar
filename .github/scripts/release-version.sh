#!/usr/bin/env bash
# Picks the tag the Release workflow deploys, and prints it in $GITHUB_OUTPUT
# format. `new` says whether the tag still has to be created.
#
# Usage:
#   release-version.sh          # release HEAD: reuse its tag or pick the next one
#   release-version.sh v1.2.0   # redeploy an existing tag, e.g. to roll back
#
# Output:
#   tag=v1.3.0
#   new=true
#
# The next version comes from the commit messages since the last vX.Y.Z tag.
# PRs are squash-merged, so each commit on main starts with its PR title. The
# biggest bump wins:
#
#   feat(api)!: drop the v1 routes       major   v1.2.3 -> v2.0.0
#   any message with a line starting
#   "BREAKING CHANGE:"                   major   v1.2.3 -> v2.0.0
#   feat(events): add reminders          minor   v1.2.3 -> v1.3.0
#   fix: ..., docs: ..., anything else   patch   v1.2.3 -> v1.2.4
#
# Before the first release the base is v0.0.0, so a history with a `feat`
# becomes v0.1.0.
set -euo pipefail

next_version() {
  local last log major minor patch
  last=$(git describe --tags --abbrev=0 --match 'v*.*.*' 2>/dev/null || true)
  # "v1.2.3..HEAD", or the whole history before the first release.
  log=$(git log --format=%B "${last:+$last..}HEAD")
  IFS=. read -r major minor patch <<< "${last:-v0.0.0}"
  major=${major#v}

  # Patterns are anchored to the line start. That matches subjects and
  # footers, but not the "* feat: ..." lines GitHub lists in a squash body.
  if grep -qE '^[a-z]+(\([^)]*\))?!:|^BREAKING[ -]CHANGE:' <<< "$log"; then
    echo "v$((major + 1)).0.0"
  elif grep -qE '^feat(\([^)]*\))?:' <<< "$log"; then
    echo "v$major.$((minor + 1)).0"
  else
    echo "v$major.$minor.$((patch + 1))"
  fi
}

fail() {
  echo "::error::$1" >&2
  exit 1
}

tag=${1:-}
new=false
if [[ -n $tag ]]; then
  git rev-parse -q --verify "refs/tags/$tag" > /dev/null || fail "No tag $tag"
else
  # Running Release twice on the same commit redeploys instead of bumping.
  tag=$(git tag --points-at HEAD --list 'v*.*.*' | head -n 1)
  if [[ -z $tag ]]; then
    tag=$(next_version)
    new=true
  fi
fi

# The tag ends up in a shell command on the server, so allow nothing else.
[[ $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "$tag is not a vX.Y.Z tag"

echo "tag=$tag"
echo "new=$new"
