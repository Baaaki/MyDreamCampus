#!/bin/sh
# Demo seed. Runs once, on an empty system: it becomes the first permanent
# state, which the super admin manages from the UI afterwards.
#
# People, courses and profiles go through the REAL admin API so the event
# chain fires and every service projection (incl. the auth login user) is
# populated correctly — a raw SQL insert would skip those projections.
# Reference data and scenario rows (periods, programs, grades, attendance,
# meals) are SQL, one file per service database; the handful of rows another
# database owns travel as CSV staging tables (sql/_staging-*.sql).
#
# The API half talks to each service directly rather than through Caddy:
# PUBLIC_HOST without a scheme turns Caddy's automatic HTTPS on, so :80 would
# answer every request with a 308. Nothing about seeding needs the edge.
#
# Auth: login returns access_token in the body; the API accepts
# `Authorization: Bearer` and skips CSRF for header-auth.
#
# Any API error other than 409 aborts the run. A seed that swallows errors
# "completes" against a system it never wrote to — which is how a login-flag
# change once left every request at 403 while CI stayed green on the seed step.
# 409 is "already there": a re-run after a failure resumes instead of stopping
# at the first record it created last time.
set -eu

AUTH_URL="${AUTH_SERVICE_URL:-http://auth-service:8081}"
STAFF_URL="${STAFF_SERVICE_URL:-http://staff-service:8082}"
STUDENT_URL="${STUDENT_SERVICE_URL:-http://student-service:8083}"
CATALOG_URL="${CATALOG_SERVICE_URL:-http://catalog-service:8084}"
: "${ADMIN_EMAIL:?ADMIN_EMAIL required}"
: "${ADMIN_INITIAL_PASSWORD:?ADMIN_INITIAL_PASSWORD required}"
: "${PG_HOST:=postgres}"
DATA=/seed/data
SQL=/seed/sql
WORK=/tmp/seed-work
STAGE=/tmp/seed-stage

if [ "${SEED_DEMO:-true}" != "true" ]; then
	echo "SEED_DEMO != true — skipping demo seed."
	exit 0
fi
# The relational half is most of the demo, and the admin's first-login flag
# can only be cleared in the auth database: without it there is no seed.
: "${SERVICE_DB_PASSWORD:?SERVICE_DB_PASSWORD required — the seed writes to every service database}"

STARTED=$(date +%s)
mkdir -p "$WORK" "$STAGE"
RESP="$WORK/resp"
HDRS="$WORK/headers"

# Same derivation the migrator uses: database `auth` -> role `auth_svc`.
svc_url() {
	echo "postgres://${1}_svc:${SERVICE_DB_PASSWORD}@${PG_HOST}:5432/${1}?sslmode=disable"
}

# q DB SQL — one value, unaligned.
q() {
	psql "$(svc_url "$1")" -v ON_ERROR_STOP=1 -Atqc "$2"
}

# --- 1. wait for the services this seed writes through ---
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

# --- 2. only an empty system is seeded ---
# Students exist only once someone — this seed or an admin — has put people
# in; either way this is no longer the empty system the seed describes.
if [ "$(q student "SELECT EXISTS (SELECT 1 FROM student.students)")" = "t" ]; then
	echo ">> system already holds data — skipping the demo seed."
	exit 0
fi

# --- 3. admin session ---
# The bootstrap admin starts with force_password_change, and the flag rides in
# the JWT: every call below would answer 403 FORCE_PASSWORD_CHANGE. It has to
# be cleared before the login that mints the token.
q auth "UPDATE auth.users SET force_password_change = false WHERE email = '$(printf '%s' "$ADMIN_EMAIL" | sed "s/'/''/g")'" >/dev/null
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

# api METHOD URL JSON — 2xx and 409 pass, anything else aborts. 429 waits out
# the limiter (catalog alone takes ~120 calls, past the per-IP budget of 100 a
# minute) and a refused connection or 5xx is retried a few times.
api() {
	_method="$1"; _url="$2"; _json="$3"; _try=0
	while :; do
		_code=$(curl -sS -o "$RESP" -D "$HDRS" -w '%{http_code}' -X "$_method" "$_url" \
			-H "$AUTH" -H 'Content-Type: application/json' -d "$_json") || _code=000
		case "$_code" in
		2??) return 0 ;;
		409) return 0 ;;
		429 | 000 | 502 | 503 | 504)
			_try=$((_try + 1))
			[ "$_try" -le 30 ] || break
			_wait=$(awk 'tolower($1) == "retry-after:" { print $2 + 0 }' "$HDRS" 2>/dev/null)
			sleep "${_wait:-2}"
			;;
		*) break ;;
		esac
	done
	echo "!! [$_code] $_method $_url -> $(head -c 300 "$RESP")"
	exit 1
}

# send METHOD ROWS — one `api` call per line of ROWS, each "<url>\t<json>".
# Reading a file instead of a pipe keeps the loop in this shell, so api's exit
# stops the whole seed.
send() {
	_n=0
	while IFS="	" read -r _url _json; do
		api "$1" "$_url" "$_json"
		_n=$((_n + 1))
	done <"$2"
	echo "   $_n × $1"
}

# --- 4. faculties and departments (catalog reference data) ---
# Before any person or course: they name their faculty and department, and the
# pickers built on this table are how an admin creates the next ones.
echo ">> seeding faculties and departments"
psql "$(svc_url catalog)" -v ON_ERROR_STOP=1 -q \
	-v faculties="$(cat "$DATA/faculties.json")" -f "$SQL/00-catalog-reference.sql"

