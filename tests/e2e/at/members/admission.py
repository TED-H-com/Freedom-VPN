"""Hard admission stop; count suite roots, not nested make/wrapper processes."""
import os
import subprocess
import sys
import time
import json
import shlex
from pathlib import Path


def suite_roots(matches, parents, own_pid):
    ancestors = set()
    while own_pid and own_pid not in ancestors:
        ancestors.add(own_pid)
        own_pid = parents.get(own_pid, 0)
    candidates = set(matches) - ancestors
    roots = []
    for pid in candidates:
        parent = parents.get(pid, 0)
        seen = set()
        nested = False
        while parent and parent not in seen:
            seen.add(parent)
            if parent in candidates:
                nested = True
                break
            parent = parents.get(parent, 0)
        if not nested:
            roots.append(pid)
    return sorted(roots)


def process_args(command):
    # ps/pgrep render argv without shell escaping. A literal unmatched quote in
    # an argument is not a malformed process and must not crash admission or
    # disappear from the inventory. Conservative tokenization keeps it counted.
    try:
        return shlex.split(command)
    except ValueError:
        return command.split()


def inventory():
    found = subprocess.run(['pgrep', '-af', 'gates.sh|mk/at-|node-test-netlab|mk/netlab|test.*tags.*netlab'], text=True, capture_output=True)
    if found.returncode not in (0, 1):
        raise RuntimeError('suite inventory unavailable')
    parents = {}
    for line in subprocess.check_output(['ps', '-eo', 'pid=,ppid='], text=True).splitlines():
        pid, parent = map(int, line.split())
        parents[pid] = parent
    matches = {}
    for line in found.stdout.splitlines():
        pid, command = line.split(maxsplit=1)
        args = process_args(command)
        # Ignore orchestration shells containing quoted command text.
        if any(a in ('-c', '-lc', '-ec') for a in args[:3]):
            continue
        if args and Path(args[0]).name != 'pgrep':
            matches[int(pid)] = line
    roots = suite_roots(matches, parents, os.getpid())
    return matches, roots


def priority(matches, roots):
    return any(any(term in matches[p] for term in ('mk/at-isolation.mk', 'node-test-netlab', 'mk/netlab', '-tags netlab')) for p in roots)


def check():
    matches, roots = inventory()
    print('Admission: other active suites =', len(roots), flush=True)
    for pid in roots:
        # Keep the raw process command in private run logs only.
        print(matches[pid], flush=True)
    if len(roots) >= 2 or priority(matches, roots):
        print('HARD STOP: isolation active or two other suites; no test launched', flush=True)
        return 75
    return 0


if __name__ == '__main__':
    sys.exit(check())


def wait_ready():
    started = time.monotonic()
    isolation_wait = False
    polls = []
    while True:
        matches, roots = inventory()
        isolation = priority(matches, roots) or any('gates.sh' in matches[p] for p in roots)
        isolation_wait = isolation_wait or isolation
        polls.append(dict(elapsed_seconds=round(time.monotonic()-started, 3), other_suites=len(roots), isolation_active=isolation))
        if not isolation and len(roots) < 2:
            break
        if time.monotonic()-started >= 90*60:
            raise RuntimeError('90-minute suite admission wait exhausted; no run started')
        print('Waiting for isolation / suite admission:', polls[-1], flush=True)
        time.sleep(60)
    receipt=dict(wait_seconds=round(time.monotonic()-started, 3), waited_for_isolation=isolation_wait, polls=polls)
    Path('.agent/tmp/f18-admission-wait.json').write_text(json.dumps(receipt, indent=2)+'\n')
    return receipt
