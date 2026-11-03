"""TEST ONLY: established UDP across configured denial and same-boot restart.

Runs exclusively inside the task's network lab. No host network operations.
The ordinary agent lifecycle, nft baseline and full Apply are unmodified.
"""
import json
from pathlib import Path
import signal
import socket
import subprocess
import sys
import time
import urllib.request

ROOT = Path('/scratch/w2')
DST = '203.0.113.11'


def client():
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    s.connect((DST, 9000))
    s.settimeout(.06)
    with (ROOT / 'f36-flow.jsonl').open('w', buffering=1) as log:
        for seq in range(2000):
            token = ('TEST ONLY %d' % seq).encode()
            sent = time.monotonic_ns()
            s.send(token)
            ok = False
            try:
                ok = s.recv(512) == token
            except socket.timeout:
                pass
            log.write(json.dumps(dict(seq=seq, sent_ns=str(sent), end_ns=str(time.monotonic_ns()),
                                      ok=ok, source_port=s.getsockname()[1])) + '\n')
            time.sleep(.02)


def admin(path):
    req = urllib.request.Request('http://127.0.0.1:9091/' + path, data=b'', method='POST')
    urllib.request.urlopen(req, timeout=5).read()


def rows():
    p = ROOT / 'f36-flow.jsonl'
    return [json.loads(line) for line in p.read_text().split('\n')[:-1]] if p.exists() else []


def wait_for(predicate, message, seconds=12):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        if predicate():
            return
        time.sleep(.02)
    raise AssertionError(message)


def nft():
    return subprocess.check_output(['nft', 'list', 'table', 'inet', 'fvpn'], text=True)


def ct():
    return subprocess.check_output(['conntrack', '-L', '-p', 'udp', '--orig-dst', DST],
                                   text=True, stderr=subprocess.DEVNULL)


def run():
    assert Path('/.dockerenv').exists(), 'lab container required'
    cfg_path = ROOT / 'agent.json'
    original = cfg_path.read_text()
    cfg = json.loads(original)
    proc = flow = None
    result = {'test_only': True}
    subprocess.run(['ip', '-n', 'fv-inet', 'addr', 'add', DST + '/32', 'dev', 'lo'], check=True)
    try:
        flow = subprocess.Popen(['ip', 'netns', 'exec', 'fv-a1', sys.executable, __file__, 'client'])
        wait_for(lambda: sum(r['ok'] for r in rows()) >= 5, 'flow not established')
        # Linux promotes a replied UDP tuple to ASSURED after its stream
        # threshold; a handful of immediate echoes need not carry it yet.
        wait_for(lambda: '[ASSURED]' in ct(), 'UDP tuple did not become ASSURED', seconds=6)
        result['conntrack_before'] = ct()
        assert '[ASSURED]' in result['conntrack_before'], 'UDP reply never established conntrack'
        subprocess.run([sys.executable, '/work/tests/netlab/agent/timing-probes.py', 'stop',
                        str(ROOT / 'f36-stop.json'), '15'], check=True)
        # A replacement with the old config would adopt these still-live rules.
        result['retained_rules'] = nft()
        assert 'fvpn_baseline=1' not in result['retained_rules']
        admin('hold')
        cfg['extra_denied_v4'] = cfg.get('extra_denied_v4', []) + [DST + '/32']
        cfg.update(heartbeat_seconds=1, poll_seconds=1, watchdog_stale_seconds=6)
        cfg['test_timing_path'] = str(ROOT / 'f36-operations.jsonl')
        cfg_path.write_text(json.dumps(cfg))
        result['restart_ns'] = str(time.monotonic_ns())
        with (ROOT / 'agent.log').open('a') as log:
            proc = subprocess.Popen(['fvpn-agent', 'run', '-config', str(cfg_path)], stdout=log, stderr=log)
        (ROOT / 'agent.pid').write_text(str(proc.pid))
        wait_for(lambda: 'fvpn_baseline=1' in nft(), 'restart adopted changed sets')
        result['baseline_observed_ns'] = str(time.monotonic_ns())
        result['baseline_rules'] = nft()
        # Two control-pass budgets. No successful Apply is available yet.
        time.sleep(4.4)
        result['blocked_rules'] = nft()
        assert 'fvpn_baseline=1' in result['blocked_rules'], 'delayed Apply left baseline'
        assert 'serving 0' in (ROOT / 'heartbeat').read_text()
        result['release_ns'] = str(time.monotonic_ns())
        admin('release')
        wait_for(lambda: 'fvpn_baseline=1' not in nft() and DST in nft(), 'first Apply missing new deny')
        wait_for(lambda: 'serving 1' in (ROOT / 'heartbeat').read_text(), 'first Apply did not serve')
        result['applied_ns'] = str(time.monotonic_ns())
        result['applied_rules'] = nft()
        time.sleep(1)
        result['conntrack_after'] = ct()
        assert '[ASSURED]' in result['conntrack_after'], 'test lost the established tuple'
        data = rows()
        result['flow'] = data
        assert len({r['source_port'] for r in data}) == 1, 'flow changed tuple'
        result['operations'] = [json.loads(x) for x in (ROOT / 'f36-operations.jsonl').read_text().splitlines()]
        boots = [r['baseline'] for r in result['operations']
                 if r.get('baseline', {}).get('cause') == 'boot_unverified']
        assert len(boots) == 1 and boots[0]['succeeded'], 'missing successful Boot baseline'
        applies = [r for r in result['operations'] if r['source'] == 'apply']
        assert applies, 'no full Apply recorded'
        first_apply = int(applies[0]['anchor']['mono_before_ns'])
        assert first_apply >= int(result['release_ns']), 'Apply escaped held renewal'
        assert all(not r['heartbeat']['serving'] for r in result['operations']
                   if r['source'] == 'heartbeat' and int(r['heartbeat']['mono_ns']) < first_apply), \
            'serving heartbeat preceded first Apply'
        boundary = int(boots[0]['end_ns'])
        result['baseline_completed_ns'] = str(boundary)
        after = [r for r in data if int(r['sent_ns']) >= boundary]
        applied = [r for r in after if int(r['sent_ns']) >= int(result['applied_ns'])]
        assert len(after) >= 15 and len(applied) >= 5, 'insufficient denied flow probes'
        assert not any(r['ok'] for r in after), 'established flow escaped baseline or Apply'
        result['passed'] = True
        print('SUMMARY F36 AT13 established_restart=PASS baseline_probes=%d applied_probes=%d' %
              (len(after) - len(applied), len(applied)))
    finally:
        for child in [proc, flow]:
            if child is not None and child.poll() is None:
                child.send_signal(signal.SIGTERM)
                try:
                    child.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    child.kill()
                    child.wait()
                    result['cleanup_timeout'] = True
        admin('release')
        cfg_path.write_text(original)
        subprocess.run(['ip', '-n', 'fv-inet', 'addr', 'del', DST + '/32', 'dev', 'lo'], check=True)
        (ROOT / 'f36-result.json').write_text(json.dumps(result, indent=2) + '\n')
    assert not result.get('cleanup_timeout'), 'agent exceeded the unchanged 15-second stop budget'


if __name__ == '__main__':
    client() if sys.argv[1:] == ['client'] else run()
