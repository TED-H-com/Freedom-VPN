#!/bin/sh
# TEST ONLY service manager inside the node container.
#
# The entrypoint supervisor is the only process that starts or stops
# fvpn-agent. Public stop creates /var/lib/vpn-node/e2e/pause-agent and waits
# until that supervisor has reaped the agent. Public start records the release,
# clears the marker, and waits until the supervisor owns exactly one matching
# process. Both are idempotent and print the PID the supervisor owns (0 when
# stopped). The updater appends the verified release directory to start.
# Repeated start of the same release must not restart (D-053).
#
# pause-agent is the only stop path. Harnesses may keep toggling that file
# directly. Removing it does not wait; the next supervisor poll (0.2s) applies
# it. Call start when the caller needs the supervisor's PID.
#
# --supervisor is internal. The entrypoint loop calls it every poll. It must
# not call the public wait path.
set -eu
base=/var/lib/vpn-node/e2e
pause=$base/pause-agent
poll=0.2

agent_state() {
    _state_rest=${1#*)}
    _state_rest=${_state_rest# }
    printf '%s\n' "${_state_rest%% *}"
}

each_agent() {
    _each_mode=$1
    _each_n=0
    _each_one=
    for _each_d in /proc/[0-9]*; do
        [ -d "$_each_d" ] || continue
        [ -r "$_each_d/cmdline" ] || continue
        [ -r "$_each_d/stat" ] || continue
        _each_p=${_each_d#/proc/}
        _each_stat=$(cat "$_each_d/stat") || continue
        [ "$(agent_state "$_each_stat")" = Z ] && continue
        _each_cmd=$(tr '\0' ' ' < "$_each_d/cmdline") || continue
        case "$_each_cmd" in
            *"/fvpn-agent run"|*"/fvpn-agent run "*) ;;
            *) continue ;;
        esac
        _each_n=$((_each_n + 1))
        _each_one=$_each_p
        if [ "$_each_mode" = kill ]; then
            kill "$_each_p" 2>/dev/null || true
        elif [ "$_each_mode" = kill9 ]; then
            kill -KILL "$_each_p" 2>/dev/null || true
        fi
    done
    if [ "$_each_mode" = count ]; then
        printf '%s\n' "$_each_n"
    elif [ "$_each_mode" = one ]; then
        [ "$_each_n" = 1 ] || return 1
        printf '%s\n' "$_each_one"
    fi
}

count_agents() {
    each_agent count
}

reap_agents() {
    each_agent kill
    _reap_i=0
    while [ "$(count_agents)" -ne 0 ] && [ "$_reap_i" -lt 25 ]; do
        sleep "$poll"
        _reap_i=$((_reap_i + 1))
    done
    if [ "$(count_agents)" -ne 0 ]; then
        each_agent kill9
        sleep "$poll"
    fi
    rm -f "$base/agent.pid"
}

# matching_owner RELEASE prints the pid when agent.pid, the exe, and the
# selected release all name the only live fvpn-agent run process.
matching_owner() {
    _match_release=$1
    _match_one=$(each_agent one) || return 1
    [ -s "$base/agent.pid" ] || return 1
    [ "$(cat "$base/agent.pid")" = "$_match_one" ] || return 1
    _match_exe=$(readlink "/proc/$_match_one/exe" 2>/dev/null || true)
    case "$_match_exe" in
        "$_match_release/bin/fvpn-agent"|"$_match_release/bin/fvpn-agent (deleted)") ;;
        *) return 1 ;;
    esac
    [ -s "$base/agent-selected" ] || return 1
    [ "$(cat "$base/agent-selected")" = "$_match_release" ] || return 1
    printf '%s\n' "$_match_one"
}

release_config() {
    case $1 in
        /usr/local) printf '%s\n' /etc/vpn-node/agent-run.json ;;
        /opt/vpn-node/releases/*) printf '%s\n' "$base/agent-${1##*/}.json" ;;
        *) return 2 ;;
    esac
}

supervise_stop() {
    reap_agents
}

supervise_start() {
    _sup_release=$1
    _sup_config=$(release_config "$_sup_release") || return 2
    if [ -f "$pause" ]; then
        reap_agents
        return 0
    fi
    if ! [ -x "$_sup_release/bin/fvpn-agent" ] || ! [ -s "$_sup_config" ]; then
        return 1
    fi
    # D-053: an already-correct single process is not restarted.
    if matching_owner "$_sup_release" >/dev/null; then
        return 0
    fi
    reap_agents
    if [ -f "$pause" ]; then
        return 0
    fi
    printf '%s\n' "$_sup_release" > "$base/agent-selected"
    nohup "$_sup_release/bin/fvpn-agent" run -config "$_sup_config" > "$base/agent.log" 2>&1 < /dev/null &
    printf '%s\n' "$!" > "$base/agent.pid"
}

public_stop() {
    mkdir -p "$base"
    : > "$pause"
    _pub_prev=1
    _pub_i=0
    while [ "$_pub_i" -lt 100 ]; do
        if [ "$(count_agents)" -eq 0 ] && { ! [ -s "$base/agent.pid" ] || ! kill -0 "$(cat "$base/agent.pid")" 2>/dev/null; }; then
            if [ "$_pub_prev" -eq 0 ]; then
                printf '%s\n' 0
                return 0
            fi
            _pub_prev=0
        else
            _pub_prev=1
        fi
        sleep "$poll"
        _pub_i=$((_pub_i + 1))
    done
    return 1
}

public_start() {
    _pub_release=${1:-}
    [ -n "$_pub_release" ] || return 2
    _pub_config=$(release_config "$_pub_release") || return 2
    [ -x "$_pub_release/bin/fvpn-agent" ] || return 2
    [ -s "$_pub_config" ] || return 2
    mkdir -p "$base"
    # Record the release before the marker disappears so the supervisor cannot
    # respawn the previous binary in the gap.
    printf '%s\n' "$_pub_release" > "$base/agent-selected"
    rm -f "$pause"
    _pub_stable=
    _pub_i=0
    while [ "$_pub_i" -lt 100 ]; do
        if _pub_pid=$(matching_owner "$_pub_release"); then
            if [ "$_pub_pid" = "$_pub_stable" ]; then
                printf '%s\n' "$_pub_pid"
                return 0
            fi
            _pub_stable=$_pub_pid
        else
            _pub_stable=
        fi
        sleep "$poll"
        _pub_i=$((_pub_i + 1))
    done
    return 1
}

if [ "${1:-}" = "--supervisor" ]; then
    shift
    case ${1:-} in
        stop) supervise_stop ;;
        start) supervise_start "${2:-}" ;;
        *) exit 2 ;;
    esac
    exit 0
fi

case ${1:-} in
    stop) public_stop ;;
    start) public_start "${2:-}" ;;
    *) exit 2 ;;
esac