# --- 5. teachers and their profiles ---
echo ">> creating teachers"
jq -r '.[] | "'"$STAFF_URL"'/api/staff\t\(tojson)"' "$DATA/teachers.json" >"$WORK/rows"
send POST "$WORK/rows"
q staff "SELECT json_object_agg(email, id) FROM staff.staff" >"$WORK/staff_ids.json"

echo ">> writing teacher profiles"
jq -r --slurpfile ids "$WORK/staff_ids.json" \
	'.[] | "'"$STAFF_URL"'/api/staff/\($ids[0][.email] // error("no staff id for \(.email)"))/profile\t\(.profile | tojson)"' \
	"$DATA/teacher_profiles.json" >"$WORK/rows"
send PUT "$WORK/rows"

# --- 6. courses, then their prerequisites ---
# Prerequisites point at catalog ids, which exist only once every course does.
# catalog also requires a prerequisite to sit in a LOWER class level; the
# mock curriculum chains same-year courses (Matematik I -> II), and those
# links are left out rather than failing the run — the list is printed.
echo ">> creating courses"
jq -r '.[] | "'"$CATALOG_URL"'/api/catalog/courses\t\(del(.prerequisites) | tojson)"' "$DATA/courses.json" >"$WORK/rows"
send POST "$WORK/rows"
q catalog "SELECT json_object_agg(course_code, json_build_object('id', id, 'class_level', class_level)) FROM course_catalog.course_catalog" >"$WORK/course_ids.json"

echo ">> linking prerequisites"
jq -r --slurpfile c "$WORK/course_ids.json" '
	.[] | . as $course
	| ($course.prerequisites | map(select($c[0][.course_code].class_level < $course.class_level))) as $ok
	| select($ok | length > 0)
	| "'"$CATALOG_URL"'/api/catalog/courses/\($course.course_code | @uri)\t\({prerequisites: $ok | map({id: $c[0][.course_code].id} + .)} | tojson)"
' "$DATA/courses.json" >"$WORK/rows"
send PUT "$WORK/rows"
jq -r --slurpfile c "$WORK/course_ids.json" '
	.[] | . as $course | .prerequisites[]
	| select($c[0][.course_code].class_level >= $course.class_level)
	| "   skipped (same class level): \($course.course_code) <- \(.course_code)"
' "$DATA/courses.json"

# --- 7. students, with their advisors ---
echo ">> creating students"
jq -r --slurpfile ids "$WORK/staff_ids.json" '
	.[] | "'"$STUDENT_URL"'/api/students\t\(
		del(.advisor_email, .status)
		+ (if .advisor_email then {advisor_id: ($ids[0][.advisor_email] // error("no staff id for \(.advisor_email)"))} else {} end)
		| tojson)"
' "$DATA/students.json" >"$WORK/rows"
send POST "$WORK/rows"
q student "SELECT json_object_agg(student_number, id) FROM student.students" >"$WORK/student_ids.json"
jq -r --slurpfile ids "$WORK/student_ids.json" \
	'.[] | select(.status) | "'"$STUDENT_URL"'/api/students/\($ids[0][.student_number])\t\({status} | tojson)"' \
	"$DATA/students.json" >"$WORK/rows"
send PUT "$WORK/rows"

# --- 8. administrative staff ---
echo ">> creating administrative staff"
jq -r '.[] | "'"$STAFF_URL"'/api/admin-staff\t\(tojson)"' "$DATA/admin_staff.json" >"$WORK/rows"
send POST "$WORK/rows"

# --- 9. login users ---
# auth writes a user per staff/student creation event, asynchronously. The
# flag below can only be cleared once those rows exist.
EMAILS=$(jq -rs '[.[0][], .[1][] | .email] | map("'"'"'" + . + "'"'"'") | join(",")' \
	"$DATA/teachers.json" "$DATA/students.json")
WANT=$(jq -s '(.[0] | length) + (.[1] | length)' "$DATA/teachers.json" "$DATA/students.json")
echo ">> waiting for $WANT login users"
i=0
until [ "$(q auth "SELECT count(*) FROM auth.users WHERE email IN ($EMAILS)")" -ge "$WANT" ]; do
	i=$((i + 1))
	if [ "$i" -gt 60 ]; then
		echo "!! auth did not receive every creation event within 120s"
		exit 1
	fi
	sleep 2
done
# Provisioned users get password = e-mail and must change it on first login.
# Demo visitors share these accounts, so the prompt is turned off for the
# exact seeded addresses only — a domain wildcard would reach real users too.
q auth "UPDATE auth.users SET force_password_change = false WHERE email IN ($EMAILS)" >/dev/null

# --- 10. scenario data, one database at a time ---
run_sql() {
	echo ">> seeding db:$1"
	psql "$(svc_url "$1")" -v ON_ERROR_STOP=1 -q -f "$SQL/$2"
}

# Order is a dependency chain, not cosmetics: catalog reads staff ids, and the
# four service seeds read catalog's offerings and periods. Each export runs
# after the database it reads from has been seeded.
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
echo ">> seeding db:meal"
psql "$(svc_url meal)" -v ON_ERROR_STOP=1 -q \
	-v cafeterias="$(cat "$DATA/cafeterias.json")" -f "$SQL/06-meal.sql"

echo ">> seed complete in $(($(date +%s) - STARTED))s. Demo login = e-posta / e-posta (ör. zeynep.sahin@uni.edu.tr)."
