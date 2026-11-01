#!/usr/bin/env python3
"""Foreground receipts for follow-up 25. Run from the worktree root via disktmp."""
import gzip
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path.cwd()
OUT = ROOT / 'docs/poc/G2-followup25-evidence'
PREFIX = 'fvpn-w1-netlab-f25'


def write(name, data):
    path = OUT / name
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2) + '\n')


def public(raw):
    return raw.decode(errors='replace').replace(str(ROOT), '<worktree>')


def log(name, raw):
    # Public path substitution precedes retention; whitespace-only presentation
    # changes have a lossless public raw.gz alongside the readable copy.
    data = public(raw).encode()
    readable = ('\n'.join(line.rstrip() for line in data.decode().splitlines()).rstrip() + '\n').encode()
    path = OUT / (name + '.txt')
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(readable)
    if data != readable:
        Path(str(path) + '.raw.gz').write_bytes(gzip.compress(data, mtime=0))


def run(name, args, env=None, live=False):
    assert not (OUT / (name + '.json')).exists(), 'never overwrite an attempt'
    start = time.monotonic()
    proc = subprocess.Popen(args, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, env=env)
    if live:
        # The make recipe owns startup and cleanup; this foreground parent
        # only observes readiness and suite load, never changes other objects.
        ready = time.monotonic() + 90
        while proc.poll() is None and time.monotonic() < ready:
            check = subprocess.run(['docker', 'inspect', '-f', '{{.State.Running}}', PREFIX], capture_output=True)
            if check.returncode == 0 and check.stdout.strip() == b'true':
                break
            time.sleep(1)
        write('host/poststart.json', dict(elapsed_seconds=time.monotonic()-start, processes=processes()))
    raw, _ = proc.communicate()
    log(name, raw)
    row = dict(command=args, exit=proc.returncode, elapsed_seconds=time.monotonic()-start,
               public_output_sha256=hashlib.sha256(public(raw).encode()).hexdigest())
    write(name + '.json', row)
    print(json.dumps(row), flush=True)
    print('\n'.join(public(raw).splitlines()[-18:]), flush=True)
    return proc.returncode


def processes():
    p = subprocess.run(['pgrep', '-af', 'gates.sh|mk/at-'], capture_output=True, text=True)
    assert p.returncode in (0, 1), p.returncode
    # Process arguments can contain private paths. Keep raw locally; publish
    # PID/kind and raw digest, as in the accepted follow-up 24 host receipts.
    private = ROOT / '.agent/tmp/f25-pgrep.txt'
    private.parent.mkdir(parents=True, exist_ok=True)
    with private.open('a') as f:
        f.write(p.stdout)
    rows = [dict(pid=int(line.split()[0]), kind='gates.sh' if 'gates.sh' in line else 'mk/at-')
            for line in p.stdout.splitlines()]
    return dict(command=['pgrep', '-af', 'gates.sh|mk/at-'], exit=p.returncode,
                matches=rows, raw_sha256=hashlib.sha256(p.stdout.encode()).hexdigest())


def admission():
    start = time.monotonic()
    polls = []
    while True:
        p = processes()
        polls.append(dict(elapsed_seconds=time.monotonic()-start, **p))
        if len(p['matches']) < 2:
            break
        print('Waiting: two or more matching suites', flush=True)
        time.sleep(30)
    write('host/admission.json', dict(waited_seconds=time.monotonic()-start, polls=polls))


def quiet():
    start = time.monotonic()
    count = 0
    while True:
        count += 1
        before = processes()
        load_before = os.getloadavg()
        p = subprocess.run(['vmstat', '5', '7'], capture_output=True, text=True, check=True)
        rows = []
        for line in p.stdout.splitlines():
            fields = line.split()
            if len(fields) >= 17 and fields[0].isdigit():
                rows.append(dict(r=int(fields[0]), id=int(fields[14])))
        # First vmstat CPU row is a since-boot aggregate, not a 5 s sample.
        samples = rows[1:]
        ready = len(samples) == 6 and all(r['r'] <= 3 and r['id'] >= 80 for r in samples)
        elapsed = time.monotonic()-start
        write(f'host/prestart-{count}.json', dict(command=['vmstat', '5', '7'], vmstat=p.stdout,
              load_before=load_before, load_after=os.getloadavg(), processes_before=before,
              processes_after=processes(), since_boot=rows[0], samples=samples,
              quiet=ready, elapsed_seconds=elapsed))
        print('HOST', count, 'quiet', ready, 'elapsed', round(elapsed, 3), 'samples', samples, flush=True)
        if ready or elapsed >= 3600:
            write('host/quiet-result.json', dict(quiet=ready, waited_seconds=elapsed, checks=count,
                                               limit_seconds=3600, host_is_not_pass_criterion=True))
            return


def pack():
    for path in sorted(OUT.rglob('*')):
        if not path.is_file() or path.name == 'retention-pins.json':
            continue
        if path.suffix in ('.json', '.jsonl') and path.stat().st_size > 1_000_000:
            raw = path.read_bytes()
            compressed = gzip.compress(raw, mtime=0)
            assert gzip.decompress(compressed) == raw
            Path(str(path) + '.gz').write_bytes(compressed)
            path.unlink()
        elif path.suffix in ('.txt', '.err'):
            raw = path.read_bytes()
            tidy = ('\n'.join(line.rstrip() for line in raw.decode().splitlines()).rstrip() + '\n').encode() if raw else raw
            if tidy != raw:
                backup = Path(str(path) + '.raw.gz')
                if not backup.exists():
                    backup.write_bytes(gzip.compress(raw, mtime=0))
                path.write_bytes(tidy)


def seal():
    pack()
    pins = []
    for path in sorted(OUT.rglob('*')):
        if not path.is_file() or path.name == 'retention-pins.json':
            continue
        raw = path.read_bytes()
        row = dict(path=path.relative_to(OUT).as_posix(), bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest())
        if path.suffix == '.gz':
            data = gzip.decompress(raw)
            row.update(uncompressed_bytes=len(data), uncompressed_sha256=hashlib.sha256(data).hexdigest())
        pins.append(row)
    write('retention-pins.json', dict(test_only=True, files=pins))
    print('sealed', len(pins), 'files', flush=True)


if __name__ == '__main__':
    action = sys.argv[1]
    if action == 'live':
        admission()
        quiet()
        # Recheck immediately before launch, retaining any new contention.
        # The earlier admission receipt is never overwritten.
        p = processes()
        write('host/immediate-prestart.json', p)
        while len(p['matches']) >= 2:
            time.sleep(30)
            p = processes()
            write(f'host/late-admission-{time.monotonic_ns()}.json', p)
        env = dict(os.environ, FVPN_NETLAB_EVIDENCE_DIR='docs/poc/G2-followup25-evidence/live-attempt1')
        sys.exit(run('gate2', ['make', '-f', 'mk/netlab.mk', 'netlab-test', 'FVPN_LAB_PREFIX=' + PREFIX], env, live=True))
    elif action == 'gate':
        sys.exit(run(sys.argv[2], sys.argv[3:]))
    elif action == 'seal':
        seal()
    else:
        raise ValueError(action)
