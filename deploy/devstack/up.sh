#!/bin/sh
# Bring up one prefixed dev stack (§3, §5). Secrets come from fvpn-e2e-prepare.
# No host network changes. Published ports are loopback-only.
set -eu
cd "$(dirname "$0")/../.."
ROOT=$(pwd)
SHORT=${FVPN_E2E_PREFIX:-e2e}
PORT_BASE=${FVPN_E2E_PORT_BASE:-18000}
STATE=${FVPN_E2E_STATE:-$ROOT/.agent/e2e-state-$SHORT}
PREFIX=fvpn-$SHORT
COMPOSE="docker compose -p $PREFIX -f $ROOT/deploy/devstack/compose.yaml"
export FVPN_E2E_PREFIX=$SHORT
export FVPN_E2E_PORT_BASE=$PORT_BASE
export FVPN_E2E_STATE=$STATE
RESET=${E2E_RESET:-0}

case $SHORT in
    [a-z]*[a-z0-9]) ;;
    *)
        echo "FVPN_E2E_PREFIX must be a short lowercase name" >&2
        exit 2
        ;;
esac
case $PORT_BASE in
    ''|*[!0-9]*)
        echo "FVPN_E2E_PORT_BASE must be an integer" >&2
        exit 2
        ;;
esac

mkdir -p "$ROOT/.agent/tmp" "$ROOT/.agent/e2e-build"
export TMPDIR=$ROOT/.agent/tmp GOTMPDIR=$ROOT/.agent/tmp GOTOOLCHAIN=local CGO_ENABLED=0

if [ "$RESET" = "1" ] || [ ! -f "$STATE/ready" ]; then
    # AdGuard rewrites its config as root. prepare cannot unlink that file.
    if [ -d "$STATE" ] && ! rm -rf "$STATE" 2>/dev/null; then
        docker rm -f "$PREFIX-state-reset" >/dev/null 2>&1 || true
        docker run --rm --name "$PREFIX-state-reset" \
            -v "$STATE:/state" --entrypoint /bin/sh \
            "ubuntu:24.04@sha256:fe81560990fae86dc4743202b2cb4d3df98d0bcf949173cecef15ce1127ef852" \
            -c 'find /state -mindepth 1 -delete'
        rm -rf "$STATE"
    fi
    go build -o "$ROOT/.agent/e2e-build/fvpn-e2e-prepare" ./tests/e2e/cmd/fvpn-e2e-prepare
    "$ROOT/.agent/e2e-build/fvpn-e2e-prepare" -state "$STATE" -reset -prefix "$SHORT" -port-base "$PORT_BASE"
fi
if [ ! -f "$STATE/render/compose.env" ]; then
    echo "compose env is missing; set E2E_RESET=1" >&2
    exit 1
fi
set -a
# shellcheck disable=SC1091
. "$STATE/render/compose.env"
set +a
if [ "$FVPN_E2E_PREFIX" != "$SHORT" ] || [ "$FVPN_E2E_PORT_BASE" != "$PORT_BASE" ]; then
    echo "state was prepared for a different prefix or port base; set E2E_RESET=1" >&2
    exit 1
fi

mkdir -p "$STATE/render/coredns" "$STATE/releases" "$STATE/node/var/e2e" "$STATE/member/var/e2e"
cp "$ROOT/deploy/devstack/nginx.conf" "$STATE/render/nginx.conf"
cp "$ROOT/deploy/devstack/coredns/Corefile" "$ROOT/deploy/devstack/coredns/example.invalid.zone" "$STATE/render/coredns/"

tag() {
    digest=$1
    name=$2
    if ! docker image inspect "$digest" >/dev/null 2>&1; then
        docker pull "$digest"
    fi
    docker tag "$digest" "$name"
}
tag "postgres@sha256:31ba9c94a03ada9988f2bb493a22a75eef3a185884533e5cdbcd498ea835b08f" "$PREFIX-postgres:17"
tag "nginx@sha256:55d1fb0944aafb8534b253e486c6cf57d3e0c785c4f6a7b3e9f02c5a080fb2ce" "$PREFIX-nginx:1.30.0"
tag "coredns/coredns@sha256:40384aa1f5ea6bfdc77997d243aec73da05f27aed0c5e9d65bfa98933c519d97" "$PREFIX-coredns:1.12.0"
tag "adguard/adguardhome@sha256:5d5e3aef39a8c8906a324f0c2da0a6123929e459b57c1e3f05e401dca5c7bf02" "$PREFIX-adguard:v0.107.59"
tag "netbirdio/netbird-server@sha256:7b5b53b2dc8bc25639f791cf72d47cf86911894ff3f8a97cb79bab9be958109c" "$PREFIX-netbird:0.82.0"

