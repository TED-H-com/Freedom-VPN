#!/usr/bin/env python3
"""Foreground disk-temp runner; explicit teardown, no shell cleanup traps."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time

sys.dont_write_bytecode = True

ROOT = Path(__file__).resolve().parents[4]
os.chdir(ROOT)

def slots():
    while True:
        result = subprocess.run(['pgrep', '-af', 'gates.sh|mk/at-'], text=True, capture_output=True)
        print('shared-host suite inventory:\n' + result.stdout, flush=True)
        # Count top-level suite process trees, not their sh/make descendants.
        rows = []
        for line in result.stdout.splitlines():
            pid, _, command = line.partition(' ')
            if pid.isdigit() and ('make ' in command or 'gates.sh' in command) and 'pgrep' not in command:
                rows.append(int(pid))
        def parent(pid):
            try:
                return int(subprocess.check_output(['ps', '-o', 'ppid=', '-p', str(pid)], text=True).strip())
            except (OSError, ValueError, subprocess.CalledProcessError):
                return 0
        ancestors = set()
        pid = os.getpid()
        while pid:
            ancestors.add(pid)
            pid = parent(pid)
        other = set(rows) - ancestors
        roots = []
        for pid in other:
            p = parent(pid)
            while p and p not in other:
                p = parent(p)
            if not p:
                roots.append(pid)
        if len(roots) < 2:
            return
        print('Two other suites active; waiting 30 seconds.', flush=True)
        time.sleep(30)

def call(args, env):
    print('+ ' + ' '.join(args), flush=True)
    return subprocess.run(args, env=env).returncode

(ROOT / '.agent/tmp').mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix='at-isolation.', dir=ROOT / '.agent/tmp') as temp:
    env = dict(os.environ, TMPDIR=temp, GOTMPDIR=temp, GOTOOLCHAIN='local', CGO_ENABLED='0')
    if sys.argv[1:] != ['assembled']:
        raise SystemExit(call(sys.argv[1:], env))
    import uuid
    env.setdefault('FVPN_AT_RUN_ID', str(uuid.uuid4()))
    env['FVPN_NETLAB'] = '1'
    env['FVPN_E2E_GRANT_SEAM'] = '1'
    env['FVPN_LAB_PREFIX'] = 'fvpn-w4-isolation-lab'
    evidence = str(ROOT / env.get('FVPN_AT_EVIDENCE_DIR', 'docs/e2e/at/isolation'))
    env['FVPN_AT_EVIDENCE_DIR'] = evidence
    env['FVPN_NETLAB_EVIDENCE_DIR'] = evidence + '/reload-stream'
    lab = stack = 1
    operation_observer = None
    lifecycle_observer = None
    import lifecycle_events
    try:
        slots()
        lab = call(['sh', 'tests/e2e/at/isolation/lab.sh', 'up'], env)
        if lab == 0:
            slots()  # Recheck after stack startup before running its suites.
            reload = call(['go', 'test', '-tags', 'netlab', '-count=1', '-v', '-timeout=10m', '-run', '^(TestAT10_AT13_ReloadRevokeBaseline|TestD104_AT13_DeadlineContinuity)$', './tests/netlab'], env)
            owned = call(['go', 'test', '-tags', 'netlab', '-count=1', '-timeout=40m', '-p', '1', '-parallel', '1', './tests/netlab/at'], env)
            lab = reload or owned
        lab = call(['sh', 'tests/e2e/at/isolation/lab.sh', 'down'], env) or lab
        call(['python3', 'tests/e2e/at/isolation/stack.py', 'down'], env)
        slots()
        stack = call(['python3', 'tests/e2e/at/isolation/stack.py', 'up'], env)
        if stack == 0:
            slots()
            lifecycle_observer = lifecycle_events.start(temp)
            operation_observer = subprocess.Popen(['python3', '-B', 'tests/e2e/at/isolation/capture_hold_operations.py', evidence], env=env)
            stack = call(['python3', 'tests/e2e/at/isolation/driver.py', 'run'], env)
            review = call(['python3', '-B', 'tests/e2e/at/isolation/review_followup10.py', evidence], env)
            stack = stack or review
    finally:
        lab = call(['sh', 'tests/e2e/at/isolation/lab.sh', 'down'], env) or lab
        stack = call(['python3', 'tests/e2e/at/isolation/stack.py', 'down'], env) or stack
        if lifecycle_observer is not None:
            lifecycle_events.finish(lifecycle_observer, Path(evidence))
        if operation_observer is not None:
            try:
                operation_observer.wait(timeout=5)
            except subprocess.TimeoutExpired:
                operation_observer.terminate()
                operation_observer.wait(timeout=5)
    inventory = {}
    for kind, args in {'containers':['ps','-a'], 'images':['images'], 'networks':['network','ls'], 'volumes':['volume','ls']}.items():
        fmt = '{{.Repository}}:{{.Tag}}' if kind == 'images' else '{{.Names}}' if kind == 'containers' else '{{.Name}}'
        out = subprocess.check_output(['docker', *args, '--format', fmt], text=True)
        inventory[kind] = [s for s in out.splitlines() if s.startswith('fvpn-w4-isolation')]
    import json
    Path(evidence, 'cleanup-inventory.json').write_text(json.dumps(inventory, indent=2)+'\n')
    print('owned-prefix inventory: ' + json.dumps(inventory), flush=True)
    print(f'at-isolation lab={lab} stack={stack}', flush=True)
    raise SystemExit(2 if lab or stack or any(inventory.values()) else 0)
