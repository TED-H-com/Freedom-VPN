#!/bin/sh
# TEST ONLY service manager inside the node container. The updater appends the
# verified release directory to start. Repeated start must not restart D-053.
set -eu
base=/var/lib/vpn-node/e2e
alive() {
    [ -s "$base/agent.pid" ] || return 1
    pid=$(cat "$base/agent.pid")
    kill -0 "$pid" 2>/dev/null || return 1
    read -r stat_pid stat_name stat_state stat_rest < "/proc/$pid/stat" || return 1
    [ "$stat_state" != Z ]
}
stop() {
    if alive; then
        kill "$pid"
        count=0
        while alive && [ "$count" -lt 15 ]; do
            sleep 1
            count=$((count + 1))
        done
        if alive; then kill -KILL "$pid"; fi
    fi
    rm -f "$base/agent.pid"
}
case "$1" in
    stop) stop ;;
    start)
        release=$2
        case "$release" in
            /usr/local) config=/etc/vpn-node/agent-run.json ;;
            /opt/vpn-node/releases/*) config="$base/agent-${release##*/}.json" ;;
            *) exit 2 ;;
        esac
        [ -x "$release/bin/fvpn-agent" ] && [ -s "$config" ]
        if alive && [ "$(cat "$base/agent-selected")" = "$release" ]; then exit 0; fi
        stop
        printf '%s\n' "$release" > "$base/agent-selected"
        nohup "$release/bin/fvpn-agent" run -config "$config" > "$base/agent.log" 2>&1 < /dev/null &
        echo "$!" > "$base/agent.pid"
        ;;
    *) exit 2 ;;
esac
