#!/usr/bin/env python3
"""Foreground disk-temp runner; explicit teardown, no shell cleanup traps."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import json
from datetime import datetime, timezone

sys.dont_write_bytecode = True

ROOT = Path(__file__).resolve().parents[4]
os.chdir(ROOT)

from suite_admission import wait as wait_for_suites

def slots():
    return wait_for_suites(os.environ.get('FVPN_AT_EVIDENCE_DIR'))

phases = []

def call(args, env):
    print('+ ' + ' '.join(args), flush=True)
    code = subprocess.run(args, env=env).returncode
    phases.append(dict(name=" ".join(args[:3]), exit_code=code))
    return code

(ROOT / '.agent/tmp').mkdir(parents=True, exist_ok=True)
with tempfile.TemporaryDirectory(prefix='at-isolation.', dir=ROOT / '.agent/tmp') as temp:
    env = dict(os.environ, TMPDIR=temp, GOTMPDIR=temp, GOTOOLCHAIN='local', CGO_ENABLED='0', PYTHONDONTWRITEBYTECODE='1')
    if sys.argv[1:] != ['assembled']:
        raise SystemExit(call(sys.argv[1:], env))
    import uuid
    env.setdefault('FVPN_AT_RUN_ID', str(uuid.uuid4()))
    git_env = dict(env); git_env.pop('LD_LIBRARY_PATH', None)
    invocation = dict(run_id=env['FVPN_AT_RUN_ID'],
        build=subprocess.check_output(['git','rev-parse','HEAD'],env=git_env,text=True).strip(),
        source_dirty=bool(subprocess.check_output(['git','status','--porcelain'],env=git_env)),
        started_at=datetime.now(timezone.utc).isoformat().replace('+00:00','Z'),
        command=('FVPN_AT_SCHEDULED=1 ' if env.get('FVPN_AT_SCHEDULED')=='1' else '')+'make -f mk/at-isolation.mk at-isolation')
    env['FVPN_NETLAB'] = '1'
    env['FVPN_E2E_GRANT_SEAM'] = '1'
    env['FVPN_LAB_PREFIX'] = 'fvpn-w4-isolation-lab'
    evidence = str(ROOT / env.get('FVPN_AT_EVIDENCE_DIR', 'docs/e2e/at/isolation'))
    env['FVPN_AT_EVIDENCE_DIR'] = evidence
    os.environ['FVPN_AT_EVIDENCE_DIR'] = evidence
    Path(evidence).mkdir(parents=True, exist_ok=True)
    Path(evidence, 'invocation.json').write_text(json.dumps(invocation, indent=2)+'\n')
    env['FVPN_NETLAB_EVIDENCE_DIR'] = evidence + '/reload-stream'
    lab = stack = 1
    operation_observer = None
    lifecycle_observer = None
    import lifecycle_events
    try:
        if env.get('FVPN_AT_SKIP_LAB') == '1':
            lab = 0
            phases.append(dict(name='lab', status='not executed', reason='explicit FVPN_AT_SKIP_LAB=1; retained lab evidence is historical'))
            print('lab not executed: explicit FVPN_AT_SKIP_LAB=1', flush=True)
        else:
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
            boundary = call(['python3', '-B', 'tests/e2e/at/isolation/review_boundary.py', evidence], env)
            stack = stack or boundary
            # D-163 may explain only this invocation's result, never a leftover.
            Path(evidence, 'populated-holds-review.json').unlink(missing_ok=True)
            holds = call(['python3', '-B', 'tests/e2e/at/isolation/review_populated_holds.py', evidence], env)
            stack = stack or holds
    except Exception as error:
        phases.append(dict(name='assembled setup/collection exception', exit_code=1, error_type=type(error).__name__))
        stack = 1
    finally:
        if env.get('FVPN_AT_SKIP_LAB') != '1':
            lab = call(['sh', 'tests/e2e/at/isolation/lab.sh', 'down'], env) or lab
        stack = call(['python3', 'tests/e2e/at/isolation/stack.py', 'down'], env) or stack
        if lifecycle_observer is not None:
            lifecycle_events.finish(lifecycle_observer, Path(evidence), invocation)
        if operation_observer is not None:
            try:
                operation_observer.wait(timeout=5)
            except subprocess.TimeoutExpired:
                operation_observer.terminate()
                try:
                    operation_observer.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    operation_observer.kill();operation_observer.wait(timeout=5)
                    phases.append(dict(name='operation observer forced cleanup', exit_code=1))
            invocation['operation_observer_exit_code'] = operation_observer.returncode
    invocation['completed_at'] = datetime.now(timezone.utc).isoformat().replace('+00:00','Z')
    from auxiliary_inputs import seal
    seal(Path(evidence), invocation)
    auxiliary = []
    for script, output in [('review_stops.py','stop-receipt-review.json'), ('review_clone.py','clone-transport-review.json')]:
        code = subprocess.run(['python3','-B','tests/e2e/at/isolation/'+script,evidence,env['FVPN_AT_RUN_ID']], env=env).returncode
        auxiliary.append(dict(script=script, output=output, exit_code=code))
    Path(evidence, 'auxiliary-reviews.json').write_text(json.dumps(dict(invocation, reviews=auxiliary,
        scope='Separate from AT and scheduled mapping; any auxiliary nonzero exit remains in the assembled command result'), indent=2)+'\n')
    inventory = {}
    for kind, args in {'containers':['ps','-a'], 'images':['images'], 'networks':['network','ls'], 'volumes':['volume','ls']}.items():
        fmt = '{{.Repository}}:{{.Tag}}' if kind == 'images' else '{{.Names}}' if kind == 'containers' else '{{.Name}}'
        out = subprocess.check_output(['docker', *args, '--format', fmt], text=True)
        inventory[kind] = [s for s in out.splitlines() if s.startswith('fvpn-w4-isolation')]
    import json
    Path(evidence, 'cleanup-inventory.json').write_text(json.dumps(inventory, indent=2)+'\n')
    print('owned-prefix inventory: ' + json.dumps(inventory), flush=True)
    if env.get('FVPN_AT_SCHEDULED') == '1':
        from nightly_summary import summarize
        result = summarize(Path(evidence), phases, env['FVPN_AT_RUN_ID'])
        if env.get('GITHUB_STEP_SUMMARY'):
            with open(env['GITHUB_STEP_SUMMARY'], 'a') as f:
                f.write(Path(evidence, 'scheduled-job.md').read_text())
        # Every subprocess exit is retained in phases and evaluated by the
        # scheduled summary, including D-163's narrowly listed coverage gap.
        stack = int(result['status'] != 'pass')
    print(f'at-isolation lab={lab} stack={stack}', flush=True)
    raise SystemExit(2 if lab or stack or any(inventory.values()) or any(r["exit_code"] for r in auxiliary) else 0)
