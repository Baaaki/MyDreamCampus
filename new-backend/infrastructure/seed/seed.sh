#!/bin/sh
# Demo seed. Creates teachers, administrative staff, courses and students
# through the REAL admin API so the event chain fires and every service
# projection (incl. the auth login user) is populated correctly — a raw SQL
# insert would skip those projections.
#
# The API half talks to each service directly rather than through Caddy. Going
# through the gateway looked tidier but breaks on the edge's own behaviour:
# PUBLIC_HOST without a scheme turns Caddy's automatic HTTPS on, so :80 answers
# every request with a 308 to https://, and the seed would have to follow a
# redirect to a certificate issued for a different name. Nothing about seeding
# needs the edge — these are four internal addresses on the compose network.
#
# The relational half used to be one seed.sql against one database. The schemas
# are separate databases today, so it is one file per database, ordered, with
# the handful of rows that used to be a cross-schema JOIN carried between them
# as CSV staging tables (sql/_staging-*.sql).
#
# Auth: login returns access_token in the body; the API accepts
# `Authorization: Bearer` and skips CSRF for header-auth, so no cookie/CSRF
# dance is needed even over plain internal HTTP.
#
# Provisioned users get password = their email and force_password_change=true.
# We flip that flag off for the demo accounts at the end (direct DB update on a
# projection flag — no event needed) so a demo login isn't interrupted.
set -eu

AUTH_URL="${AUTH_SERVICE_URL:-http://auth-service:8081}"
STAFF_URL="${STAFF_SERVICE_URL:-http://staff-service:8082}"
STUDENT_URL="${STUDENT_SERVICE_URL:-http://student-service:8083}"
CATALOG_URL="${CATALOG_SERVICE_URL:-http://catalog-service:8084}"
: "${ADMIN_EMAIL:?ADMIN_EMAIL required}"
: "${ADMIN_INITIAL_PASSWORD:?ADMIN_INITIAL_PASSWORD required}"

if [ "${SEED_DEMO:-true}" != "true" ]; then
	echo "SEED_DEMO != true — skipping demo seed."
	exit 0
fi

# --- 1. wait for the four services this seed writes through ---
for url in "$AUTH_URL" "$STAFF_URL" "$STUDENT_URL" "$CATALOG_URL"; do
	echo ">> waiting for $url/health ..."
	i=0
	until curl -fsS "$url/health" >/dev/null 2>&1; do
		i=$((i + 1))
		if [ "$i" -gt 60 ]; then
			echo "!! $url did not become healthy in time"
			exit 1
		fi
		sleep 2
	done
done

# --- 2. login as admin ---
echo ">> logging in as $ADMIN_EMAIL"
LOGIN_BODY=$(curl -fsS -X POST "$AUTH_URL/api/auth/login" \
	-H 'Content-Type: application/json' \
	-d "$(jq -n --arg e "$ADMIN_EMAIL" --arg p "$ADMIN_INITIAL_PASSWORD" '{email:$e,password:$p}')")
TOKEN=$(printf '%s' "$LOGIN_BODY" | jq -r '.access_token // empty')
if [ -z "$TOKEN" ]; then
	echo "!! admin login failed: $LOGIN_BODY"
	exit 1
fi
AUTH="Authorization: Bearer $TOKEN"

# POST helper: never aborts the run on a single failure (unique-constraint 409s
# on re-run are expected and harmless). Prints status + a snippet on error.
post() {
	_url="$1"; _json="$2"
	_code=$(curl -sS -o /tmp/resp -w '%{http_code}' -X POST "$_url" \
		-H "$AUTH" -H 'Content-Type: application/json' -d "$_json")
	if [ "$_code" -ge 400 ]; then
		echo "   [$_code] POST $_url  ->  $(head -c 200 /tmp/resp)"
	else
		echo "   [$_code] POST $_url"
	fi
	return 0
}

# Seeded before the idempotency check below: administrative staff joined the
# seed later, and behind the check a deployment seeded before that would never
# get any. Records are unique by e-mail, so re-runs only produce harmless 409s.
echo ">> creating administrative staff"
jq -c '.[]' /seed/data/admin_staff.json | while IFS= read -r row; do post "$STAFF_URL/api/admin-staff" "$row"; done

