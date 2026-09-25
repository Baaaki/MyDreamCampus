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

# lock_until <deadline> keeps the write lock for the time the edit session
# has left, at least a minute so the timeout check gets to end it cleanly.
lock_until() {
    remaining=$(($(iso_to_epoch "$1") - $(date +%s)))
    [ "$remaining" -ge 60 ] || remaining=60
    set_write_lock "$remaining"
}

begin_edit() {
    update_status "busy" "begin_edit" "" ""
    if run_step restore.sh "" "true"; then
        deadline=$(epoch_to_iso $(($(date +%s) + EDIT_TIMEOUT_MINUTES * 60)))
        lock_until "$deadline"
        update_status "editing" "begin_edit" "" "$deadline"
    else
        remove_write_lock
        update_status "normal" "begin_edit" "$STEP_ERROR" ""
    fi
}

# save keeps editing under the same deadline when the snapshot fails: the
# super admin's work stays, and the session still ends on time.
save() {
    deadline=$(status_field edit_deadline)
    update_status "busy" "save" "" "$deadline"
    if run_step snapshot.sh "$1"; then
        remove_write_lock
        update_status "normal" "save" "" ""
    else
        lock_until "$deadline"
        update_status "editing" "save" "$STEP_ERROR" "$deadline"
    fi
}

# Minutes since midnight. The leading zero is stripped: ash reads 08 as a bad
# octal number.
minutes_of_day() {
    h=${1%%:*}
    m=${1##*:}
    echo $((${h#0} * 60 + ${m#0}))
}

# The nightly restore runs once a day within an hour of NIGHTLY_RESET_AT, not
# only in its exact minute: a long command or a restart spanning that minute
# must not skip a night, and a server that was down all night must not rewind
# the visitors in the middle of the day.
check_nightly() {
    today=$(date +"%Y-%m-%d")
    [ "$(cat "$LAST_NIGHTLY_FILE" 2>/dev/null || true)" != "$today" ] || return 0
    now=$(minutes_of_day "$(date +%H:%M)")
    start=$(minutes_of_day "$NIGHTLY_RESET_AT")
    { [ "$now" -ge "$start" ] && [ "$now" -lt $((start + 60)) ]; } || return 0

    echo "$today" > "$LAST_NIGHTLY_FILE"
    if [ "$(status_field mode)" = "editing" ]; then
        echo ">> [demo-ops] Nightly reset skipped: editing mode is currently active."
        return 0
    fi
    echo ">> [demo-ops] Executing nightly baseline restore..."
    restore_to "" "nightly_restore"
}

# An edit session whose deadline cannot be read counts as expired: it must
# never outlive its lock and hold the nightly restore off.
check_edit_timeout() {
    [ "$(status_field mode)" = "editing" ] || return 0
    deadline=$(status_field edit_deadline)
    [ "$(date +%s)" -ge "$(iso_to_epoch "$deadline")" ] || return 0
    echo ">> [demo-ops] Edit deadline expired (${deadline:-none}). Canceling edit automatically..."
    restore_to "" "timeout_cancel_edit"
}

# A restart must not end an edit session, so editing keeps its deadline and
# lock. A restart in the middle of an operation cannot tell how far it got,
# and says so instead of reading as success.
prev_mode=$(status_field mode)
prev_deadline=$(status_field edit_deadline)
prev_error=$(status_field last_error)

if ! ensure_baseline; then
    echo "!! [demo-ops] Initial baseline snapshot failed; retrying every 5 minutes"
    prev_error=$(status_field last_error)
fi

case "$prev_mode" in
    editing)
        lock_until "$prev_deadline"
        update_status "editing" "startup" "$prev_error" "$prev_deadline"
        ;;
    busy)
        remove_write_lock
        update_status "normal" "startup" "Önceki işlem demo-ops yeniden başlarken yarıda kaldı; gerekirse tekrarlayın." ""
        ;;
    *)
        update_status "normal" "startup" "$prev_error" ""
        ;;
esac
echo ">> [demo-ops] Ready and listening for commands on ops:commands..."

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
                begin_edit
                ;;

            save)
                save "$req_by"
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
    check_nightly
    check_edit_timeout
done