build_one() {
    pkg=$1
    name=${pkg##*/}
    echo "building $name"
    GOOS=linux GOARCH=amd64 go build -trimpath -o "$ROOT/.agent/e2e-build/$name" "$pkg"
}
build_one ./apps/control/cmd/fvpn-control
build_one ./apps/worker/cmd/fvpn-worker
build_one ./node/agent/cmd/fvpn-agent
build_one ./node/watchdog/cmd/fvpn-watchdog
build_one ./node/updater/cmd/fvpn-updater
build_one ./tests/e2e/cmd/fvpn-e2e-idp
build_one ./tests/e2e/cmd/fvpn-e2e-payment
build_one ./tests/e2e/cmd/fvpn-e2e-origin
build_one ./tests/e2e/cmd/fvpn-e2e-nbbootstrap

service_image() {
    name=$1
    bin=$2
    ctx=$ROOT/.agent/e2e-build/ctx-$name
    rm -rf "$ctx"
    mkdir -p "$ctx"
    cp "$bin" "$ctx/service"
    cp "$ROOT/deploy/devstack/Dockerfile.service" "$ctx/Dockerfile"
    docker build -t "$PREFIX-$name:local" "$ctx"
}
service_image idp "$ROOT/.agent/e2e-build/fvpn-e2e-idp"
service_image payment "$ROOT/.agent/e2e-build/fvpn-e2e-payment"
service_image origin "$ROOT/.agent/e2e-build/fvpn-e2e-origin"
service_image control "$ROOT/.agent/e2e-build/fvpn-control"
service_image worker "$ROOT/.agent/e2e-build/fvpn-worker"

ctx=$ROOT/.agent/e2e-build/ctx-node
rm -rf "$ctx"
mkdir -p "$ctx"
cp "$ROOT/.agent/e2e-build/fvpn-agent" "$ROOT/.agent/e2e-build/fvpn-watchdog" "$ROOT/.agent/e2e-build/fvpn-updater" "$ctx/"
cp "$ROOT/deploy/devstack/entrypoint-node.sh" "$ROOT/deploy/devstack/entrypoint-member.sh" "$ctx/"
cp "$ROOT/deploy/devstack/Dockerfile.node" "$ctx/Dockerfile"
docker build -t "$PREFIX-node:local" "$ctx"

$COMPOSE up -d postgres idp payment origin netbird gateway coredns adguard

wait_exec() {
    name=$1
    shift
    i=0
    last=""
    while [ "$i" -lt 40 ]; do
        if last=$(docker exec "$name" "$@" 2>&1); then
            return 0
        fi
        i=$((i + 1))
        sleep 2
    done
    echo "timed out: $name: $last" >&2
    return 1
}
wait_http() {
    url=$1
    limit=${2:-60}
    i=0
    last=""
    while [ "$i" -lt "$limit" ]; do
        if last=$(curl -fsS -k --max-time 3 -o /dev/null "$url" 2>&1); then
            return 0
        fi
        i=$((i + 1))
        sleep 2
    done
    echo "timed out: $url: $last" >&2
    return 1
}
wait_exec "$PREFIX-postgres" pg_isready -U fvpn -d fvpn
wait_http "https://127.0.0.1:${FVPN_E2E_PORT_IDP}/healthz" 60
wait_exec "$PREFIX-payment" wget -q -O /dev/null --no-check-certificate https://127.0.0.1/healthz
wait_http "https://127.0.0.1:${FVPN_E2E_PORT_ORIGIN}/healthz" 60

i=0
until "$ROOT/.agent/e2e-build/fvpn-e2e-nbbootstrap" -dial "127.0.0.1:${FVPN_E2E_PORT_GATEWAY}" -ca "$STATE/ca.pem" -state "$STATE"; do
    i=$((i + 1))
    if [ "$i" -ge 30 ]; then
        echo "netbird bootstrap failed" >&2
        exit 1
    fi
    sleep 2
done

$COMPOSE up -d control worker
wait_http "https://127.0.0.1:${FVPN_E2E_PORT_CONTROL}/readyz" 90
$COMPOSE up -d node member
echo "e2e stack is up"
