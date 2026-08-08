#!/bin/sh
# Apply goose migrations. Two targets:
#
#   1. Legacy — the single `mydreamcampus` database ($DB_URL) and the separate
#      notification database ($NOTIF_DB_URL). Runs only while those vars are
#      set; the monolith still boots against them until phase 4 of the
#      microservices migration. Dropping the vars turns this half off.
#   2. Per-service — one database per service, each migrated by its OWN role
#      (auth_svc, staff_svc, ...) so the tables end up owned by that role.
#
# Module order matches the monolith Makefile (migrate-up-all). Each module keeps
# its own goose version table, in both halves.
set -e

: "${PG_HOST:=postgres}"
: "${SERVICE_DB_PASSWORD:?SERVICE_DB_PASSWORD is required}"

# Wait for the server to accept connections before goose touches it. Compose's
# `depends_on: condition: service_healthy` already guarantees this, but that
# condition is a compose-only concept: a PaaS that reads this file (Openship)
# keeps the dependency EDGE and drops the condition, so migrate can win the
# race against an initdb that is still running. With `restart: "no"` a failure
# here is terminal and the schema never lands, so the gate has to be in-script.
wait_for_db() {
	label="$1"
	url="$2"
	i=0
	until pg_isready -d "$url" >/dev/null 2>&1; do
		i=$((i + 1))
		if [ "$i" -gt 60 ]; then
			echo "!! $label did not accept connections within 120s"
			exit 1
		fi
		[ "$i" = 1 ] && echo ">> waiting for $label ..."
		sleep 2
	done
}

# DSN of a service, connecting as that service's own role. Database and role
# name are derived from the same word: db `auth` -> user `auth_svc`.
svc_url() {
	echo "postgres://${1}_svc:${SERVICE_DB_PASSWORD}@${PG_HOST}:5432/${1}?sslmode=disable"
}

# ── Legacy: monolith + standalone notification ───────────────────────────────

if [ -n "$DB_URL" ]; then
	wait_for_db "monolith database" "$DB_URL"
	for module in auth staff student course_catalog enrollment attendance grades meal payment; do
		dir="/migrations/modules/$module"
		[ -d "$dir" ] || continue
		echo ">> goose up (monolith) — $module"
		goose -dir "$dir" -table "goose_db_version_$module" postgres "$DB_URL" up
	done
fi

if [ -n "$NOTIF_DB_URL" ]; then
	wait_for_db "legacy notification database" "$NOTIF_DB_URL"
	echo ">> goose up (legacy notification)"
	goose -dir /migrations/notification -table goose_db_version_notification postgres "$NOTIF_DB_URL" up
fi

# ── Per-service databases ────────────────────────────────────────────────────

wait_for_db "postgres server" "$(svc_url auth)"

# pg_isready only proves the SERVER is up — it neither authenticates nor checks
# that the database exists. The service databases are created by
# postgres/init-databases.sh, which the Postgres entrypoint runs ONLY on an
# empty volume, so on an upgraded stack they are simply absent and goose's bare
# error would not say why.
if ! psql "$(svc_url auth)" -c 'SELECT 1' >/dev/null 2>&1; then
	echo "!! cannot open the auth database as auth_svc."
	echo "   The per-service databases are provisioned on the FIRST boot of an"
	echo "   empty postgres volume. On an existing volume, run it by hand:"
	echo "     docker cp new-backend/infrastructure/postgres/init-databases.sh mydreamcampus-postgres:/tmp/"
	echo "     docker exec -e SERVICE_DB_PASSWORD=<password> -e POSTGRES_USER=postgres \\"
	echo "       mydreamcampus-postgres bash /tmp/init-databases.sh"
	exit 1
fi

# module directory : target database. catalog is the one pair where the two
# differ — schema `course_catalog` lives in database `catalog`.
migrate_pairs="auth:auth staff:staff student:student course_catalog:catalog \
enrollment:enrollment attendance:attendance grades:grades meal:meal"

for pair in $migrate_pairs; do
	module="${pair%%:*}"
	db="${pair##*:}"
	dir="/migrations/modules/$module"
	[ -d "$dir" ] || continue
	url="$(svc_url "$db")"
	echo ">> goose up — $module → db:$db"
	goose -dir "$dir" -table "goose_db_version_$module" postgres "$url" up
done

echo ">> goose up — notification → db:notification"
goose -dir /migrations/notification -table goose_db_version_notification postgres "$(svc_url notification)" up

echo ">> migrations complete"
