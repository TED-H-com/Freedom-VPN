#!/usr/bin/env python3
"""Foreground host admission and D-133 overlap receipt; no reservation."""
import os
import subprocess
import time

from followup16_preflight import identity
from run import ROOT, PUBLIC, write_json, utc


def foreign_runs():
    # Required inventory; never persist raw command lines of other lanes.
    result = subprocess.run(['bash', '-c', "ps -eo pid,etimes,args | grep -E '[g]ates\\.sh|[m]k/at-|[n]ode-test-netlab|[n]etlab-test|[e]2e-test'"], capture_output=True, text=True)
    own = set()
    pid = os.getpid()
    while pid:
        own.add(pid)
        _, pid = identity(pid)
    found = []
    for line in result.stdout.splitlines():
        fields = line.split()
        if not fields or not fields[0].isdigit():
            continue
        pid = int(fields[0])
        record, parent = identity(pid)
        if pid in own or record is None:
            continue
        lineage = set()
        while parent and parent not in lineage and parent not in own:
            lineage.add(parent)
            _, parent = identity(parent)
        # Our run's children are allowed; other descendants of the shell are not.
        if parent == os.getpid():
            continue
        found.append(record)
    return found


def main():
    started = time.monotonic()
    prior = float(os.getenv('FVPN_SECOPS_PRIOR_WAIT_SECONDS', '0'))
    if not 0 <= prior < 7200:
        raise SystemExit('prior waiting exhausted 120-minute allowance')
    record = dict(started_at=utc(), baseline_commit=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),
                  application_base=subprocess.check_output(['git','merge-base','HEAD','main'],cwd=ROOT,text=True).strip(),
                  prior_wait_seconds=prior, checks=[], admitted=False)
    while True:
        admission = subprocess.run(['bash', os.environ['FVPN_HOLD_SH']],capture_output=True,text=True)
        active = foreign_runs()
        free = admission.returncode == 0 and admission.stdout.strip() == 'free'
        elapsed = time.monotonic()-started
        record['checks'].append(dict(at=utc(), admission_free=free, active_processes=active))
        record['total_wait_seconds'] = prior+elapsed
        write_json(PUBLIC/'followup17-preflight.json', record)
        if free and not active:
            record['vmstat_command'] = 'vmstat 5 7'
            record['vmstat'] = subprocess.check_output(['vmstat','5','7'],text=True)
            # Recheck immediately before dispatch after the 30-second snapshot.
            admission = subprocess.run(['bash', os.environ['FVPN_HOLD_SH']],capture_output=True,text=True)
            active = foreign_runs()
            free = admission.returncode == 0 and admission.stdout.strip() == 'free'
            record['checks'].append(dict(at=utc(), admission_free=free, active_processes=active))
            if free and not active:
                break
        elapsed = time.monotonic()-started
        if prior+elapsed >= 7200:
            print('BLOCKED: total 120-minute host/suite wait exhausted')
            return 125
        print(f'Waiting for host/suite admission: {prior+elapsed:.1f}s total', flush=True)
        time.sleep(min(60,7200-prior-elapsed))
    record['admitted'] = True
    record['finished_at'] = utc()
    write_json(PUBLIC/'followup17-preflight.json', record)
    print(f"Host preflight passed after {record['total_wait_seconds']:.3f}s total waiting", flush=True)
    # Own child remains attached to this foreground driver; join before return.
    monitor = dict(started_at=utc(), samples=[], confounded=False, policy='D-133; any foreign assembled run overlap confounds this attempt; no retry')
    with subprocess.Popen(['python3','-B','tests/e2e/at/secops/admitted.py','make','-f','mk/at-secops.mk','at-secops-followup17'],cwd=ROOT) as child:
        while child.poll() is None:
            active = foreign_runs()
            monitor['samples'].append(dict(at=utc(), active_processes=active))
            monitor['confounded'] = monitor['confounded'] or bool(active)
            write_json(PUBLIC/'followup17-overlap.json',monitor)
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                pass
        monitor['finished_at'] = utc()
        monitor['exit_code'] = child.returncode
        write_json(PUBLIC/'followup17-overlap.json',monitor)
        return child.returncode


if __name__ == '__main__':
    raise SystemExit(main())
