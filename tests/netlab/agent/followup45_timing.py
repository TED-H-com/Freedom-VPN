#!/usr/bin/env -S python3 -B
"""TEST ONLY: declared, reserved f45 A,B,A,B runner. No host network commands.

prepare revalidates and copies the frozen f43 archives and overlays. run admits at most two reserved
windows. Every command, refusal, sample and failed leg remains under .agent/f45.
The driver must be launched from the lane worktree, in the foreground.
"""
import datetime
import hashlib
import json
import os
from pathlib import Path
import queue
import re
import shutil
import signal
import subprocess
import sys
import threading
import time

PREFIX = 'fvpn-w2-agent-core-f45'
PATTERN = 'gates.sh|mk/at-|go test|node-test-netlab|netlab-test'
SHARED = {'fvpn-pg'}  # Shared PostgreSQL is not another lane's test container.
INTERVAL = 4.0
ROOT = Path.cwd()
OUT = ROOT / '.agent/f45/timing'
TREES = ROOT / '.agent/tmp/f45-timing'


def utc():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def append(path, row):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('a') as stream:
        stream.write(json.dumps(row) + '\n')


def save(path, row):
    path.write_text(json.dumps(row, indent=2) + '\n')


def admission():
    result = subprocess.run(['bash', os.environ['FVPN_HOLD_SH']], capture_output=True, text=True)
    row = {'utc': utc(), 'exit': result.returncode, 'output': result.stdout.strip()}
    append(OUT / 'admission.jsonl', row)
    print(row['output'], file=sys.stderr, flush=True)
    return result.returncode == 0 and row['output'] == 'free'


def wrapped_tool():
    name, args = Path(sys.argv[0]).name, sys.argv[1:]
    real = shutil.which(name, path=os.environ['F45_ORIGINAL_PATH'])
    if not real:
        raise SystemExit('tool unavailable: ' + name)
    if name in {'make', 'docker'} or (name == 'go' and args[:1] == ['test']):
        if not admission():
            raise SystemExit(125)
    leg = Path(os.environ['F45_LEG'])
    if name == 'go':
        if args[:1] == ['test'] and 'netlab' in args:
            image = subprocess.check_output(['docker', 'inspect', '--format', '{{.Image}}', PREFIX], text=True).strip()
            save(leg / 'fixture-image.json', {'image_id': image})
            measure_clock(leg)
        result = subprocess.run([real, *args])
        if args[:1] == ['test'] and 'netlab' in args:
            with (leg / 'spans.jsonl').open('wb') as stream:
                captured = subprocess.run(['docker', 'exec', PREFIX, 'cat',
                    '/scratch/w2/operations.jsonl.f43-spans'], stdout=stream, stderr=subprocess.PIPE)
            save(leg / 'span-capture.json', {'exit': captured.returncode})
        if result.returncode == 0 and args[:1] == ['build'] and '-o' in args:
            binary = Path(args[args.index('-o') + 1])
            if binary.name in {'fvpn-agent', 'fvpn-watchdog', 'fvpn-labcontrol'}:
                target = leg / 'binaries.json'
                hashes = json.loads(target.read_text()) if target.exists() else {}
                hashes[binary.name] = hashlib.sha256(binary.read_bytes()).hexdigest()
                save(target, hashes)
        raise SystemExit(result.returncode)
    os.execv(real, [real, *args])


def parse_stat(raw):
    # comm can contain spaces and parentheses; fields start after its last ')'.
    left, right = raw.index('('), raw.rindex(')')
    fields = raw[right + 2:].split()
    return {'comm': raw[left + 1:right], 'ppid': int(fields[1]),
            'ticks': int(fields[11]) + int(fields[12]), 'start': int(fields[19])}


def proc_snapshot():
    result, unreadable = {}, []
    for path in Path('/proc').iterdir():
        if not path.name.isdigit():
            continue
        try:
            entry = parse_stat((path / 'stat').read_text())
            entry['cgroup'] = (path / 'cgroup').read_text()
            result[int(path.name)] = entry
        except (FileNotFoundError, ProcessLookupError):
            pass
        except (PermissionError, ValueError, OSError) as error:
            unreadable.append({'pid': int(path.name), 'error': type(error).__name__})
    return result, unreadable


