#!/bin/sh
# Apply goose migrations for every module. Module order matches the monolith
# Makefile (migrate-up-all) so cross-module FKs resolve — e.g. student before
# enrollment. Each module keeps its own goose version table in the shared DB.
set -e

: "${DB_URL:?DB_URL is required}"

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

wait_for_db "monolith database" "$DB_URL"
[ -n "$NOTIF_DB_URL" ] && wait_for_db "notification database" "$NOTIF_DB_URL"

for module in auth staff student course_catalog enrollment attendance grades meal payment; do
	dir="/migrations/modules/$module"
	[ -d "$dir" ] || continue
	echo ">> goose up (monolith) — $module"
	goose -dir "$dir" -table "goose_db_version_$module" postgres "$DB_URL" up
done

if [ -n "$NOTIF_DB_URL" ]; then
	echo ">> goose up (notification)"
	goose -dir /migrations/notification -table goose_db_version_notification postgres "$NOTIF_DB_URL" up
fi

echo ">> migrations complete"
