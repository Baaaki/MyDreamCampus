#!/bin/sh
set -e

DIR="$(cd "$(dirname "$0")" && pwd)"
. "$DIR/lib.sh"

echo ">> [demo-ops] Starting demo-ops daemon..."

if [ "$DEMO_MODE" != "true" ]; then
    echo ">> [demo-ops] DEMO_MODE is not 'true'. demo-ops will sleep indefinitely."
    exec sleep infinity
fi

wait_for_services

mkdir -p "$BASELINE_DIR"

# run_step <script> [args...] runs snapshot.sh or restore.sh. On failure
# STEP_ERROR holds the Turkish reason the script left for the panel.
run_step() {
    script="$1"
    shift
    rm -f "$ERROR_FILE"
    STEP_ERROR=""
    if "$DIR/$script" "$@"; then
        return 0
    fi
    STEP_ERROR=$(cat "$ERROR_FILE" 2>/dev/null || true)
    [ -n "$STEP_ERROR" ] || STEP_ERROR="İşlem beklenmedik bir hatayla durdu; ayrıntı demo-ops loglarında."
    echo "!! [demo-ops] $script failed: $STEP_ERROR"
    return 1
}

# restore_to <version|""> <action> returns the system to a baseline and leaves
# it writable. A failed restore may have rewound some databases and not the
# others, so its reason stays on the panel instead of reading as success.
restore_to() {
    update_status "busy" "$2" "" ""
    if run_step restore.sh "$1" "false"; then
        update_status "normal" "$2" "" ""
    else
        remove_write_lock
        update_status "normal" "$2" "$STEP_ERROR" ""
    fi
}

# ensure_baseline takes the first permanent state right after the seed, and
# keeps retrying every 5 minutes if that failed: without a baseline neither
# the nightly restore nor editing can work.
LAST_BASELINE_ATTEMPT=0
ensure_baseline() {
    current=$(get_current_version)
    if [ -n "$current" ] && [ -d "$BASELINE_DIR/$current" ]; then
        return 0
    fi
    now=$(date +%s)
    [ $((now - LAST_BASELINE_ATTEMPT)) -ge 300 ] || return 1
    LAST_BASELINE_ATTEMPT=$now

    echo ">> [demo-ops] No baseline found. Creating one from the current databases..."
    update_status "busy" "initial_snapshot" "" ""
    if run_step snapshot.sh "system-initial"; then
        update_status "normal" "initial_snapshot" "" ""
        return 0
    fi
    update_status "normal" "initial_snapshot" "İlk kalıcı durum alınamadı: $STEP_ERROR" ""
    return 1
}

if ! ensure_baseline; then
    echo "!! [demo-ops] Initial baseline snapshot failed; retrying every 5 minutes"
fi

current_error=$(redis_cmd GET ops:status 2>/dev/null | jq -r '.last_error // empty' 2>/dev/null || true)
update_status "normal" "startup" "$current_error" ""
echo ">> [demo-ops] Ready and listening for commands on ops:commands..."

LAST_NIGHTLY_RUN=""

while true; do
    # BRPOP blocks for up to 30 seconds
    cmd_raw=$(redis_cmd BRPOP ops:commands 30 || true)

    if [ -n "$cmd_raw" ]; then
        # redis-cli BRPOP outputs:
        # 1) "ops:commands"
        # 2) "<json_payload>"
        cmd_json=$(echo "$cmd_raw" | tail -n 1)

        action=$(echo "$cmd_json" | jq -r '.action // empty' 2>/dev/null || true)
        req_by=$(echo "$cmd_json" | jq -r '.requested_by // "superadmin"' 2>/dev/null || true)
        target_ver=$(echo "$cmd_json" | jq -r '.version // empty' 2>/dev/null || true)

        echo ">> [demo-ops] Received command: action=$action requested_by=$req_by version=$target_ver"

        case "$action" in
            begin_edit)
                update_status "busy" "begin_edit" "" ""
                if run_step restore.sh "" "true"; then
                    now_epoch=$(date +%s)
                    deadline_epoch=$((now_epoch + EDIT_TIMEOUT_MINUTES * 60))
                    # ISO 8601 UTC date
                    deadline_iso=$(date -u -d "@$deadline_epoch" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")
                    set_write_lock $((EDIT_TIMEOUT_MINUTES * 60))
                    update_status "editing" "begin_edit" "" "$deadline_iso"
                else
                    remove_write_lock
                    update_status "normal" "begin_edit" "$STEP_ERROR" ""
                fi
                ;;

            save)
                update_status "busy" "save" "" ""
                if run_step snapshot.sh "$req_by"; then
                    remove_write_lock
                    update_status "normal" "save" "" ""
                else
                    # Keep editing mode so user doesn't lose modifications on save error
                    set_write_lock $((EDIT_TIMEOUT_MINUTES * 60))
                    update_status "editing" "save" "$STEP_ERROR" ""
                fi
                ;;

            cancel_edit)
                restore_to "" "cancel_edit"
                ;;

            restore_now)
                restore_to "" "restore_now"
                ;;

            restore_version)
                if [ -n "$target_ver" ] && [ -d "$BASELINE_DIR/$target_ver" ]; then
                    restore_to "$target_ver" "restore_version"
                else
                    update_status "normal" "restore_version" "Sürüm bulunamadı: $target_ver" ""
                fi
                ;;

            *)
                echo "!! [demo-ops] Unknown action: $action"
                ;;
        esac
    fi

    ensure_baseline || true

    # 1. Scheduled Nightly Reset (NIGHTLY_RESET_AT)
    now_hm=$(date +"%H:%M")
    today=$(date +"%Y-%m-%d")

    if [ "$now_hm" = "$NIGHTLY_RESET_AT" ] && [ "$today" != "$LAST_NIGHTLY_RUN" ]; then
        status_raw=$(redis_cmd GET ops:status || true)
        current_mode=$(echo "$status_raw" | jq -r '.mode // "normal"' 2>/dev/null || echo "normal")

        if [ "$current_mode" = "editing" ]; then
            echo ">> [demo-ops] Nightly reset skipped: editing mode is currently active."
        else
            echo ">> [demo-ops] Executing nightly baseline restore at $now_hm..."
            restore_to "" "nightly_restore"
        fi
        LAST_NIGHTLY_RUN="$today"
    fi

    # 2. Check editing mode timeout
    status_raw=$(redis_cmd GET ops:status || true)
    mode=$(echo "$status_raw" | jq -r '.mode // empty' 2>/dev/null || true)
    deadline=$(echo "$status_raw" | jq -r '.edit_deadline // empty' 2>/dev/null || true)

    if [ "$mode" = "editing" ] && [ -n "$deadline" ] && [ "$deadline" != "null" ]; then
        now_epoch=$(date +%s)
        deadline_epoch=$(date -d "$deadline" +%s 2>/dev/null || echo 0)
        if [ "$deadline_epoch" -gt 0 ] && [ "$now_epoch" -ge "$deadline_epoch" ]; then
            echo ">> [demo-ops] Edit deadline expired ($deadline). Canceling edit automatically..."
            restore_to "" "timeout_cancel_edit"
        fi
    fi
done
