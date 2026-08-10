#!/bin/bash
# One database + one role per service, all on the shared Postgres instance.
#
# Runs from the Postgres entrypoint (/docker-entrypoint-initdb.d), which fires
# ONLY on an empty data volume. An existing volume skips this file entirely —
# provision it by hand, see the command migrate/entrypoint.sh prints on failure.
#
# A .sh and not a .sql because the entrypoint passes environment variables to
# shell scripts only, and the role passwords come from the environment.
#
# Each database carries a schema of the same name as the service — the `auth`
# database holds an `auth` schema — so every migration .sql and all
# sqlc-generated code addresses its tables exactly as it always did.
set -e

: "${POSTGRES_USER:=postgres}"

# One shared password for every service role. Isolation is not the password's
# job but the database's: auth_svc cannot open the catalog database even
# knowing it, because CONNECT is granted per database.
SVC_PASSWORD="${SERVICE_DB_PASSWORD:?SERVICE_DB_PASSWORD is required}"

create_service_db() {
	db="$1"
	schema="$2"
	user="${1}_svc"

	# :'pw' lets psql quote the literal, so a password containing a quote
	# cannot break out of the statement.
	psql -v ON_ERROR_STOP=1 -v pw="$SVC_PASSWORD" -U "$POSTGRES_USER" -d postgres <<-EOSQL
		CREATE DATABASE "$db";
		CREATE USER "$user" WITH PASSWORD :'pw';
		REVOKE CONNECT ON DATABASE "$db" FROM PUBLIC;
		GRANT CONNECT, TEMPORARY ON DATABASE "$db" TO "$user";
	EOSQL

	# Extensions need a superuser, hence this block runs as $POSTGRES_USER.
	# AUTHORIZATION hands the service schema to the service role: goose runs as
	# that role, so it also owns every table it creates and no extra GRANT is
	# needed. public is granted too because goose writes its version table
	# there and PG15+ no longer gives PUBLIC the CREATE privilege on it.
	psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$db" <<-EOSQL
		CREATE EXTENSION IF NOT EXISTS "pgcrypto";
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		CREATE SCHEMA IF NOT EXISTS "$schema" AUTHORIZATION "$user";
		GRANT CREATE ON DATABASE "$db" TO "$user";
		GRANT USAGE, CREATE ON SCHEMA public TO "$user";
	EOSQL

	echo ">> provisioned db:$db schema:$schema user:$user"
}

#                  database    schema
create_service_db auth        auth
create_service_db staff       staff
create_service_db student     student
create_service_db catalog     course_catalog
create_service_db enrollment  enrollment
create_service_db attendance  attendance
create_service_db grades      grades
create_service_db meal        meal

# payment keeps no database — it is a stateless mock with no migrations.

# notification is the one service whose tables live in `public` rather than a
# named schema, so its role owns that schema outright instead of a service one.
psql -v ON_ERROR_STOP=1 -v pw="$SVC_PASSWORD" -U "$POSTGRES_USER" -d postgres <<-EOSQL
	CREATE DATABASE "notification";
	CREATE USER "notification_svc" WITH PASSWORD :'pw';
	REVOKE CONNECT ON DATABASE "notification" FROM PUBLIC;
	GRANT CONNECT, TEMPORARY ON DATABASE "notification" TO "notification_svc";
EOSQL

psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d notification <<-EOSQL
	CREATE EXTENSION IF NOT EXISTS "pgcrypto";
	ALTER SCHEMA public OWNER TO "notification_svc";
EOSQL

echo ">> provisioned db:notification schema:public user:notification_svc"
