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

# Deploy only a commit whose `ci-passed` check run succeeded. Every way out
# short of that exits before the pull, so the next tick asks again: pending
# and failed runs, an unreachable API, and a repo this script cannot name.
# The last one fails the unit instead of deploying unchecked — a gate that
# opens when it cannot look is no gate.
if ! command -v jq >/dev/null; then
	echo "!! jq is not installed, so the CI gate cannot read GitHub's answer (apt install jq)" >&2
	exit 1
fi

if [ -z "${GITHUB_REPOSITORY:-}" ]; then
	origin_url=$(git config --get remote.origin.url || true)
	GITHUB_REPOSITORY=$(echo "$origin_url" | sed -E -n 's#^.*github\.com[:/]([^/]+/[^/]+)/?$#\1#p' | sed 's/\.git$//')
fi
if [ -z "$GITHUB_REPOSITORY" ]; then
	echo "!! cannot tell the GitHub repo from origin; set GITHUB_REPOSITORY=owner/name" >&2
	exit 1
fi

auth_args=()
if [ -n "${GITHUB_TOKEN:-}" ]; then
	auth_args=(-H "Authorization: Bearer $GITHUB_TOKEN")
fi

api_url="https://api.github.com/repos/${GITHUB_REPOSITORY}/commits/${remote}/check-runs?check_name=ci-passed"
resp=$(curl -sS -w "\n%{http_code}" "${auth_args[@]}" \
	-H "Accept: application/vnd.github+json" \
	-H "X-GitHub-Api-Version: 2022-11-28" \
	"$api_url" 2>/dev/null || true)

http_code=$(echo "$resp" | tail -n1)
body=$(echo "$resp" | sed '$d')

if [ "$http_code" != "200" ]; then
	echo ">> ${remote:0:7}: GitHub API answered HTTP $http_code; retrying next tick"
	exit 0
fi

count=$(echo "$body" | jq -r '.total_count // 0')
if [ "$count" -eq 0 ]; then
	echo ">> ${remote:0:7}: waiting for CI (no ci-passed run yet)"
	exit 0
fi

latest=$(echo "$body" | jq -c '.check_runs | sort_by(.id) | last')
status=$(echo "$latest" | jq -r '.status // empty')
conclusion=$(echo "$latest" | jq -r '.conclusion // empty')

if [ "$status" != "completed" ]; then
	echo ">> ${remote:0:7}: waiting for CI (ci-passed is $status)"
	exit 0
fi

if [ "$conclusion" != "success" ]; then
	echo ">> ${remote:0:7}: not deploying, ci-passed concluded $conclusion"
	exit 0
fi

echo ">> ${remote:0:7}: ci-passed succeeded"

echo ">> deploying ${last:0:7} -> ${remote:0:7}"

# --ff-only: never invent a merge commit on the server. Local edits there are
# a mistake worth failing loudly on.
git pull --ff-only origin "$BRANCH"
make deploy

echo "$remote" >"$STATE"
echo ">> deploy complete: $(git log -1 --oneline)"
