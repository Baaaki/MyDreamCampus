#!/usr/bin/env bash
# Polls origin/<branch> and redeploys when it moves forward.
#
# Pull-based on purpose: a home server sits behind NAT, so GitHub cannot reach
# in to trigger a deploy. The server reaching out always works, needs no
# inbound port, and keeps no CI credentials on the box.
#
# Installed as a systemd *user* timer (`make autodeploy-install`) so it shares
# the rootless docker socket with the deploy itself.

set -euo pipefail

cd "$(dirname "$0")/.."

BRANCH="${DEPLOY_BRANCH:-main}"
STATE=".git/last-deployed-sha"

git fetch --quiet origin "$BRANCH"

remote=$(git rev-parse "origin/$BRANCH")
last=$(cat "$STATE" 2>/dev/null || echo none)

# Compare against the last SHA that deployed *successfully*, not against HEAD:
# if a build fails after the pull, the next tick must retry it rather than
# consider the commit already handled.
if [ "$remote" = "$last" ]; then
	exit 0
fi

echo ">> deploying ${last:0:7} -> ${remote:0:7}"

# --ff-only: never invent a merge commit on the server. Local edits there are
# a mistake worth failing loudly on.
git pull --ff-only origin "$BRANCH"
make deploy

echo "$remote" >"$STATE"
echo ">> deploy complete: $(git log -1 --oneline)"
