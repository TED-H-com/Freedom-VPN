"""Targeted, deterministic historical extraction; no synthetic retry is an observation."""
import gzip
import hashlib
import json
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parents[3]
out, pins = {}, []
for label, path, historical in [
    ('F22-01', 'docs/e2e/at/isolation/reload-stream/TestAT10_AT13_ReloadRevokeBaseline', True),
    ('F27-01-B1', 'docs/poc/G2-followup27-evidence/comparison/B1/reload/TestAT10_AT13_ReloadRevokeBaseline', False),
]:
    def read(name):
        data = (subprocess.check_output(['git', 'show', '4e77e816:' + path + '/' + name], cwd=root)
                if historical else (root / path / name).read_bytes())
        pins.append(dict(path=path + '/' + name, commit='4e77e816' if historical else '870a931a',
                         sha256=hashlib.sha256(data).hexdigest()))
        return gzip.decompress(data) if name.endswith('.gz') else data
    events = [json.loads(line) for line in read('stream-a1.log.jsonl').splitlines()]
    bad = next(e for e in events if e['event'] == 'tcp_read' and e.get('outcome'))
    send = next(e for e in events if e['sequence'] == bad['sequence'] and e['event'] == 'tcp_send')
    t = int(send['wall_ns'])
    windows = []
    for a in map(json.loads, read('at10-apply.jsonl.gz').splitlines()):
        b = {b['kind']: b for b in a['batches']}
        times = [int(x['wall_ns']) for x in [b['guard_install']['start'], b['guard_install']['end'],
                                            b['guard_release']['start'], b['guard_release']['end']]]
        if t - 100_000_000 <= times[0] <= t + 1_800_000_000:
            windows.append(dict(generation=a['generation'], relative_ns=[n-t for n in times]))
    replay = []
    for offset in [0, 208154467, 416030325, 826029367, 832000000, 1664000000]:
        hits = []
        for w in windows:
            a, b, c, d = w['relative_ns']
            if a <= offset <= d:
                hits.append(dict(generation=w['generation'], phase='install' if offset <= b else 'certain' if offset < c else 'release'))
        replay.append(dict(offset_ns=offset, source='send' if offset == 0 else
                           'f30 cycle26 offset transplanted; not observed here' if offset < 800000000 else
                           'f28 cycle40 measured third retry offset transplanted; not observed here' if offset == 826029367 else
                           'hypothetical backoff; not measured by f30', hits=hits))
    counts = {}
    for role in ['origin', 'node', 'responder']:
        data = read('at10-' + role + '.txt' + ('.gz' if historical else ''))
        counts[role] = dict(tcp_header_lines=data.count(b'proto TCP'), udp_header_lines=data.count(b'proto UDP'))
    out[label] = dict(send=send, timeout=bad, elapsed_ns=int(bad['mono_ns'])-int(send['mono_ns']),
                      windows=windows, replay=replay, capture_counts=counts,
                      nearby_tcp=[e for e in events if e['event'].startswith('tcp') and bad['sequence']-2 <= e['sequence'] <= bad['sequence']+2])
out['sources'] = pins
out['interpretation'] = 'Timing compatibility only. No historical TCP packets, retry schedule after +416ms, or echo token identities were retained. Agent has no positive enforced Apply floor; campaign spacing unchanged.'
Path(__file__).with_name('historical-extraction.json').write_text(json.dumps(out, indent=2) + '\n')
