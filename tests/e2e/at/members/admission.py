"""Hard admission stop; count suite roots, not nested make/wrapper processes."""
import os
import subprocess
import sys


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


def check():
    found = subprocess.run(['pgrep', '-af', 'gates.sh|mk/at-'], text=True, capture_output=True)
    if found.returncode not in (0, 1):
        raise RuntimeError('suite inventory unavailable')
    parents = {}
    for line in subprocess.check_output(['ps', '-eo', 'pid=,ppid='], text=True).splitlines():
        pid, parent = map(int, line.split())
        parents[pid] = parent
    matches = {int(line.split(maxsplit=1)[0]): line for line in found.stdout.splitlines()}
    roots = suite_roots(matches, parents, os.getpid())
    print('Admission: other active suites =', len(roots), flush=True)
    for pid in roots:
        # Keep the raw process command in private run logs only.
        print(matches[pid], flush=True)
    if len(roots) >= 2:
        print('HARD STOP: two or more other suites; no test launched', flush=True)
        return 75
    return 0


if __name__ == '__main__':
    sys.exit(check())
