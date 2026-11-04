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
            if os.environ.get('FVPN_SECOPS_TIMING_JOURNAL'):
                # Executable wrappers remain private; the committed harness owns
                # admission, timing receipts and offline validation.
                wrapper.write_text('#!/bin/bash\nset -e\n'+shlex.quote(sys.executable)+
                    ' -B '+shlex.quote(str(root/'tests/e2e/at/secops/timing_admission.py'))+
                    ' command '+shlex.quote(name)+'\nexec '+shlex.quote(executable)+' "$@"\n')
            else:
                wrapper.write_text('#!/bin/bash\nset -u\nfor attempt in $(seq 1 121); do\n  admission=$(PATH='+shlex.quote(original_path)+' bash "$FVPN_HOLD_SH" 2>&1)\n  result=$?\n  printf \'%s %s %s\\n\' "$(date -u +%FT%TZ)" '+shlex.quote(name)+' "$admission" >> "$FVPN_SECOPS_ADMISSION_LOG"\n  if [ "$result" = 0 ] && [ "$admission" = free ]; then\n    exec '+shlex.quote(executable)+' "$@"\n  fi\n  if [ "$attempt" = 121 ]; then exit 125; fi\n  sleep 60\ndone\n')
            wrapper.chmod(0o700)
        env = dict(os.environ, PATH=directory+':'+original_path, TMPDIR=str(scratch), GOTMPDIR=str(scratch),
                   MAKE=str(Path(directory)/'make'), PYTHONDONTWRITEBYTECODE='1', FVPN_SECOPS_ADMISSION_LOG=str(private/'admission.log'))
        return subprocess.run(sys.argv[1:], cwd=root, env=env, check=False).returncode


if __name__ == '__main__':
    raise SystemExit(main())