def descendants(processes, root, remembered):
    own = {pid for pid, item in processes.items() if (pid, item['start']) in remembered}
    own.add(root)
    while True:
        added = {pid for pid, item in processes.items() if item['ppid'] in own} - own
        if not added:
            break
        own.update(added)
    remembered.update((pid, processes[pid]['start']) for pid in own if pid in processes)
    return own


def cpu_class(pid, entry, own, container_ids):
    if pid in own:
        return 'runner_descendants'
    if any(container_id in entry['cgroup'] for container_id in container_ids):
        return 'own_containers'
    if pid == 2 or entry['ppid'] == 2:
        return 'kernel_threads'
    comm = entry['comm']
    if comm in {'dockerd', 'containerd', 'docker-proxy', 'runc'} or comm.startswith('containerd-shim'):
        return 'container_runtime'
    return 'foreign'


def docker(*args):
    if not admission():
        raise RuntimeError('Docker read/cleanup not admitted')
    return subprocess.check_output(['docker', *args], text=True)


def running_containers():
    raw = docker('ps', '--no-trunc', '--format', '{{.ID}} {{.Names}}')
    return [dict(zip(('id', 'name'), line.split(maxsplit=1))) for line in raw.splitlines()]


class ContainerState:
    def __init__(self, initial):
        self.running = {row['id']: row for row in initial}
        self.own_ids = {row['id'] for row in initial if row['name'].startswith(PREFIX)}
        self.foreign_seen = {}

    def update(self, event):
        actor = event.get('Actor', {})
        identity = actor.get('ID', event.get('id', ''))
        name = actor.get('Attributes', {}).get('name', '')
        action = event.get('Action', event.get('status', ''))
        if name.startswith(PREFIX):
            self.own_ids.add(identity)
        if action in {'start', 'unpause'} and identity and name:
            row = {'id': identity, 'name': name}
            self.running[identity] = row
            if name.startswith('fvpn-') and not name.startswith(PREFIX) and name not in SHARED:
                self.foreign_seen[identity] = row
        elif action in {'die', 'destroy'}:
            self.running.pop(identity, None)

    def snapshot(self):
        foreign = dict(self.foreign_seen)
        for identity, row in self.running.items():
            if row['name'].startswith('fvpn-') and not row['name'].startswith(PREFIX) and row['name'] not in SHARED:
                foreign[identity] = row
        self.foreign_seen.clear()
        return list(self.running.values()), list(foreign.values()), set(self.own_ids)


