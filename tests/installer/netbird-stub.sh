#!/bin/sh
# TEST ONLY stand-in for NetBird 0.82.0. It does not open a tunnel.
set -eu
cmd="${1:-}"
case "$cmd" in
version)
    printf '%s\n' '0.82.0'
    ;;
status)
    printf '%s\n' 'Management: Connected'
    ;;
service)
    if [ "${2:-}" = "run" ]; then
        exec sleep infinity
    fi
    ;;
up)
    ;;
*)
    ;;
esac
