#!/bin/sh
# Build unprivileged first; only the isolated topology needs privileged Docker.
set -eu
cd "$(dirname "$0")/../../../.."
repo=$(pwd)
prefix=${FVPN_LAB_PREFIX:-fvpn-w4-isolation-lab}
case "$prefix" in fvpn-w4-isolation-*) ;; *) echo 'wrong isolation lab prefix' >&2; exit 2 ;; esac
state=$repo/.agent/at-isolation-lab
image=$prefix:lab
mkdir -p "$state/scratch"

wipe() {
    docker run --rm --name "$prefix-cleanup" --network none \
        -v "$state/scratch:/scratch" "$image" \
        find /scratch -mindepth 1 -delete
}

case "${1:-}" in
    up)
        mkdir -p "$repo/.agent/tmp"
        task_tmp=$(mktemp -d "$repo/.agent/tmp/at-isolation-lab.XXXXXX")
        trap 'rm -rf "$task_tmp"' EXIT
        export TMPDIR=$task_tmp GOTMPDIR=$task_tmp GOTOOLCHAIN=local CGO_ENABLED=0
        go build -o "$state/fvpn-netlab" ./tests/netlab/cmd/fvpn-netlab
        docker build -t "$image" tests/netlab
        wipe
        docker run -d --name "$prefix" --privileged --network none \
            -v "$repo:/work:ro" -v "$state/scratch:/scratch" \
            -v "$state/fvpn-netlab:/usr/local/bin/fvpn-netlab:ro" "$image"
        docker exec "$prefix" bash /work/tests/netlab/scripts/topology.sh
        ;;
    down)
        docker rm -f "$prefix" >/dev/null 2>&1 || true
        if docker image inspect "$image" >/dev/null 2>&1; then
            wipe
            docker rmi "$image" >/dev/null
        fi
        rm -f "$state/fvpn-netlab"
        rmdir "$state/scratch" "$state" 2>/dev/null || true
        ;;
    *) echo 'usage: lab.sh up|down' >&2; exit 2 ;;
esac
