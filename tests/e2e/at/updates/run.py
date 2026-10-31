#!/usr/bin/env python3
"""Foreground acceptance runner; disk scratch and cleanup without shell traps."""
import os
import time
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[4]


def run(args, env):
    return subprocess.run(args, cwd=ROOT, env=env, check=False).returncode


def main():
    (ROOT / '.agent/tmp').mkdir(parents=True, exist_ok=True)
    scratch = tempfile.mkdtemp(prefix='at-updates-', dir=ROOT / '.agent/tmp')
    env = dict(os.environ, TMPDIR=scratch, GOTMPDIR=scratch,
               GOTOOLCHAIN='local', CGO_ENABLED='0')
    live = sys.argv[1] == 'live'
    code = 1
    try:
        if live:
            env.update(FVPN_E2E_PREFIX='w4-updates', FVPN_E2E_PORT_BASE='42000',
                       FVPN_E2E_STATE=str(ROOT / '.agent/e2e-state-w4-updates'))
            if os.environ.get('AT_UPDATES_REUSE') != '1':
                code = run(['make', '-f', 'mk/e2e.mk', 'e2e-down'], env)
                if code:
                    return code
                code = run(['make', '-f', 'mk/e2e.mk', 'e2e-up', 'E2E_RESET=1'], env)
                if code:
                    return code
            # Recheck shared-host admission immediately after startup.
            run(['pgrep', '-af', 'gates.sh|mk/at-'], env)
            env['FVPN_E2E'] = '1'
            started = str(time.time())
            code = run(['go', 'test', '-tags', 'e2e', '-count=1', '-v',
                        '-timeout', '45m', '-p', '1', './tests/e2e/at/updates',
                        '-run', '^TestUpdatesFollowup$'], env)
            supplemental = run(['python3', 'tests/e2e/at/updates/supplemental.py', started], env)
            if supplemental:
                code = code or supplemental
        else:
            for args in [
                ['go', 'vet', './tests/e2e/at/updates'],
                ['go', 'vet', '-tags', 'e2e', './tests/e2e/at/updates'],
                ['go', 'run', 'honnef.co/go/tools/cmd/staticcheck@v0.8.2', './tests/e2e/at/updates'],
                ['go', 'run', 'honnef.co/go/tools/cmd/staticcheck@v0.8.2', '-tags', 'e2e', './tests/e2e/at/updates'],
                ['go', 'test', '-tags', 'e2e', './tests/e2e/at/updates', '-run', '^(TestUpdatesFollowupEvidence|TestD098_Followup9ArchivePins|TestD132_Followup10ArchivePins|TestD132_Followup11ArchivePins|TestD098_Followup12FirstAttemptArchivePins)$'],
            ]:
                code = run(args, env)
                if code:
                    return code
            print('at-updates-lint: gofmt, vet and staticcheck (offline + e2e) passed')
    finally:
        if live:
            teardown = run(['make', '-f', 'mk/e2e.mk', 'e2e-down'], env)
            if teardown:
                code = teardown
        shutil.rmtree(scratch)
    return code


if __name__ == '__main__':
    sys.exit(main())
