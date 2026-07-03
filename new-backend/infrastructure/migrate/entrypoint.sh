#!/bin/sh
# Apply goose migrations for every module. Module order matches the monolith
# Makefile (migrate-up-all) so cross-module FKs resolve — e.g. student before
# enrollment. Each module keeps its own goose version table in the shared DB.
set -e

: "${DB_URL:?DB_URL is required}"

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
