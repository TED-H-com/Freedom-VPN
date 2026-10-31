#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../../.."
exec python3 tests/netlab/agent/run-live.py
