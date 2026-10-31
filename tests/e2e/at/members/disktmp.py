"""Run an area gate with disk scratch and deterministic scoped cleanup."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile

root = Path(__file__).resolve().parents[4]
base = root / '.agent' / 'tmp'
base.mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix='at-members.', dir=base) as scratch:
    env = dict(os.environ, TMPDIR=scratch, GOTMPDIR=scratch, AT_MEMBERS_TMP=scratch)
    result = subprocess.run(sys.argv[1:], env=env, cwd=root, check=False)
sys.exit(result.returncode)
