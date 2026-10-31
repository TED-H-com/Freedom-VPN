#!/usr/bin/env bash
# D-085: foreground-only engine/read latency and injection regression.
set -euo pipefail
cd "$(dirname "$0")/../../.."
export GOMAXPROCS=4 GOFLAGS=-p=4 CGO_ENABLED=0 GOTOOLCHAIN=local
mkdir -p .agent/tmp
task_tmp=$(mktemp -d "$PWD/.agent/tmp/live.XXXXXX")
export TMPDIR="$task_tmp" GOTMPDIR="$task_tmp"
lab_name="fvpn-w2-agent-core-live-$$"
lab_image="$lab_name:lab"
cleanup() {
    docker rm -fv "$lab_name" >/dev/null 2>&1 || true
    # A killed test can leave root-owned 0700 temp directories on the mount.
    if [ -d "$task_tmp/scratch" ]; then
        docker run --rm --name "$lab_name-clean" --network none \
            -v "$task_tmp/scratch:/scratch" "$lab_image" \
            find /scratch -mindepth 1 -delete >/dev/null 2>&1 || true
    fi
    docker rmi "$lab_image" >/dev/null 2>&1 || true
    rm -rf "$task_tmp"
}
trap cleanup EXIT
go test -c -tags netlab -o "$task_tmp/agent.test" ./node/agent
docker build -t "$lab_image" tests/netlab
# Scratch stays on disk inside the worktree. The test creates its veth pairs
# only in this isolated network namespace; the host has no dataplane changes.
mkdir "$task_tmp/scratch"
docker run --rm --name "$lab_name" --privileged --network none \
    -e FVPN_AGENT_LIVE_NETLAB=1 -e GOMAXPROCS=4 -e TMPDIR=/scratch \
    -v "$task_tmp/agent.test:/agent.test:ro" -v "$task_tmp/scratch:/scratch" \
    "$lab_image" /agent.test -test.run '^TestD085_LiveReadNetlab$' -test.v -test.timeout=2m
