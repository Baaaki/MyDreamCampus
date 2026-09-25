#!/bin/sh
set -e

: "${PG_HOST:=postgres}"
: "${POSTGRES_USER:=postgres}"
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"
: "${SERVICE_DB_PASSWORD:?SERVICE_DB_PASSWORD is required}"
: "${REDIS_ADDR:=redis:6379}"
: "${REDIS_PASSWORD:?REDIS_PASSWORD is required}"
: "${RABBITMQ_HOST:=rabbitmq}"
: "${RABBITMQ_USER:=rabbitmq}"
: "${RABBITMQ_PASSWORD:?RABBITMQ_PASSWORD is required}"
: "${RABBITMQ_PORT:=15672}"
: "${BASELINE_DIR:=/baselines}"
: "${BASELINE_KEEP:=7}"
: "${EDIT_TIMEOUT_MINUTES:=120}"
: "${NIGHTLY_RESET_AT:=04:00}"
: "${DEMO_MODE:=false}"
: "${BASELINE_OFFSITE_CMD:=}"

REDIS_HOST="${REDIS_ADDR%%:*}"
REDIS_PORT="${REDIS_ADDR##*:}"
[ "$REDIS_PORT" = "$REDIS_HOST" ] && REDIS_PORT=6379

ALL_DBS="auth staff student catalog enrollment attendance grades meal payment notification"

MIGRATE_PAIRS="auth:auth staff:staff student:student course_catalog:catalog \
enrollment:enrollment attendance:attendance grades:grades meal:meal payment:payment"

# snapshot.sh and restore.sh leave the user-facing reason of a failure here;
# ops.sh puts it on the status panel.
ERROR_FILE="${ERROR_FILE:-/tmp/ops-last-error}"

# fail logs the reason in English, leaves the Turkish one for the panel and
# stops the script.
fail() {
    echo "!! $1" >&2
    printf '%s' "$2" > "$ERROR_FILE"
    exit 1
}

redis_cmd() {
    redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" --no-auth-warning "$@"
}

psql_super() {
    db="$1"
    shift
    PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$PG_HOST" -U "$POSTGRES_USER" -d "$db" -q -t -A "$@"
}

svc_url() {
    db="$1"
    echo "postgres://${db}_svc:${SERVICE_DB_PASSWORD}@${PG_HOST}:5432/${db}?sslmode=disable"
}

set_write_lock() {
    ttl="${1:-$((EDIT_TIMEOUT_MINUTES * 60))}"
    redis_cmd SET ops:write_lock "1" EX "$ttl" >/dev/null
}

remove_write_lock() {
    redis_cmd DEL ops:write_lock >/dev/null
}

get_current_version() {
    if [ -f "$BASELINE_DIR/current" ]; then
        cat "$BASELINE_DIR/current"
    elif [ -L "$BASELINE_DIR/current" ]; then
        readlink "$BASELINE_DIR/current" | xargs basename
    else
        echo ""
    fi
}

get_versions_json() {
    if [ -d "$BASELINE_DIR" ]; then
        versions=$(ls -1 "$BASELINE_DIR" 2>/dev/null | grep -E '^[0-9]{8}-[0-9]{6}$' | sort -r | head -n "$BASELINE_KEEP" || true)
        if [ -n "$versions" ]; then
            echo "$versions" | jq -R -s -c 'split("\n")[:-1]'
        else
            echo "[]"
        fi
    else
        echo "[]"
    fi
}

update_status() {
    mode="$1"
    last_action="$2"
    last_error="$3"
    edit_deadline="$4"

    current=$(get_current_version)
    versions=$(get_versions_json)
    now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # JSON escaping and null handling
    if [ -z "$edit_deadline" ] || [ "$edit_deadline" = "null" ]; then
        deadline_json="null"
    else
        deadline_json="\"$edit_deadline\""
    fi

    if [ -z "$last_error" ] || [ "$last_error" = "null" ]; then
        err_json="null"
    else
        err_json=$(printf '%s' "$last_error" | jq -R .)
    fi

    action_json=$(printf '%s' "$last_action" | jq -R .)
    current_json=$(printf '%s' "$current" | jq -R .)

    status_json=$(cat <<EOF
{
  "mode": "$mode",
  "current": $current_json,
  "versions": $versions,
  "edit_deadline": $deadline_json,
  "last_action": $action_json,
  "last_error": $err_json,
  "updated_at": "$now"
}
EOF
)
    redis_cmd SET ops:status "$status_json" >/dev/null
}

wait_for_services() {
    echo ">> [demo-ops] Waiting for Postgres, Redis, and RabbitMQ..."
    until PGPASSWORD="$POSTGRES_PASSWORD" pg_isready -h "$PG_HOST" -U "$POSTGRES_USER" -d postgres >/dev/null 2>&1; do
        sleep 2
    done

    until redis_cmd PING >/dev/null 2>&1; do
        sleep 2
    done

    until curl -s -u "$RABBITMQ_USER:$RABBITMQ_PASSWORD" "http://$RABBITMQ_HOST:$RABBITMQ_PORT/api/overview" >/dev/null 2>&1; do
        sleep 2
    done
    echo ">> [demo-ops] All infra services are ready."
}

purge_rabbitmq_queues() {
    echo ">> [demo-ops] Purging RabbitMQ queues..."
    queues=$(curl -s -u "$RABBITMQ_USER:$RABBITMQ_PASSWORD" "http://$RABBITMQ_HOST:$RABBITMQ_PORT/api/queues" 2>/dev/null | jq -r '.[].name // empty' || true)
    for q in $queues; do
        echo "   Purging queue: $q"
        curl -s -X DELETE -u "$RABBITMQ_USER:$RABBITMQ_PASSWORD" "http://$RABBITMQ_HOST:$RABBITMQ_PORT/api/queues/%2f/$q/contents" >/dev/null 2>&1 || true
    done
}
