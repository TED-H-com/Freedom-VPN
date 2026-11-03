#!/usr/bin/env python3
"""TEST ONLY foreground receipts. Invoke through tests/netlab/scripts/disktmp.py."""
import gzip
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path.cwd()
OUT = ROOT / 'docs/poc/G2-followup26-evidence'


def write(name, value):
    p = OUT / name
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(value, indent=2) + '\n')


def processes():
    # Raw arguments may contain whole task prompts; retain only PID/kind publicly.
    hold = subprocess.run(['bash', os.environ['FVPN_F26_HOLD_SCRIPT']], capture_output=True, text=True)
    p = subprocess.run(['pgrep', '-af', 'gates.sh|mk/at-|go test'], capture_output=True, text=True)
    assert p.returncode in (0, 1)
    listing = subprocess.check_output(['ps', '-eo', 'pid=,ppid=,args='], text=True)
    private = ROOT / '.agent/tmp/f26/resume-processes.txt'
    private.parent.mkdir(parents=True, exist_ok=True)
    private.write_text(hold.stdout + hold.stderr + '\n' + p.stdout + '\n' + listing)
    rows, parents = [], {}
    for line in listing.splitlines():
        fields = line.split()
        if len(fields) < 3:
            continue
        pid, ppid = map(int, fields[:2])
        parents[pid] = ppid
        exe, args = Path(fields[2]).name, fields[3:]
        kind = None
        if exe == 'go' and args and args[0] == 'test':
            kind = 'go-test'
        elif exe in ('make', 'gmake'):
            if any('mk/at-' in a for a in args):
                kind = 'at-suite'
            elif 'node-test-netlab' in args:
                kind = 'node-test-netlab'
            elif 'netlab-test' in args:
                kind = 'netlab-test'
        elif exe.startswith('python') and '-c' not in args and any(a.endswith('.agent/f36-resume/compare.py') for a in args):
            kind = 'agent-core36-comparison'
        elif exe in ('bash', 'sh'):
            script = next((a for a in args if not a.startswith('-')), '')
            flags = args[:args.index(script)] if script else args
            command_string = any(a.startswith('-') and not a.startswith('--') and 'c' in a[1:] for a in flags)
            if not command_string and Path(script).name == 'gates.sh':
                kind = 'gates-suite'
        elif exe == 'gates.sh':
            kind = 'gates-suite'
        if kind:
            rows.append(dict(pid=pid, kind=kind))
    def ours(pid):
        seen = set()
        while pid in parents and pid not in seen:
            if pid == os.getpid():
                return True
            seen.add(pid)
            pid = parents[pid]
        return False
    other = [r for r in rows if not ours(r['pid'])]
    own = [r for r in rows if ours(r['pid'])]
    free = hold.returncode == 0 and hold.stdout.strip() == 'free'
    return dict(utc=time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
                hold_command=['bash', '<admission-script>'], hold_exit=hold.returncode,
                hold_output='free' if free else 'not free',
                hold_raw_sha256=hashlib.sha256((hold.stdout+hold.stderr).encode()).hexdigest(),
                other_runs=other, own_runs=own, ready=free and not other,
                pgrep_command=['pgrep', '-af', 'gates.sh|mk/at-|go test'],
                pgrep_exit=p.returncode, pgrep_raw_sha256=hashlib.sha256(p.stdout.encode()).hexdigest())


def admission(name, quiet=False):
    start = time.monotonic()
    polls = []
    clear_since = None
    windows = 0
    while True:
        state = processes()
        now = time.monotonic()
        elapsed = now-start
        if not state['ready']:
            clear_since = None
            windows = 0
        elif clear_since is None:
            clear_since = now
        else:
            windows += 1
        ready = state['ready'] and (not quiet or (windows >= 3 and now-clear_since >= 180))
        polls.append(dict(elapsed_seconds=elapsed, consecutive_free_60s_windows=windows, **state))
        write('admission/' + name + '.json', dict(waited_seconds=elapsed, quiet_required=quiet,
              qualified=ready, required_windows=3 if quiet else 0, interval_seconds=60, polls=polls))
        print('ADMITTED' if ready else 'WAIT', name, round(elapsed,3), 'seconds',
              'free windows', windows, 'other runs', state['other_runs'], flush=True)
        if ready:
            return
        time.sleep(60)


