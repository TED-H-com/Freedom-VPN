#!/bin/sh
# Remove every object this prefix created. Another prefix is left alone.
# Shared neutral build-cache images and upstream tags are left alone.
set -eu
cd "$(dirname "$0")/../.."
ROOT=$(pwd)
SHORT=${FVPN_E2E_PREFIX:-e2e}
PREFIX=fvpn-$SHORT
STATE=${FVPN_E2E_STATE:-$ROOT/.agent/e2e-state-$SHORT}
export FVPN_E2E_PREFIX=$SHORT
export FVPN_E2E_STATE=$STATE
mkdir -p "$STATE"

# Load the slot this state was prepared with, when it exists, so compose
# names the same networks it created. Defaults cover a stack that never started.
if [ -f "$STATE/render/compose.env" ]; then
    set -a
    # shellcheck disable=SC1091
    . "$STATE/render/compose.env"
    set +a
fi

docker compose -p "$PREFIX" -f "$ROOT/deploy/devstack/compose.yaml" down -v --remove-orphans --timeout 20 || true

ids=$(docker ps -aq --filter "name=^${PREFIX}-" || true)
if [ -n "$ids" ]; then
    # shellcheck disable=SC2086
    docker rm -f $ids >/dev/null
fi
nets=$(docker network ls --format '{{.Name}}' | grep -E "^${PREFIX}(-|$)" || true)
if [ -n "$nets" ]; then
    # shellcheck disable=SC2086
    docker network rm $nets >/dev/null || true
fi
vols=$(docker volume ls --format '{{.Name}}' | grep -E "^${PREFIX}-" || true)
if [ -n "$vols" ]; then
    # shellcheck disable=SC2086
    docker volume rm $vols >/dev/null || true
fi
images=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E "^${PREFIX}-" || true)
if [ -n "$images" ]; then
    # shellcheck disable=SC2086
    docker rmi $images >/dev/null || true
fi

left=$(
    {
        docker ps -a --format '{{.Names}}' | grep -E "^${PREFIX}-" || true
        docker network ls --format '{{.Name}}' | grep -E "^${PREFIX}(-|$)" || true
        docker volume ls --format '{{.Name}}' | grep -E "^${PREFIX}-" || true
    } | grep . || true
)
if [ -n "$left" ]; then
    echo "prefix objects remain:" >&2
    echo "$left" >&2
    exit 1
fi
echo "e2e stack is down"
