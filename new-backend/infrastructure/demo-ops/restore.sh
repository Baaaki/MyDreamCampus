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

# The super admin's password, first-login flag, token version and sessions
# are not demo data: rewound with the rest of auth, a changed password would
# go back to the baseline's (ADMIN_INITIAL_PASSWORD on the first one) and the
# super admin would be logged out every hour. They travel as JSON rows so the
# argon2 hashes never pass through the shell, and are read back by column
# name once the migrations below have run.
KEEP_DIR=$(mktemp -d)
trap 'rm -rf "$KEEP_DIR"' EXIT
psql_super auth -v ON_ERROR_STOP=1 <<EOF \
    || fail "[restore] Could not keep the super admin's credentials aside" \
        "Süper adminin şifresi saklanamadı; geri dönüş başlamadı, veri değişmedi."
\copy (SELECT row_to_json(u) FROM (SELECT id, password_hash, force_password_change, token_version FROM auth.users WHERE is_superadmin) u) TO '$KEEP_DIR/admins'
\copy (SELECT row_to_json(s) FROM auth.sessions s WHERE s.user_id IN (SELECT id FROM auth.users WHERE is_superadmin)) TO '$KEEP_DIR/sessions'
EOF

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

# Every database is rewound by now, so a failure here must not stop the
# script short of the Redis reset: the transaction leaves the baseline's
# password in place and the log says so.
if [ -s "$KEEP_DIR/admins" ]; then
    echo ">> [restore] Putting back the super admin's credentials and sessions..."
    psql_super auth -v ON_ERROR_STOP=1 <<EOF \
        || echo "!! [restore] Could not put back the super admin's credentials; the baseline's password is in effect"
BEGIN;
CREATE TEMP TABLE kept_admins (j jsonb);
CREATE TEMP TABLE kept_sessions (j jsonb);
\copy kept_admins FROM '$KEEP_DIR/admins'
\copy kept_sessions FROM '$KEEP_DIR/sessions'
UPDATE auth.users u
   SET password_hash = k.password_hash,
       force_password_change = k.force_password_change,
       token_version = k.token_version
  FROM kept_admins ka, jsonb_populate_record(NULL::auth.users, ka.j) k
 WHERE u.id = k.id;
DELETE FROM auth.sessions s
 USING kept_admins ka, jsonb_populate_record(NULL::auth.users, ka.j) k
 WHERE s.user_id = k.id;
INSERT INTO auth.sessions
SELECT k.*
  FROM kept_sessions ks, jsonb_populate_record(NULL::auth.sessions, ks.j) k
 WHERE k.user_id IN (SELECT id FROM auth.users);
COMMIT;
EOF
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
# Still under the write lock, so no visitor write can land between the
# rewind and the stamp going away.
clear_changes

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