def public(raw):
    cache = subprocess.check_output(['go', 'env', 'GOCACHE'], text=True).strip()
    return raw.decode(errors='replace').replace(str(ROOT), '<worktree>').replace(cache, '<go-cache>').replace(str(Path.home()), '<home>')


def run(name, args, quiet=False):
    assert not (OUT / (name + '.json')).exists(), 'never overwrite an attempt'
    admission(name, quiet)
    start = time.monotonic()
    monitors = []
    # Foreground child, with a mandatory observation every 60 seconds. An
    # observed overlap is retained, never hidden by retrying or touching peers.
    proc = subprocess.Popen(args, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    while True:
        try:
            raw, _ = proc.communicate(timeout=60)
            break
        except subprocess.TimeoutExpired:
            state = processes()
            monitors.append(dict(elapsed_seconds=time.monotonic()-start, **state))
            write('monitor/' + name + '.json', dict(checks=monitors))
            print('MONITOR', name, round(time.monotonic()-start,3), 'seconds',
                  'other runs', state['other_runs'], flush=True)
    monitors.append(dict(elapsed_seconds=time.monotonic()-start, **processes()))
    write('monitor/' + name + '.json', dict(checks=monitors))
    data = public(raw).encode()
    readable = ('\n'.join(s.rstrip() for s in data.decode().splitlines()).rstrip() + '\n').encode() if data else data
    (OUT / (name + '.txt')).write_bytes(readable)
    if readable != data:
        (OUT / (name + '.txt.raw.gz')).write_bytes(gzip.compress(data, mtime=0))
    receipt = dict(command=[a.replace(str(ROOT), '<worktree>').replace(str(Path.home()), '<home>') for a in args],
                   exit=proc.returncode, elapsed_seconds=time.monotonic()-start,
                   quiet_required=quiet, other_run_overlap_observed=any(r['other_runs'] for r in monitors),
                   public_output_sha256=hashlib.sha256(data).hexdigest())
    write(name + '.json', receipt)
    print(json.dumps(receipt), flush=True)
    print('\n'.join(data.decode().splitlines()[-22:]), flush=True)
    return proc.returncode


def seal():
    pins = []
    for p in sorted(OUT.rglob('*')):
        if not p.is_file() or p.name == 'retention-pins.json':
            continue
        raw = p.read_bytes()
        if p.suffix in ('.json', '.jsonl') and len(raw) > 1_000_000:
            p2 = Path(str(p) + '.gz')
            p2.write_bytes(gzip.compress(raw, mtime=0))
            p.unlink()
            p = p2
            raw = p.read_bytes()
        if p.suffix in ('.txt', '.err') and raw:
            tidy = ('\n'.join(line.rstrip() for line in raw.decode().splitlines()).rstrip()+'\n').encode()
            if tidy != raw:
                backup = Path(str(p)+'.raw.gz')
                if not backup.exists():
                    backup.write_bytes(gzip.compress(raw, mtime=0))
                p.write_bytes(tidy)
                raw = tidy
        row = dict(path=p.relative_to(OUT).as_posix(), bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest())
        if p.suffix == '.gz':
            b = gzip.decompress(raw)
            row.update(uncompressed_bytes=len(b), uncompressed_sha256=hashlib.sha256(b).hexdigest())
        pins.append(row)
    known = {r['path'] for r in pins}
    for p in sorted(OUT.rglob('*.gz')):
        if p.relative_to(OUT).as_posix() not in known:
            raw = p.read_bytes()
            data = gzip.decompress(raw)
            pins.append(dict(path=p.relative_to(OUT).as_posix(), bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest(),
                             uncompressed_bytes=len(data), uncompressed_sha256=hashlib.sha256(data).hexdigest()))
    write('retention-pins.json', dict(test_only=True, files=pins))
    print('sealed', len(pins), 'files')


if __name__ == '__main__':
    if sys.argv[1] in ('gate', 'quiet-gate'):
        sys.exit(run(sys.argv[2], sys.argv[3:], quiet=sys.argv[1] == 'quiet-gate'))
    elif sys.argv[1] == 'plan':
        plan = json.loads(Path(sys.argv[2]).read_text())
        os.environ.update(plan.get('environment', {}))
        sys.exit(run(plan['name'], plan['command'], quiet=plan.get('quiet', False)))
    elif sys.argv[1] == 'admit':
        admission(sys.argv[2])
    elif sys.argv[1] == 'seal':
        seal()
