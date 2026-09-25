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

# Gating: deploy only commits where CI has passed
if [ -z "${GITHUB_REPOSITORY:-}" ]; then
	origin_url=$(git config --get remote.origin.url || true)
	GITHUB_REPOSITORY=$(echo "$origin_url" | sed -E -n 's#.*github\.com[:/]([^/]+/[^/.]+)(\.git)?.*#\1#p')
fi

if [ -n "$GITHUB_REPOSITORY" ]; then
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
		echo ">> commit ${remote:0:7}: GitHub API kontrolu basarisiz (HTTP $http_code). Sonraki turda tekrar denenecek."
		exit 0
	fi

	count=$(echo "$body" | jq -r '.total_count // 0' 2>/dev/null || echo 0)
	if [ "$count" -eq 0 ]; then
		echo ">> commit ${remote:0:7}: CI bekliyor (ci-passed henuz olusmadi)"
		exit 0
	fi

	status=$(echo "$body" | jq -r '.check_runs | sort_by(.id) | reverse | .[0].status // empty' 2>/dev/null)
	conclusion=$(echo "$body" | jq -r '.check_runs | sort_by(.id) | reverse | .[0].conclusion // empty' 2>/dev/null)

	if [ "$status" != "completed" ]; then
		echo ">> commit ${remote:0:7}: CI bekliyor (ci-passed durumu: $status)"
		exit 0
	fi

	if [ "$conclusion" != "success" ]; then
		echo ">> commit ${remote:0:7}: CI basarisiz (ci-passed sonucu: $conclusion)"
		exit 0
	fi

	echo ">> commit ${remote:0:7}: CI basarili (ci-passed: success)"
else
	echo ">> uyari: GITHUB_REPOSITORY cozumlenemedi, CI kontrolu atlandi."
fi

echo ">> deploying ${last:0:7} -> ${remote:0:7}"

# --ff-only: never invent a merge commit on the server. Local edits there are
# a mistake worth failing loudly on.
git pull --ff-only origin "$BRANCH"
make deploy

echo "$remote" >"$STATE"
echo ">> deploy complete: $(git log -1 --oneline)"

