#!/bin/sh
# Runs on the server from a checkout of the release tag (see
# .github/workflows/release.yml). Pulls that release's images and restarts the
# services whose image changed; migrations run before the API starts.
set -eu

if [ $# -ne 1 ]; then
  echo "usage: $0 vX.Y.Z" >&2
  exit 2
fi

export APP_VERSION="$1"
cd "$(dirname "$0")"
docker compose --env-file .env.prod pull migrate api web
docker compose --env-file .env.prod up -d --no-build --remove-orphans
# Older releases keep their tags, so plain `prune` would never remove them.
# A rollback pulls its images again.
docker image prune -af
