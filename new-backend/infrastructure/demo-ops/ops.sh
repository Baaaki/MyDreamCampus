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

# Check if initial baseline exists
if [ ! -f "$BASELINE_DIR/current" ] || [ ! -d "$BASELINE_DIR/$(cat "$BASELINE_DIR/current" 2>/dev/null)" ]; then
    echo ">> [demo-ops] No initial baseline found. Creating first baseline from seeded databases..."
    "$DIR/snapshot.sh" "system-initial" || echo "!! [demo-ops] Initial baseline snapshot failed"
fi

update_status "normal" "startup" "" ""
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
                if "$DIR/restore.sh" "" "true"; then
                    now_epoch=$(date +%s)
                    deadline_epoch=$((now_epoch + EDIT_TIMEOUT_MINUTES * 60))
                    # ISO 8601 UTC date
                    deadline_iso=$(date -u -d "@$deadline_epoch" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")
                    set_write_lock $((EDIT_TIMEOUT_MINUTES * 60))
                    update_status "editing" "begin_edit" "" "$deadline_iso"
                else
                    remove_write_lock
                    update_status "normal" "begin_edit" "Failed to restore baseline for editing" ""
                fi
                ;;

            save)
                update_status "busy" "save" "" ""
                if "$DIR/snapshot.sh" "$req_by"; then
                    remove_write_lock
                    update_status "normal" "save" "" ""
                else
                    # Keep editing mode so user doesn't lose modifications on save error
                    set_write_lock $((EDIT_TIMEOUT_MINUTES * 60))
                    update_status "editing" "save" "Failed to save baseline snapshot" ""
                fi
                ;;

            cancel_edit)
                update_status "busy" "cancel_edit" "" ""
                "$DIR/restore.sh" "" "false" || true
                remove_write_lock
                update_status "normal" "cancel_edit" "" ""
                ;;

            restore_now)
                update_status "busy" "restore_now" "" ""
                "$DIR/restore.sh" "" "false" || true
                remove_write_lock
                update_status "normal" "restore_now" "" ""
                ;;

            restore_version)
                update_status "busy" "restore_version" "" ""
                if [ -n "$target_ver" ] && [ -d "$BASELINE_DIR/$target_ver" ]; then
                    "$DIR/restore.sh" "$target_ver" "false" || true
                    remove_write_lock
                    update_status "normal" "restore_version" "" ""
                else
                    update_status "normal" "restore_version" "Version not found: $target_ver" ""
                fi
                ;;

            *)
                echo "!! [demo-ops] Unknown action: $action"
                ;;
        esac
    fi

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
            update_status "busy" "nightly_restore" "" ""
            "$DIR/restore.sh" "" "false" || true
            remove_write_lock
            update_status "normal" "nightly_restore" "" ""
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
            update_status "busy" "timeout_cancel_edit" "" ""
            "$DIR/restore.sh" "" "false" || true
            remove_write_lock
            update_status "normal" "timeout_cancel_edit" "" ""
        fi
    fi
done
