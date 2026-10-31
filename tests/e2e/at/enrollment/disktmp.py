#!/usr/bin/env python3
"""Run a foreground gate with unique disk scratch and automatic cleanup."""

import os
from pathlib import Path
import subprocess
import sys
import tempfile

root = Path(__file__).resolve().parents[4]
scratch_root = root / ".agent/tmp"
scratch_root.mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix="at-enrollment-gate-", dir=scratch_root) as scratch:
    env = dict(os.environ, TMPDIR=scratch, GOTMPDIR=scratch,
               PYTHONDONTWRITEBYTECODE="1", GOTOOLCHAIN="local", CGO_ENABLED="0")
    raise SystemExit(subprocess.call(sys.argv[1:], cwd=root, env=env))
