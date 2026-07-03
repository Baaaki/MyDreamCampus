#!/bin/sh
# Demo seed. Creates teachers, courses and students through the REAL admin API
# so the event chain fires and every module projection (incl. the auth login
# user) is populated correctly — a raw SQL insert would skip those projections.
#
# Auth: login returns access_token in the body; the API accepts
# `Authorization: Bearer` and skips CSRF for header-auth, so no cookie/CSRF
# dance is needed even over plain internal HTTP.
#
# Provisioned users get password = their email and force_password_change=true.
# We flip that flag off for the demo accounts at the end (direct DB update on a
# projection flag — no event needed) so a demo login isn't interrupted.
set -eu

API="${API_URL:-http://monolith:8080}"
: "${ADMIN_EMAIL:?ADMIN_EMAIL required}"
: "${ADMIN_INITIAL_PASSWORD:?ADMIN_INITIAL_PASSWORD required}"

if [ "${SEED_DEMO:-true}" != "true" ]; then
	echo "SEED_DEMO != true — skipping demo seed."
	exit 0
fi

# --- 1. wait for the monolith to be reachable ---
echo ">> waiting for monolith at $API/health ..."
i=0
until curl -fsS "$API/health" >/dev/null 2>&1; do
	i=$((i + 1))
	if [ "$i" -gt 60 ]; then
		echo "!! monolith did not become healthy in time"
		exit 1
	fi
	sleep 2
done

# --- 2. login as admin ---
echo ">> logging in as $ADMIN_EMAIL"
LOGIN_BODY=$(curl -fsS -X POST "$API/api/auth/login" \
	-H 'Content-Type: application/json' \
	-d "$(jq -n --arg e "$ADMIN_EMAIL" --arg p "$ADMIN_INITIAL_PASSWORD" '{email:$e,password:$p}')")
TOKEN=$(printf '%s' "$LOGIN_BODY" | jq -r '.access_token // empty')
if [ -z "$TOKEN" ]; then
	echo "!! admin login failed: $LOGIN_BODY"
	exit 1
fi
AUTH="Authorization: Bearer $TOKEN"

# --- 3. idempotency: skip if the first demo student already exists ---
if curl -fsS "$API/api/students?limit=200" -H "$AUTH" 2>/dev/null | grep -q "2021510001"; then
	echo ">> demo data already present — skipping."
	exit 0
fi

# POST helper: never aborts the run on a single failure (unique-constraint 409s
# on re-run are expected and harmless). Prints status + a snippet on error.
post() {
	_path="$1"; _json="$2"
	_code=$(curl -sS -o /tmp/resp -w '%{http_code}' -X POST "$API/$_path" \
		-H "$AUTH" -H 'Content-Type: application/json' -d "$_json")
	if [ "$_code" -ge 400 ]; then
		echo "   [$_code] POST /$_path  ->  $(head -c 200 /tmp/resp)"
	else
		echo "   [$_code] POST /$_path"
	fi
	return 0
}

echo ">> creating teachers"
jq -c '.[]' /seed/data/teachers.json | while IFS= read -r row; do post "api/staff" "$row"; done

echo ">> creating courses"
jq -c '.[]' /seed/data/courses.json | while IFS= read -r row; do post "api/catalog/courses" "$row"; done

echo ">> creating students"
jq -c '.[]' /seed/data/students.json | while IFS= read -r row; do post "api/students" "$row"; done

# --- 4. let async event projections drain, then unblock demo logins ---
if [ -n "${DB_URL:-}" ]; then
	echo ">> waiting for login projections, then clearing force_password_change"
	sleep 8
	# Scope strictly to the exact seeded emails (+ admin). A domain wildcard
	# would also disable the flag for real users sharing the domain.
	EMAILS=$(
		{ jq -rs 'add | map(.email) | .[]' /seed/data/teachers.json /seed/data/students.json
		  printf '%s\n' "$ADMIN_EMAIL"; } \
		| while IFS= read -r e; do [ -n "$e" ] && printf "'%s'," "$e"; done \
		| sed 's/,$//'
	)
	psql "$DB_URL" -v ON_ERROR_STOP=1 -c \
		"UPDATE auth.users SET force_password_change = false WHERE email IN ($EMAILS);" \
		|| echo "   (force_password_change update skipped: $?)"
fi

echo ">> seed complete. Demo login = e-posta / e-posta (ör. zeynep.sahin@uni.edu.tr)."
