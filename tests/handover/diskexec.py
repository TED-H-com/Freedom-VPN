#!/usr/bin/env python3
"""Run a foreground handover check with unique disk scratch, without shell traps."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile

root = Path(__file__).resolve().parents[2]
scratch_root = root / '.agent/tmp'
scratch_root.mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix='handover.', dir=scratch_root) as scratch:
    result = subprocess.run(sys.argv[1:], cwd=root, check=False, env=dict(
        os.environ, TMPDIR=scratch, GOTMPDIR=scratch, GOTOOLCHAIN='local',
        CGO_ENABLED='0', PYTHONDONTWRITEBYTECODE='1'))
raise SystemExit(result.returncode)
