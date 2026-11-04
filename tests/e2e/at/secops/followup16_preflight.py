#!/usr/bin/env python3
"""Bounded foreground concurrency wait; never reserves the shared host."""
import os
from pathlib import Path
import subprocess
import time

from run import PUBLIC, write_json, utc


def identity(pid):
    try:
        raw = Path('/proc/'+str(pid)+'/stat').read_text()
        left, rest = raw.rsplit(')', 1)
        fields = rest.split()
        return dict(pid=pid, comm=left.split('(', 1)[1], ticks=str(int(fields[11])+int(fields[12]))), int(fields[1])
    except (OSError, ValueError, IndexError):
        return None, 0


def main():
    start = time.monotonic()
    prior_wait = float(os.environ.get("FVPN_SECOPS_PRIOR_WAIT_SECONDS", "0"))
    if not 0 <= prior_wait <= 7200:
        raise SystemExit("prior admission wait outside total 120-minute allowance")
    own = set()
    pid = os.getpid()
    while pid:
        own.add(pid)
        _, pid = identity(pid)
    checks = []
    while True:
        # Exact requested inventory. Raw command lines stay private in memory.
        result = subprocess.run(['bash', '-c', "ps -eo pid,etimes,args | grep -E '[g]ates\\.sh|[m]k/at-|[n]ode-test-netlab|[n]etlab-test'"], capture_output=True, text=True)
        active = []
        for line in result.stdout.splitlines():
            fields = line.split()
            if fields and fields[0].isdigit() and int(fields[0]) not in own:
                record, _ = identity(int(fields[0]))
                if record:
                    active.append(record)
        admission = subprocess.run(['bash', os.environ['FVPN_HOLD_SH']], capture_output=True, text=True)
        free = admission.returncode == 0 and admission.stdout.strip() == 'free'
        checks.append(dict(at=utc(), elapsed_seconds=time.monotonic()-start, admission_free=free, active_processes=active))
        done = free and not active
        elapsed = time.monotonic()-start
        write_json(PUBLIC/'followup16-preflight.json', dict(started_at=checks[0]['at'], finished_at=utc() if done else None,
                   wait_seconds=elapsed, prior_admission_wait_seconds=prior_wait, total_wait_seconds=prior_wait+elapsed,
                   limit_seconds=7200, admitted=done, checks=checks))
        if done:
            print(f'PASS: host and suite admission after {elapsed:.3f} seconds')
            return 0
        if prior_wait+elapsed >= 7200:
            print('BLOCKED: 120-minute concurrency wait exhausted')
            return 125
        print(f'Waiting for host/suite admission: {elapsed:.0f}s', flush=True)
        time.sleep(min(60, 7200-prior_wait-elapsed))


if __name__ == '__main__':
    raise SystemExit(main())
