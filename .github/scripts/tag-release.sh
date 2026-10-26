#!/usr/bin/env bash
# Releases a commit without rebuilding: the images CI pushed for it get the
# release tag, then the commit gets the git tag. CI pushes images only after
# every check passes on main, so a missing image stops the release before the
# commit is tagged.
#
# Usage: tag-release.sh TAG COMMIT
#   tag-release.sh v1.3.0 1fc361e5...   (full SHA, as in $GITHUB_SHA)
#
# Copies, for example, ghcr.io/jonathan-skantz/jonska-calendar-api:sha-1fc361e5...
# to ghcr.io/jonathan-skantz/jonska-calendar-api:v1.3.0, and the same for
# -migrate and -web. Needs $GITHUB_REPOSITORY and a logged-in GHCR session.
set -euo pipefail

tag=$1
commit=$2

for name in api migrate web; do
  image="ghcr.io/$GITHUB_REPOSITORY-$name"
  # Only the manifest is copied; the layers are already in the registry.
  if ! docker buildx imagetools create -t "$image:$tag" "$image:sha-$commit"; then
    echo "::error::No $name image for $commit. Wait for CI on main to pass." >&2
    exit 1
  fi
done

git tag "$tag" "$commit"
git push origin "$tag"
