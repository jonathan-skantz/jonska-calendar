#!/usr/bin/env bash
# Checks that a deployed site is up: Caddy serves the page, and the API
# answers /api/auth/me with 401 (no session), which proves Caddy reaches it.
# Retries for about a minute while the containers start.
#
# Usage: smoke-check.sh SITE_URL
#   smoke-check.sh https://cal.example.com
set -euo pipefail

site=${1%/}

for attempt in 1 2 3 4 5 6; do
  api_status=$(curl -s -o /dev/null -w '%{http_code}' "$site/api/auth/me" || true)
  if curl -fsS -o /dev/null "$site/" && [[ $api_status == 401 ]]; then
    echo "Site and API are up"
    exit 0
  fi
  echo "Attempt $attempt: /api/auth/me returned $api_status, expected 401"
  sleep 10
done

echo "::error::$site is not healthy after the deploy" >&2
exit 1
