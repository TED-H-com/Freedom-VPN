#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
STATE_DIR="${SCRIPT_DIR}/.state"

cd "${REPO_ROOT}"
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./internal/dnspolicy/cmd/bootstrap \
  --state-dir "${STATE_DIR}" \
  --upstream "upstream:5353"
