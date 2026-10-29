#!/bin/sh
# Remove every object this stack created. Upstream image tags are left alone.
set -eu
cd "$(dirname "$0")/../.."
ROOT=$(pwd)
PREFIX=fvpn-w3-devstack-e2e
STATE=${FVPN_E2E_STATE:-$ROOT/.agent/e2e-state}
export FVPN_E2E_STATE=$STATE
mkdir -p "$STATE"

docker compose -p "$PREFIX" -f "$ROOT/deploy/devstack/compose.yaml" down -v --remove-orphans --timeout 20 || true

ids=$(docker ps -aq --filter "name=$PREFIX" || true)
if [ -n "$ids" ]; then
    # shellcheck disable=SC2086
    docker rm -f $ids >/dev/null
fi
nets=$(docker network ls --filter "name=$PREFIX" -q || true)
if [ -n "$nets" ]; then
    # shellcheck disable=SC2086
    docker network rm $nets >/dev/null || true
fi
vols=$(docker volume ls --filter "name=$PREFIX" -q || true)
if [ -n "$vols" ]; then
    # shellcheck disable=SC2086
    docker volume rm $vols >/dev/null || true
fi
images=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep "^$PREFIX-" || true)
if [ -n "$images" ]; then
    # shellcheck disable=SC2086
    docker rmi $images >/dev/null || true
fi

left=$(
    {
        docker ps -a --format '{{.Names}}'
        docker network ls --format '{{.Name}}'
        docker volume ls --format '{{.Name}}'
    } | grep "^$PREFIX" || true
)
if [ -n "$left" ]; then
    echo "prefix objects remain:" >&2
    echo "$left" >&2
    exit 1
fi
echo "e2e stack is down"
