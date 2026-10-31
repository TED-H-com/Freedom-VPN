#!/bin/sh
# TEST ONLY: daemon, status and version use the pinned binary. `up` checks
# the daemon socket without contacting a management server or opening a tunnel.
set -eu
if [ "${1:-}" = up ]; then
    shift
    addr=
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --daemon-addr) addr="$2"; shift ;;
        esac
        shift
    done
    test "$addr" = unix:///run/fvpn-netbird/netbird.sock
    exec /usr/lib/fvpn-tests/netbird status --daemon-addr "$addr"
fi
exec /usr/lib/fvpn-tests/netbird "$@"
