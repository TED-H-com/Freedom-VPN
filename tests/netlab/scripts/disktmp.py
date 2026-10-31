#!/usr/bin/env python3
"""Run a foreground gate with private disk scratch and Python-owned teardown."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile

root = Path(__file__).resolve().parents[3]
base = root / ".agent" / "tmp"
base.mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix="netlab-", dir=base) as scratch:
    env = dict(os.environ, TMPDIR=scratch, GOTMPDIR=scratch,
               CGO_ENABLED="0", GOTOOLCHAIN="local")
    result = subprocess.run(sys.argv[1:], env=env, check=False)
sys.exit(result.returncode)
