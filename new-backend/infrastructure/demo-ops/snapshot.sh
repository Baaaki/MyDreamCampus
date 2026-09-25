#!/bin/sh
set -e

DIR="$(cd "$(dirname "$0")" && pwd)"
. "$DIR/lib.sh"

requested_by="${1:-superadmin}"

echo ">> [snapshot] Starting snapshot requested by: $requested_by"

# 1. Wait for drains (outbox and rabbitmq) up to 60s
echo ">> [snapshot] Waiting for outbox and RabbitMQ queues to drain (max 60s)..."
drained=false
for i in $(seq 1 60); do
    pending_total=0

    # Check each service's outbox table
    for pair in "auth:auth.outbox_events" "staff:staff.outbox_events" "student:student.outbox_events" \
                "catalog:course_catalog.outbox_events" "enrollment:enrollment.outbox_events" \
                "attendance:attendance.outbox_events" "grades:grades.outbox_events" \
                "meal:meal.outbox_events" "payment:payment.outbox_events"; do
        db="${pair%%:*}"
        tbl="${pair##*:}"
        # An outbox that cannot be read is not known to be empty.
        cnt=$(psql_super "$db" -c "SELECT count(*) FROM $tbl WHERE status = 'pending';" 2>/dev/null || echo "1")
        cnt=$(echo "$cnt" | tr -d '[:space:]')
        [ -z "$cnt" ] && cnt=1
        pending_total=$((pending_total + cnt))
    done

    # Dead-letter queues hold what consumers gave up on and nothing drains
    # them: counting them would fail every save until someone empties them.
    rmq_messages=$(curl -s -u "$RABBITMQ_USER:$RABBITMQ_PASSWORD" "http://$RABBITMQ_HOST:$RABBITMQ_PORT/api/queues" 2>/dev/null | jq '[.[] | select(.name | endswith(".dlq") | not) | .messages // 0] | add // 0' || echo "0")
    rmq_messages=$(echo "$rmq_messages" | tr -d '[:space:]')
    [ -z "$rmq_messages" ] && rmq_messages=0

    if [ "$pending_total" -eq 0 ] && [ "$rmq_messages" -eq 0 ]; then
        drained=true
        echo ">> [snapshot] Queues and outboxes drained successfully after ${i}s."
        break
    fi

    sleep 1
done

if [ "$drained" != "true" ]; then
    fail "[snapshot] Timeout waiting for outbox ($pending_total pending) or rabbitmq ($rmq_messages messages) to drain" \
        "Kaydedilemedi: bekleyen olaylar 60 sn içinde işlenmedi (outbox: $pending_total, kuyruk: $rmq_messages). Birkaç dakika sonra tekrar deneyin."
fi

# 2. Dump into a hidden work directory; it gets the version name only once
# every file is written, so a half-written version is never offered for restore.
VERSION=$(date +"%Y%m%d-%H%M%S")
DEST="$BASELINE_DIR/$VERSION"
WORK="$BASELINE_DIR/.tmp-$VERSION"
trap 'rm -rf "$WORK"' EXIT
# Leftovers of a snapshot the container was killed in the middle of.
rm -rf "$BASELINE_DIR"/.tmp-*
mkdir -p "$WORK"

echo ">> [snapshot] Dumping databases to $WORK..."

for db in $ALL_DBS; do
    echo "   Dumping $db..."
    PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -h "$PG_HOST" -U "$POSTGRES_USER" -Fc "$db" > "$WORK/$db.dump" \
        || fail "[snapshot] pg_dump failed for $db" "Kaydedilemedi: $db veritabanının yedeği alınamadı."
done

# Collect goose versions for manifest
manifest_versions="{"
first=true
for pair in $MIGRATE_PAIRS "notification:notification"; do
    module="${pair%%:*}"
    db="${pair##*:}"
    table="goose_db_version_$module"
    v=$(psql_super "$db" -c "SELECT version_id FROM $table ORDER BY id DESC LIMIT 1;" 2>/dev/null || echo "unknown")
    v=$(echo "$v" | tr -d '[:space:]')
    [ -z "$v" ] && v="unknown"
    if [ "$first" = true ]; then
        first=false
    else
        manifest_versions="$manifest_versions,"
    fi
    manifest_versions="$manifest_versions\"$db\":\"$v\""
done
manifest_versions="$manifest_versions}"

now_iso=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
cat <<EOF > "$WORK/manifest.json"
{
  "version": "$VERSION",
  "created_at": "$now_iso",
  "requested_by": "$requested_by",
  "goose_versions": $manifest_versions
}
EOF

mv "$WORK" "$DEST"

# 3. Update current pointer
echo "$VERSION" > "$BASELINE_DIR/current"
ln -sfn "$VERSION" "$BASELINE_DIR/current_link" || true

# 4. Prune old baselines keeping BASELINE_KEEP
echo ">> [snapshot] Pruning old baselines (keeping $BASELINE_KEEP)..."
dirs_to_remove=$(ls -1 "$BASELINE_DIR" | grep -E '^[0-9]{8}-[0-9]{6}$' | sort -r | tail -n +"$((BASELINE_KEEP + 1))" || true)
for d in $dirs_to_remove; do
    if [ -d "$BASELINE_DIR/$d" ]; then
        echo "   Removing old baseline: $d"
        rm -rf "$BASELINE_DIR/$d"
    fi
done

# 5. Offsite backup if command configured
if [ -n "$BASELINE_OFFSITE_CMD" ]; then
    echo ">> [snapshot] Running offsite backup command..."
    eval "$BASELINE_OFFSITE_CMD" || echo "!! [snapshot] Offsite backup command failed"
fi

echo ">> [snapshot] Snapshot completed successfully: $VERSION"
