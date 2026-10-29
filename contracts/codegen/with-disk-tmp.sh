#!/usr/bin/env bash
# Go build/test scratch must be on disk, including standalone area targets.
set -euo pipefail
mkdir -p .agent/tmp
task_tmp=$(mktemp -d "$(pwd)/.agent/tmp/contracts.XXXXXX")
trap 'python3 -c "import shutil,sys; shutil.rmtree(sys.argv[1])" "$task_tmp"' EXIT
export TMPDIR="$task_tmp" GOTMPDIR="$task_tmp"
"$@"
