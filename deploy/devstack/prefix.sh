#!/bin/sh
# Exact dev-stack prefix ownership.
#
# An object belongs to SHORT when its name is exactly fvpn-SHORT, or
# fvpn-SHORT-<known token> with further -suffixes. The token is a compose
# service, a named volume, or a runtime object this lab creates (member peers,
# node2, node-clone, restore, state-reset, probes under node-).
#
# Prefix e2e therefore owns fvpn-e2e-node and fvpn-e2e-member-exports-u01,
# and does not own fvpn-e2e-exports-node. The following dash is part of the
# token boundary, not a prefix of every longer name.
#
# A prefix that itself continues with a known token (for example e2e-node)
# still collides with that token. e2e-exports does not: exports is not a token.

prefix_valid() {
    _valid_short=$1
    [ -n "$_valid_short" ] || return 1
    [ "${#_valid_short}" -le 21 ] || return 1
    case $_valid_short in
        *[!a-z0-9-]*|*-|[!a-z]*) return 1 ;;
    esac
}

# prefix_owned SHORT NAME
# NAME may be a container, network, volume, or repository:tag.
# The tag is ignored for the match and is not written back onto the caller's
# variables: a filter must print repository:tag, not the :latest name.
prefix_owned() {
    _own_short=$1
    _own_name=$2
    prefix_valid "$_own_short" || return 1
    _own_name=${_own_name%%:*}
    _own_tokens='pgbackrest-repo|postgres|node-netns|node-clone|installer-node|node2|state-reset|netbird|gateway|coredns|adguard|control|worker|payment|origin|member|restore|idp|node|pg|nb'
    # shellcheck disable=SC2086
    printf '%s\n' "$_own_name" | grep -Eq "^(fvpn-${_own_short}-(${_own_tokens})(-[^:/]+)*|fvpn-${_own_short})$"
}

# prefix_filter SHORT reads names on stdin and prints those SHORT owns.
# A repository:tag is printed unchanged.
prefix_filter() {
    _filter_short=$1
    while IFS= read -r _filter_name; do
        [ -n "$_filter_name" ] || continue
        if prefix_owned "$_filter_short" "$_filter_name"; then
            printf '%s\n' "$_filter_name"
        fi
    done
}

if [ "${0##*/}" = "prefix.sh" ]; then
    case ${1:-} in
        owned) prefix_owned "${2:-}" "${3:-}" ;;
        filter) prefix_filter "${2:-}" ;;
    esac
fi
