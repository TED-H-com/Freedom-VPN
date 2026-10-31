#!/usr/bin/env bash
# Foreground D-096 campaign. No host networking and no external test traffic.
set -euo pipefail
cd "$(dirname "$0")/../../.."
export GOMAXPROCS=4 GOFLAGS=-p=4 CGO_ENABLED=0 GOTOOLCHAIN=local
prefix=${FVPN_LAB_PREFIX:-fvpn-w1-netlab-real}
case "$prefix" in fvpn-w1-netlab-*) ;; *) echo 'requires fvpn-w1-netlab-* prefix' >&2; exit 1;; esac
# The shared host permits at most two integration campaigns at once.
while [ "$(pgrep -fc '^bash bin/gates[.]sh' || true)" -ge 2 ]; do
    echo 'Two gates.sh campaigns are active; waiting for a lab slot.'
    sleep 30
done
if [ "${1:-real-netbird}" = lab ]; then
    trap 'make -f mk/netlab.mk netlab-down FVPN_LAB_PREFIX="$prefix" >/dev/null 2>&1 || true; docker rmi "$prefix:lab" >/dev/null 2>&1 || true' EXIT
    make -f mk/netlab.mk netlab-test FVPN_LAB_PREFIX="$prefix"
    exit
fi
mkdir -p .agent/tmp
scratch=$(mktemp -d "$PWD/.agent/tmp/real-netbird.XXXXXX")
export TMPDIR="$scratch" GOTMPDIR="$scratch"
cleanup() {
    docker rm -fv "$prefix" >/dev/null 2>&1 || true
    docker rmi "$prefix:real" "$prefix:lab" >/dev/null 2>&1 || true
    rm -rf "$scratch"
}
trap cleanup EXIT
# Privileged scratch is in the container's disk overlay, removed with it.
go test -c -o "$scratch/agent.test" ./node/agent
docker build -t "$prefix:lab" tests/netlab
docker build --build-arg "LAB_IMAGE=$prefix:lab" -f tests/netlab/netbird.Dockerfile -t "$prefix:real" tests/netlab
docker run -d --name "$prefix" --hostname "$prefix" --privileged --network none \
    -v "$PWD:/work:ro" "$prefix:real"
docker exec "$prefix" python3 /work/tests/netlab/scripts/real_netbird.py
docker cp "$scratch/agent.test" "$prefix:/agent.test"
docker exec -e FVPN_REAL_NETBIRD=1 -e TMPDIR=/scratch -e GOMAXPROCS=4 \
    "$prefix" /agent.test -test.run '^TestD096_RealNetBird$' -test.v -test.timeout=4m