# --- 3. idempotency: skip if the first demo student already exists ---
if curl -fsS "$STUDENT_URL/api/students?limit=200" -H "$AUTH" 2>/dev/null | grep -q "2021510001"; then
	echo ">> demo data already present — skipping."
	exit 0
fi

echo ">> creating teachers"
jq -c '.[]' /seed/data/teachers.json | while IFS= read -r row; do post "$STAFF_URL/api/staff" "$row"; done

echo ">> creating courses"
jq -c '.[]' /seed/data/courses.json | while IFS= read -r row; do post "$CATALOG_URL/api/catalog/courses" "$row"; done

echo ">> creating students"
jq -c '.[]' /seed/data/students.json | while IFS= read -r row; do post "$STUDENT_URL/api/students" "$row"; done

# --- 4. relational demo data, one database at a time ---
if [ -z "${SERVICE_DB_PASSWORD:-}" ]; then
	echo ">> SERVICE_DB_PASSWORD unset — skipping the relational seed."
	echo ">> seed complete (API only)."
	exit 0
fi

: "${PG_HOST:=postgres}"
STAGE=/tmp/seed-stage
mkdir -p "$STAGE"

# Same derivation the migrator uses: database `auth` -> role `auth_svc`.
svc_url() {
	echo "postgres://${1}_svc:${SERVICE_DB_PASSWORD}@${PG_HOST}:5432/${1}?sslmode=disable"
}

run_sql() {
	echo ">> seeding db:$1"
	psql "$(svc_url "$1")" -v ON_ERROR_STOP=1 -q -f "/seed/sql/$2"
}

# The projections the API seed's events fill (auth users, the *_view tables)
# arrive asynchronously. Give them a moment before reading student rows back.
echo ">> waiting for event projections to drain"
sleep 8

echo ">> clearing force_password_change for the demo accounts"
# Scope strictly to the exact seeded emails (+ admin). A domain wildcard would
# also disable the flag for real users sharing the domain.
EMAILS=$(
	{ jq -rs 'add | map(.email) | .[]' /seed/data/teachers.json /seed/data/students.json
	  printf '%s\n' "$ADMIN_EMAIL"; } \
	| while IFS= read -r e; do [ -n "$e" ] && printf "'%s'," "$e"; done \
	| sed 's/,$//'
)
psql "$(svc_url auth)" -v ON_ERROR_STOP=1 -c \
	"UPDATE auth.users SET force_password_change = false WHERE email IN ($EMAILS);" \
	|| echo "   (force_password_change update skipped: $?)"

# Order is a dependency chain, not cosmetics: catalog reads staff ids, and the
# four service seeds read catalog's offerings and periods. Each export runs
# after the database it reads from has been seeded.
run_sql staff 01-staff.sql

echo ">> exporting staff rows for catalog"
psql "$(svc_url staff)" -v ON_ERROR_STOP=1 -q -c \
	"\copy (SELECT id, email, first_name, last_name FROM staff.staff) TO '$STAGE/staff.csv' CSV"

run_sql catalog 02-catalog.sql

echo ">> exporting student and catalog rows for the remaining services"
psql "$(svc_url student)" -v ON_ERROR_STOP=1 -q -c \
	"\copy (SELECT id, student_number, first_name, last_name, email, department, class_level, is_active FROM student.students) TO '$STAGE/students.csv' CSV"
psql "$(svc_url catalog)" -v ON_ERROR_STOP=1 -q -c \
	"\copy (SELECT id, course_code, name, credits, department, class_level FROM course_catalog.course_catalog) TO '$STAGE/courses.csv' CSV"
psql "$(svc_url catalog)" -v ON_ERROR_STOP=1 -q -c \
	"\copy (SELECT id, semester, course_code, credits, class_level, instructor_id, instructor_fullname, assessment_schema FROM course_catalog.semester_courses) TO '$STAGE/semester_courses.csv' CSV"
psql "$(svc_url catalog)" -v ON_ERROR_STOP=1 -q -c \
	"\copy (SELECT id, semester, period_start, period_end, is_active, period_type FROM course_catalog.academic_periods) TO '$STAGE/periods.csv' CSV"

run_sql grades     03-grades.sql
run_sql attendance 04-attendance.sql
run_sql enrollment 05-enrollment.sql
run_sql meal       06-meal.sql

echo ">> seed complete. Demo login = e-posta / e-posta (ör. zeynep.sahin@uni.edu.tr)."
