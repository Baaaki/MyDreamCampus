#!/bin/sh
set -e

DIR="$(cd "$(dirname "$0")" && pwd)"
. "$DIR/lib.sh"

target_version="$1"
keep_lock="${2:-false}"

if [ -z "$target_version" ]; then
    target_version=$(get_current_version)
fi

if [ -z "$target_version" ]; then
    echo "!! [restore] No target version specified and no current baseline found."
    exit 1
fi

SRC="$BASELINE_DIR/$target_version"
if [ ! -d "$SRC" ]; then
    echo "!! [restore] Baseline directory does not exist: $SRC"
    exit 1
fi

echo ">> [restore] Restoring baseline $target_version (keep_lock=$keep_lock)..."

# 1. Acquire write lock during restore
set_write_lock $((EDIT_TIMEOUT_MINUTES * 60))

# 2. Restore each database
for db in $ALL_DBS; do
    dump_file="$SRC/$db.dump"
    if [ ! -f "$dump_file" ]; then
        echo "!! [restore] Dump file not found for $db: $dump_file"
        exit 1
    fi

    echo "   Restoring database: $db..."
    restore_success=false

    for attempt in 1 2; do
        if PGPASSWORD="$POSTGRES_PASSWORD" PGOPTIONS="-c lock_timeout=5000" pg_restore \
            -h "$PG_HOST" -U "$POSTGRES_USER" -d "$db" --clean --if-exists --single-transaction "$dump_file"; then
            restore_success=true
            break
        else
            echo "   !! Attempt $attempt failed for $db, retrying once..."
            sleep 2
        fi
    done

    if [ "$restore_success" != "true" ]; then
        echo "!! [restore] Failed to restore database $db"
        exit 1
    fi

    # Ensure schema ownership belongs to service role
    schema="$db"
    [ "$db" = "catalog" ] && schema="course_catalog"
    svc_role="${db}_svc"
    psql_super "$db" -c "ALTER SCHEMA $schema OWNER TO $svc_role;" >/dev/null 2>&1 || true
done

# 3. Run goose up for any newer migrations deployed between baseline and now
echo ">> [restore] Applying migrations on restored databases..."
for pair in $MIGRATE_PAIRS; do
    module="${pair%%:*}"
    db="${pair##*:}"
    dir="/migrations/modules/$module"
    [ -d "$dir" ] || continue
    url="$(svc_url "$db")"
    goose -dir "$dir" -table "goose_db_version_$module" postgres "$url" up >/dev/null
done

if [ -d "/migrations/notification" ]; then
    goose -dir /migrations/notification -table goose_db_version_notification postgres "$(svc_url notification)" up >/dev/null
fi

# 4. Purge RabbitMQ queues
purge_rabbitmq_queues

# 5. Flush Redis DB 0
echo ">> [restore] Flushing Redis DB 0..."
redis_cmd FLUSHDB >/dev/null

# 6. Rewrite current baseline pointer if restoring a specific past version
echo "$target_version" > "$BASELINE_DIR/current"
ln -sfn "$target_version" "$BASELINE_DIR/current_link" || true

# 7. Lock handling: if keep_lock is true, ensure lock is set; otherwise remove it
if [ "$keep_lock" = "true" ]; then
    set_write_lock $((EDIT_TIMEOUT_MINUTES * 60))
    echo ">> [restore] Write lock kept as requested."
else
    remove_write_lock
    echo ">> [restore] Write lock released."
fi

echo ">> [restore] Baseline $target_version restored successfully."
