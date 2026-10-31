#!/usr/bin/env bash
# Go build/test scratch must be on disk, including standalone area targets.
set -euo pipefail
exec python3 - "$@" <<'PY'
import os
from pathlib import Path
import subprocess
import sys
import tempfile

scratch = Path('.agent/tmp').resolve()
scratch.mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix='contracts.', dir=scratch) as task_tmp:
    env = dict(os.environ, TMPDIR=task_tmp, GOTMPDIR=task_tmp)
    result = subprocess.run(sys.argv[1:], env=env, check=False)
sys.exit(result.returncode)
PY
