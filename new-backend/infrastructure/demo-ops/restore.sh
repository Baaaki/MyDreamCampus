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
    fail "[restore] No target version specified and no current baseline found." \
        "Geri dönülecek kalıcı durum yok; önce bir kalıcı durum kaydedilmeli."
fi

SRC="$BASELINE_DIR/$target_version"
if [ ! -d "$SRC" ]; then
    fail "[restore] Baseline directory does not exist: $SRC" \
        "Kalıcı durum bulunamadı: $target_version"
fi

# Every dump is checked before anything is touched: a bad file found half-way
# would leave some databases rewound and the rest not.
for db in $ALL_DBS; do
    dump_file="$SRC/$db.dump"
    if [ ! -s "$dump_file" ] || ! pg_restore -l "$dump_file" >/dev/null 2>&1; then
        fail "[restore] Dump file missing or unreadable for $db: $dump_file" \
            "Kalıcı durum $target_version bozuk: $db yedeği okunamıyor. Başka bir sürüme dönün."
    fi
done

echo ">> [restore] Restoring baseline $target_version (keep_lock=$keep_lock)..."

# 1. Acquire write lock during restore
set_write_lock $((EDIT_TIMEOUT_MINUTES * 60))

# 2. Restore each database
for db in $ALL_DBS; do
    dump_file="$SRC/$db.dump"
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
        fail "[restore] Failed to restore database $db" \
            "Geri dönüş $db veritabanında başarısız oldu; veritabanları birbiriyle tutarsız olabilir. Geri dönüşü tekrarlayın."
    fi

    terminate_connections "$db" || echo "!! [restore] Could not end old sessions on $db; its service may answer 500 until they close"

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
    goose -dir "$dir" -table "goose_db_version_$module" postgres "$url" up >/dev/null \
        || fail "[restore] goose up failed for $module" \
            "Geri dönüş sonrası $db migration'ları uygulanamadı."
done

if [ -d "/migrations/notification" ]; then
    goose -dir /migrations/notification -table goose_db_version_notification postgres "$(svc_url notification)" up >/dev/null \
        || fail "[restore] goose up failed for notification" \
            "Geri dönüş sonrası notification migration'ları uygulanamadı."
fi

# A migration may have altered what a service re-prepared since its
# database came back.
for db in $ALL_DBS; do
    terminate_connections "$db" || echo "!! [restore] Could not end old sessions on $db; its service may answer 500 until they close"
done
# pgxpool pings only a connection that sat idle for over a second, so one
# used just before the termination above is still handed out dead for that
# long (57P01, a 500 to whoever gets it). Past the second every dead one is
# pinged and replaced, so the restore is not reported done before then.
sleep 2

# 4. Purge RabbitMQ queues
purge_rabbitmq_queues

# 5. Reset Redis: time machine, token blacklist, rate limits, idempotency
# and attendance buffers go; the ops:* keys stay.
echo ">> [restore] Resetting Redis (keeping ops:* keys)..."
reset_redis >/dev/null

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
