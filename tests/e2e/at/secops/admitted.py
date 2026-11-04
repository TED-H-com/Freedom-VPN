#!/usr/bin/env python3
"""Run foreground commands with admission checked before nested host commands."""
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import sys
import tempfile


def main():
    root = Path(__file__).resolve().parents[4]
    if not os.environ.get('FVPN_HOLD_SH'):
        raise SystemExit('FVPN_HOLD_SH is required')
    scratch = root / '.agent/tmp'
    scratch.mkdir(parents=True, exist_ok=True)
    private = root / '.agent/secops'
    private.mkdir(parents=True, exist_ok=True)
    original_path = os.environ['PATH']
    with tempfile.TemporaryDirectory(prefix='admitted.', dir=scratch) as directory:
        for name in ('docker', 'make', 'go', 'python3'):
            executable = shutil.which(name)
            if not executable:
                raise SystemExit('missing command: '+name)
            wrapper = Path(directory) / name
            wrapper.write_text('''#!/bin/bash
set -u
for attempt in $(seq 1 10); do
  admission=$(PATH='''+shlex.quote(original_path)+''' bash "$FVPN_HOLD_SH" 2>&1)
  result=$?
  printf '%s %s %s\\n' "$(date -u +%FT%TZ)" '''+shlex.quote(name)+''' "$admission" >> "$FVPN_SECOPS_ADMISSION_LOG"
  if [ "$result" = 0 ] && [ "$admission" = free ]; then
    exec '''+shlex.quote(executable)+''' "$@"
  fi
  if [ "$attempt" = 10 ]; then exit 125; fi
  sleep 60
done
''')
            wrapper.chmod(0o700)
        env = dict(os.environ, PATH=directory+':'+original_path, TMPDIR=str(scratch), GOTMPDIR=str(scratch),
                   PYTHONDONTWRITEBYTECODE='1', FVPN_SECOPS_ADMISSION_LOG=str(private/'admission.log'))
        return subprocess.run(sys.argv[1:], cwd=root, env=env, check=False).returncode


if __name__ == '__main__':
    raise SystemExit(main())