class ContainerEvents:
    """One admitted, joined foreground child; no Docker calls during the suite.

    The admission script regards sibling make processes as competitors. An
    event stream admitted before make preserves continuous inventory evidence
    without bypassing that check or changing the frozen product trees.
    """
    def __init__(self, label):
        since = str(int(time.time()) - 1)
        self.state = ContainerState(running_containers())
        if not admission():
            raise RuntimeError('Docker events not admitted')
        self.child = subprocess.Popen(['docker', 'events', '--since', since,
                                      '--filter', 'type=container', '--format', '{{json .}}'],
                                     stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        self.pending = queue.Queue()
        self.label = label
        self.reader = threading.Thread(target=self.read, daemon=False)
        self.reader.start()

    def read(self):
        for raw in self.child.stdout:
            try:
                event = json.loads(raw)
            except ValueError:
                self.pending.put(None)
                continue
            append(OUT / 'container-events.jsonl', {'utc': utc(), 'label': self.label, 'event': event})
            self.pending.put(event)

    def snapshot(self):
        if self.child.poll() is not None:
            raise RuntimeError('Docker event stream ended before suite')
        while True:
            try:
                event = self.pending.get_nowait()
            except queue.Empty:
                break
            if event is None:
                raise RuntimeError('unreadable Docker event')
            self.state.update(event)
        return self.state.snapshot()

    def close(self):
        self.child.terminate()
        try:
            self.child.wait(timeout=5)
        except subprocess.TimeoutExpired:
            self.child.kill()
            self.child.wait()
        self.reader.join()
        self.child.stdout.close()
        self.child.stderr.close()


class Monitor:
    def __init__(self, label):
        self.label = label
        self.events = ContainerEvents(label)
        self.remembered = set()
        self.previous, self.unreadable = proc_snapshot()
        self.own = descendants(self.previous, os.getpid(), self.remembered)
        self.at_ns = mono_ns()
        self.at = self.at_ns / 1e9
        self.hz = os.sysconf('SC_CLK_TCK')
        self.container_ids = set()

    def sample(self):
        containers, foreign_containers, own_ids = self.events.snapshot()
        self.container_ids.update(own_ids)
        current, unreadable = proc_snapshot()
        now_ns = mono_ns()
        now = now_ns / 1e9
        own = descendants(current, os.getpid(), self.remembered)
        seconds = now - self.at
        totals = {key: 0 for key in ['foreign', 'runner_descendants', 'own_containers', 'kernel_threads', 'container_runtime']}
        contributors = []
        for pid, item in current.items():
            old = self.previous.get(pid)
            delta = max(0, item['ticks'] - old['ticks']) if old and old['start'] == item['start'] else item['ticks']
            kind = cpu_class(pid, item, own, self.container_ids)
            totals[kind] += delta
            if kind == 'foreign' and delta:
                contributors.append({'pid': pid, 'comm': item['comm'], 'ticks': delta})
        check = subprocess.run(['pgrep', '-af', PATTERN], text=True, capture_output=True)
        if check.returncode not in (0, 1):
            raise RuntimeError('unreadable suite process inventory')
        raw = check.stdout
        foreign_suites = [line for line in raw.splitlines() if int(line.split(maxsplit=1)[0]) not in own]
        vmstat = subprocess.check_output(['vmstat'], text=True)
        cores = {key: ticks / self.hz / seconds for key, ticks in totals.items()}
        row = {'utc': utc(), 'clock': 'CLOCK_MONOTONIC', 'mono_ns': str(now_ns),
               'interval_start_ns': str(self.at_ns), 'interval_end_ns': str(now_ns),
               'label': self.label, 'seconds': seconds, 'cpu_ticks': totals,
               'cpu_cores': cores, 'foreign_cpu_processes': contributors,
               'foreign_suites': foreign_suites, 'running_containers': containers,
               'foreign_containers': foreign_containers, 'vmstat': vmstat,
               'unreadable': self.unreadable + unreadable,
               'departed_pids': sorted(set(self.previous) - set(current))}
        qualify(row)
        append(OUT / 'samples.jsonl', row)
        self.previous, self.at, self.own, self.unreadable = current, now, own, unreadable
        self.at_ns = now_ns
        return row


def mono_ns():
    return time.clock_gettime_ns(time.CLOCK_MONOTONIC)


def qualify(row):
    row['marked'] = row['cpu_cores']['foreign'] > 0.5
    row['top_foreign_processes'] = sorted(row['foreign_cpu_processes'],
                                         key=lambda p: (-p['ticks'], p['pid']))[:5]
    row['qualified'] = (not row['foreign_suites'] and not row['foreign_containers']
                        and not row['unreadable'] and 0 < row['seconds'] <= 5.0)


def clock_matches(row):
    probe = row['probe']
    # Equal clock values are required; namespace identities may differ.
    # Even an identical namespace must report zero offsets on both sides.
    return (row['host_timens_offsets'].split() == ['monotonic', '0', '0', 'boottime', '0', '0']
            and probe['timens_offsets'].split() == ['monotonic', '0', '0', 'boottime', '0', '0']
            and int(row['host_before_ns']) <= int(probe['before_ns']) <= int(probe['mono_ns'])
                <= int(probe['after_ns']) <= int(row['host_after_ns']))


def measure_clock(leg):
    rows = []
    for _ in range(3):
        host = os.readlink('/proc/self/ns/time')
        offsets = Path('/proc/self/timens_offsets').read_text()
        before = mono_ns()
        raw = subprocess.check_output(['docker', 'exec', PREFIX, '/work/.agent/f45-clock'], text=True)
        after = mono_ns()
        row = {'utc': utc(), 'host_before_ns': str(before), 'host_after_ns': str(after),
               'host_time_namespace': host, 'host_timens_offsets': offsets, 'probe': json.loads(raw)}
        row['matched'] = clock_matches(row)
        rows.append(row)
    matched = all(r['matched'] for r in rows)
    save(leg / 'clock.json', {'clock': 'CLOCK_MONOTONIC', 'matched': matched, 'measurements': rows})
    if not matched:
        raise RuntimeError('clock mismatch; stop timing item')


def file_hash(path, prefix=b'', algorithm='sha256'):
    digest = hashlib.new(algorithm)
    digest.update(prefix)
    with path.open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def prepare():
    old = ROOT / '.agent/f43/timing'
    declared = json.loads((ROOT / 'docs/node/followup43-timing-pins-2026-11-03.json').read_text())
    OUT.mkdir(parents=True, exist_ok=True)
    if (OUT / 'window.json').exists():
        raise RuntimeError('declaration already frozen')
    sources = declared['sources']
    for arm, source in sources.items():
        archive = old / (arm + '.tar')
        if file_hash(archive) != source['archive_sha256']:
            raise RuntimeError('f43 archive hash mismatch: rebuild and redeclare required')
        shutil.copyfile(archive, OUT / archive.name)
        target = TREES / arm
        target.mkdir(parents=True, exist_ok=False)
        subprocess.run(['tar', '-xf', str(OUT / archive.name), '-C', str(target)], check=True)
        (target / '.agent/tmp').mkdir(parents=True, exist_ok=True)
        dest = OUT / 'overlays' / arm
        dest.mkdir(parents=True)
        for name in ['engine.go', 'followup43_spans.go']:
            original = old / 'overlays' / arm / name
            if file_hash(original) != declared['overlays'][arm][name]:
                raise RuntimeError('f43 overlay hash mismatch: rebuild and redeclare required')
            shutil.copyfile(original, dest / name)
        save(dest / 'overlay.json', {'Replace': {
            str(target / 'node/agent/engine.go'): str(dest / 'engine.go'),
            str(target / 'node/agent/followup43_spans.go'): str(dest / 'followup43_spans.go')}})
    save(OUT / 'sources.json', sources)
    save(OUT / 'overlays/pins.json', declared['overlays'])
    verify_inputs()
    save(OUT / 'window.json', {'created': utc(), **{a: s['commit'] for a, s in sources.items()},
         'windows': {'1': ['A1', 'B1', 'A2', 'B2'], '2': ['A3', 'B3', 'A4', 'B4']},
         'prefix': PREFIX, 'interval_seconds': INTERVAL, 'reservation_minutes': 30,
         'maximum_reservations': 2, 'minimum_retained_whole_step_fraction': 0.5,
         'rule': 'foreign CPU >0.5 marks interval; other qualification unchanged'})


def cleanup_inventory():
    for name in docker('ps', '-a', '--format', '{{.Names}}').splitlines():
        if name.startswith(PREFIX):
            docker('rm', '-fv', name)
    for tag in docker('images', '--format', '{{.Repository}}:{{.Tag}}').splitlines():
        if tag.startswith(PREFIX):
            docker('rmi', tag)
    for kind in ['network', 'volume']:
        for name in docker(kind, 'ls', '--format', '{{.Name}}').splitlines():
            if name.startswith(PREFIX):
                docker(kind, 'rm', name)
    return inventory()


def inventory():
    commands = {'containers': ['ps', '-a', '--format', '{{.Names}}'],
                'images': ['images', '--format', '{{.Repository}}:{{.Tag}}'],
                'networks': ['network', 'ls', '--format', '{{.Name}}'],
                'volumes': ['volume', 'ls', '--format', '{{.Name}}']}
    return {kind: [line for line in docker(*args).splitlines() if line.startswith(PREFIX)]
            for kind, args in commands.items()}


def run_leg(label, window, window_number, deadline):
    arm = label[0]
    evidence = OUT / ('window' + str(window_number)) / label
    evidence.mkdir(parents=True, exist_ok=False)
    def refused(reason, code=None):
        row = {'leg': label, 'arm': arm, 'source': window[arm], 'window': window_number,
               'qualified': False, 'reason': reason, 'exit': code, 'utc': utc(),
               'evidence': str(evidence.relative_to(ROOT))}
        save(evidence / 'result.json', row)
        print('REFUSED ' + json.dumps(row), flush=True)
        return row
    if not admission():
        return refused('entry admission refused', 125)
    try:
        monitor = Monitor(label)
    except (RuntimeError, subprocess.CalledProcessError, OSError) as error:
        return refused(str(error))
    time.sleep(INTERVAL)
    try:
        preflight = monitor.sample()
    except (RuntimeError, subprocess.CalledProcessError, OSError) as error:
        monitor.events.close()
        return refused(str(error))
    if not preflight['qualified']:
        monitor.events.close()
        return refused('preflight failed')
    bindir = OUT / 'bin'
    bindir.mkdir(exist_ok=True)
    for name in ['go', 'make', 'docker']:
        link = bindir / name
        if not link.exists():
            link.symlink_to(Path(__file__).resolve())
    tree = TREES / arm
    env = dict(os.environ, F45_ORIGINAL_PATH=os.environ['PATH'], F45_LEG=str(evidence),
               CGO_ENABLED='0', GOTOOLCHAIN='auto', TMPDIR=str(tree / '.agent/tmp'), GOTMPDIR=str(tree / '.agent/tmp'),
               PYTHONDONTWRITEBYTECODE='1', GOMAXPROCS='4',
               GOFLAGS='-p=4 -buildvcs=false -overlay=' + str(OUT / 'overlays' / arm / 'overlay.json'), FVPN_AGENT_EVIDENCE_DIR=str(evidence),
               F45_TIMING_ROOT=str(ROOT))
    env['GOCACHE'] = subprocess.check_output(['go', 'env', 'GOCACHE'], text=True).strip()
    env['PATH'] = str(bindir) + os.pathsep + env['PATH']
    cmd = ['make', '-f', 'mk/node.mk', 'node-test-netlab', 'FVPN_LAB_PREFIX=' + PREFIX,
           'MAKE=' + str(bindir / 'make')]
    started, begin = utc(), time.monotonic()
    print('START ' + label + ' ' + started, flush=True)
    qualified, reason, samples = True, '', 0
    try:
        with (evidence / 'suite.log').open('w') as stream:
            child = subprocess.Popen(cmd, cwd=tree, env=env, stdout=stream, stderr=subprocess.STDOUT, start_new_session=True)
            while child.poll() is None:
                try:
                    child.wait(timeout=max(0, INTERVAL - (time.monotonic() - monitor.at)))
                except subprocess.TimeoutExpired:
                    pass
                try:
                    row = monitor.sample()
                    samples += 1
                    qualified = row['qualified']
                    if not qualified:
                        reason = 'qualification breach (suite, container, unreadable state or interval); see samples'
                except (RuntimeError, subprocess.CalledProcessError, OSError) as error:
                    qualified, reason = False, str(error)
                if time.monotonic() >= deadline - 30:
                    qualified, reason = False, 'reservation ending; stop before protected window expires'
                if not qualified:
                    if child.poll() is None:
                        os.killpg(child.pid, signal.SIGTERM)
                    try:
                        child.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        os.killpg(child.pid, signal.SIGKILL)
                        child.wait()
                    break
            code = child.wait()
    finally:
        monitor.events.close()
    row = {'leg': label, 'arm': arm, 'source': window[arm], 'window': window_number, 'started': started,
           'completed': utc(), 'seconds': time.monotonic() - begin, 'exit': code, 'qualified': qualified,
           'reason': reason, 'samples': samples, 'evidence': str(evidence.relative_to(ROOT)),
           'timing_runner_sha256': file_hash(Path(__file__))}
    text = (evidence / 'suite.log').read_text()
    row['passed_scenarios'] = re.findall(r'^--- PASS: (Test\S+)', text, re.M)
    row['failed_scenarios'] = re.findall(r'^--- FAIL: (Test\S+)', text, re.M)
    row['d087_d104_failed'] = bool(re.search(r'--- FAIL: TestLabD087_D104_', text))
    retain_interrupted(tree, evidence)
    row['inventory'] = cleanup_inventory()
    if any(row['inventory'].values()):
        row['qualified'], row['reason'] = False, 'owned inventory not empty'
    import followup45_results as results_tool
    marked = [s for s in results_tool.lines(OUT / 'samples.jsonl') if s['label'] == label and s['marked']]
    measurement = results_tool.evidence(evidence, marked)
    row.update(results_tool.pool_decision(row, measurement))
    row['scenario_failures'] = results_tool.scenario_failures(row, evidence, marked)
    row['stop_item'] = bool(row['qualified'] and row.get('exit') and (not row['failed_scenarios'] or
        any(not f['marked'] for f in row['scenario_failures'])))
    if (evidence / 'clock.json').exists() and not results_tool.read(evidence / 'clock.json')['matched']:
        row['stop_item'] = True
    save(evidence / 'result.json', row)
    print('END ' + json.dumps(row), flush=True)
    return row


def reserve(number):
    begin, deadline = time.monotonic(), time.monotonic() + 5400
    while time.monotonic() < deadline:
        result = subprocess.run(['bash', os.environ['FVPN_HOLD_SH'], '--reserve', '30', 'agent-core f45 D-150'], text=True, capture_output=True)
        row = {'window': number, 'utc': utc(), 'waited_seconds': time.monotonic() - begin,
               'exit': result.returncode, 'stdout': result.stdout, 'stderr': result.stderr}
        append(OUT / 'reservations.jsonl', row)
        print('RESERVE ' + json.dumps(row), flush=True)
        if result.returncode == 0:
            return time.monotonic()
        time.sleep(30)
    return None


def verify_inputs():
    sources = json.loads((OUT / 'sources.json').read_text())
    for arm, source in sources.items():
        if file_hash(OUT / (arm + '.tar')) != source['archive_sha256']:
            raise RuntimeError('archive changed: ' + arm)
        for entry in subprocess.check_output(['git', 'ls-tree', '-r', '-z', source['commit']]).split(b'\0'):
            if not entry:
                continue
            meta, name = entry.split(b'\t', 1)
            mode, kind, expected = meta.split()
            path = TREES / arm / os.fsdecode(name)
            data = os.fsencode(os.readlink(path)) if mode == b'120000' else path.read_bytes()
            actual = hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest()
            if actual != expected.decode():
                raise RuntimeError('frozen blob changed: ' + arm + '/' + os.fsdecode(name))
    pins = json.loads((OUT / 'overlays/pins.json').read_text())
    for arm, files in pins.items():
        for name in ['engine.go', 'followup43_spans.go']:
            if file_hash(OUT / 'overlays' / arm / name) != files[name]:
                raise RuntimeError('overlay changed: ' + arm + '/' + name)
    append(OUT / 'input-verification.jsonl', {'utc': utc(), 'all_blobs_match': True,
           'archives_match': True, 'overlays_match': True})


def run():
    verify_inputs()
    declared = json.loads((ROOT / 'docs/node/followup45-timing-pins-2026-11-03.json').read_text())
    for name, expected in declared['files'].items():
        if file_hash(ROOT / name) != expected:
            raise RuntimeError('declared tool changed: ' + name)
    for arm, expected in declared['clock_binaries'].items():
        if file_hash(TREES / arm / '.agent/f45-clock') != expected:
            raise RuntimeError('clock probe changed: ' + arm)
    window = json.loads((OUT / 'window.json').read_text())
    results = json.loads((OUT / 'results.json').read_text()) if (OUT / 'results.json').exists() else []
    releases = [json.loads(line) for line in (OUT / 'releases.jsonl').read_text().splitlines()] if (OUT / 'releases.jsonl').exists() else []
    attempts = [json.loads(line) for line in (OUT / 'windows.jsonl').read_text().splitlines()] if (OUT / 'windows.jsonl').exists() else []
    number = len(attempts) + 1
    if number > 2 or any(r.get('stop_item') for r in results):
        raise RuntimeError('timing item stopped or both windows exhausted')
    if number == 2 and releases:
        elapsed = time.time() - datetime.datetime.fromisoformat(releases[-1]['utc']).timestamp()
        remaining = max(0, 600 - elapsed)
        print('Waiting ten minutes after release before second reservation', flush=True)
        while remaining > 0:
            pause = min(30, remaining)
            time.sleep(pause)
            remaining -= pause
    append(OUT / 'windows.jsonl', {'window': number, 'utc': utc()})
    reserved = reserve(number)
    stop = ''
    if reserved is None:
        stop = '90-minute reservation admission wait expired'
    else:
        try:
            print('Reserved; waiting 60 seconds before preflight', flush=True)
            time.sleep(60)
            for label in window['windows'][str(number)]:
                if time.monotonic() - reserved > 30 * 60 - 420:
                    stop = 'insufficient reservation time for another full leg'
                    break
                row = run_leg(label, window, number, reserved + 30 * 60)
                results.append(row)
                from followup45_results import repeated_b_failures
                repeated = repeated_b_failures(results)
                if repeated:
                    row['stop_item'], row['repeated_b_failures'] = True, repeated
                save(OUT / 'results.json', results)
                if row.get('stop_item'):
                    stop = 'suite failure outside marked operation, clock mismatch, or repeated pooled B failure'
                    break
                if not row['qualified']:
                    stop = 'unqualified leg; no retry in this reservation'
                    break
        finally:
            result = subprocess.run(['bash', os.environ['FVPN_HOLD_SH'], '--release'], capture_output=True, text=True)
            append(OUT / 'releases.jsonl', {'utc': utc(), 'window': number, 'exit': result.returncode,
                   'stdout': result.stdout, 'stderr': result.stderr})
            print('RELEASE ' + result.stdout.strip(), flush=True)
    completion = {'utc': utc(), 'stop': stop, 'windows_attempted': number,
                  'pooled_legs': [r['leg'] for r in results if r.get('pooled')],
                  'production_gate': any(r['arm'] == 'B' and r.get('pooled') and r['exit'] == 0
                     and len(r['passed_scenarios']) == 13 for r in results)}
    save(OUT / 'completion.json', completion)
    print(json.dumps(completion), flush=True)


def retain_interrupted(tree, evidence):
    # Retain interrupted writer streams before any later leg wipes its scratch.
    if (evidence / 'suite/operations.jsonl').exists():
        return
    source = tree / '.agent/tmp' / (PREFIX + '-scratch') / 'w2'
    if not source.exists():
        return
    dest = evidence / 'interrupted'
    dest.mkdir(exist_ok=True)
    for name in ['operations.jsonl', 'operations.jsonl.f43-spans', 'renewal.json',
                 'renewal.json.client.json', 'expiry.json', 'expiry.json.restart.json',
                 'agent.log', 'watchdog.log']:
        # The regular files may be root-readable only: an owned container reads
        # only its own scratch, before cleanup; no host permissions are changed.
        result = subprocess.run(['bash', os.environ['FVPN_HOLD_SH']], capture_output=True, text=True)
        append(OUT / 'admission.jsonl', {'utc': utc(), 'exit': result.returncode, 'output': result.stdout.strip()})
        if result.returncode or result.stdout.strip() != 'free':
            raise RuntimeError('interrupted evidence capture not admitted')
        got = subprocess.run(['docker', 'exec', PREFIX, 'cat', '/scratch/w2/' + name], capture_output=True)
        if got.returncode == 0:
            (dest / name).write_bytes(got.stdout)
    save(dest / 'capture.json', {'files': {p.name: file_hash(p) for p in dest.iterdir() if p.is_file()}})


if __name__ == '__main__':
    if Path(sys.argv[0]).name in {'go', 'make', 'docker'}:
        ROOT = Path(os.environ['F45_TIMING_ROOT'])
        OUT = ROOT / '.agent/f45/timing'
        wrapped_tool()
    elif sys.argv[1:2] == ['prepare']:
        prepare()
    elif sys.argv[1:] == ['run']:
        run()
    else:
        raise SystemExit('usage: followup45_timing.py prepare | run')
