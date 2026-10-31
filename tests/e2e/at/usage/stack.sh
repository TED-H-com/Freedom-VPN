#!/usr/bin/env bash
# Run the shared driver with lane-owned image tags; no product/lab fork.
set -euo pipefail
cd "$(dirname "$0")/../../../.."
export FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000
case "${1:-}" in
  up)
    mkdir -p .agent/usage-docker
    export AT_USAGE_DOCKER
    AT_USAGE_DOCKER=$(command -v docker)
    cat > .agent/usage-docker/docker <<'WRAPPER'
#!/usr/bin/env python3
import os
import sys

args = [arg.replace("fvpn-devstack-", "fvpn-w4-usage-") for arg in sys.argv[1:]]
os.execv(os.environ["AT_USAGE_DOCKER"], ["docker", *args])
WRAPPER
    chmod +x .agent/usage-docker/docker
    export PATH="$PWD/.agent/usage-docker:$PATH"
    export FVPN_E2E_IMAGE_NGINX=fvpn-w4-usage-nginx:1.30.0
    export FVPN_E2E_IMAGE_COREDNS=fvpn-w4-usage-coredns:1.12.0
    export FVPN_E2E_IMAGE_ADGUARD=fvpn-w4-usage-adguard:v0.107.59
    export FVPN_E2E_IMAGE_NETBIRD=fvpn-w4-usage-netbird:0.82.0
    git rev-parse HEAD > .agent/usage-image-source.txt
    E2E_RESET=1 make -f mk/e2e.mk e2e-up
    ;;
  down) make -f mk/e2e.mk e2e-down ;;
  *) echo 'usage: bash tests/e2e/at/usage/stack.sh up|down' >&2; exit 2 ;;
esac
