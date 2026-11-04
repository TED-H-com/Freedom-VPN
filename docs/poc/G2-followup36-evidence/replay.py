"""Offline D-161 replay. Run through tests/netlab/scripts/disktmp.py.

Original records are read-only. Rebuild the checked UDP projection in disk
scratch, including live1's blank-separator fix; retain new verdicts separately.
"""
import gzip
import hashlib
import json
from pathlib import Path
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'tests/netlab/scripts'))
from reload_analysis import analyze, acceptance_evidence, verdict_table
from trace_split import udp_projection

OUT = Path(__file__).resolve().parent
SCENARIO = 'TestAT10_AT13_ReloadRevokeBaseline'
SOURCES = {name: ROOT / 'docs/poc/G2-followup35-evidence/runs' / name / SCENARIO
           for name in ['live1', 'live2', 'live3', 'live4', 'live5', 'gate2']}
SOURCES.update({
    'F22-01': ROOT / 'docs/e2e/at/isolation/history/follow-up-22-run/reload-stream' / SCENARIO,
    'F27-01-B1': ROOT / 'docs/poc/G2-followup27-evidence/comparison/B1/reload' / SCENARIO,
})
pins, summary = [], {}
for label, source in SOURCES.items():
    with tempfile.TemporaryDirectory(prefix='d161-replay-') as tmp:
        scratch = Path(tmp)
        for f in sorted(source.iterdir()):
            if not f.name.startswith(('at10-', 'stream-a1', 'd120-verdict')):
                continue
            data = f.read_bytes()
            pins.append(dict(path=str(f.relative_to(ROOT)), sha256=hashlib.sha256(data).hexdigest(), bytes=len(data)))
            if f.name.endswith('.raw.gz'):
                continue
            (scratch / f.name.removesuffix('.gz')).write_bytes(gzip.decompress(data) if f.suffix == '.gz' else data)
        combined = (scratch / 'at10-origin-headers.txt').exists()
        if combined:
            for role in ('origin', 'node', 'responder'):
                name = 'at10-' + role
                udp, stats, meta = udp_projection((scratch / (name+'-headers.txt')).read_text(),
                                                   (scratch / (name+'-headers.err')).read_text())
                (scratch / (name+'.txt')).write_text(udp)
                (scratch / (name+'.err')).write_text(stats)
        original = json.loads((scratch / 'at10-analysis.json').read_text())
        original_verdict = original['d120']['verdict']
        result = analyze(scratch/'stream-a1.log.jsonl', scratch/'at10-apply.jsonl', scratch/'at10')
        result['d120'] = v = acceptance_evidence(result, (scratch/'stream-a1.txt').read_text())
        dest = OUT / 'replays' / label
        dest.mkdir(parents=True, exist_ok=True)
        (dest/'at10-analysis.json').write_text(json.dumps(result, indent=2)+'\n')
        (dest/'d120-verdict.txt').write_text(verdict_table(v))
        summary[label] = dict(source=str(source.relative_to(ROOT)), original_verdict=original_verdict,
                             replay_verdict=v['verdict'], combined_captures=combined,
                             **{k: v[k] for k in ('tcp_reads', 'tcp_timeouts', 'tcp_guard_losses',
                                'guard_attributed_timeouts', 'unattributed_timeouts', 'longest_tcp_stall',
                                'udp_losses', 'total_losses', 'errors')})
        if label != 'live1':
            assert v['verdict'] == original_verdict, label
        if label.startswith('F'):
            assert not v['passed'] and any('missing TCP capture' in e and '(iv)' in e for e in v['errors'])
        print(label, original_verdict, '=>', v['verdict'],
              'reads', v['tcp_reads'], 'timeouts', v['tcp_timeouts'],
              'guard transmissions', len(v['tcp_guard_losses']), 'longest', v['longest_tcp_stall'])
for pin in pins:
    assert hashlib.sha256((ROOT / pin['path']).read_bytes()).hexdigest() == pin['sha256']
(OUT/'replay-sources.json').write_text(json.dumps(pins, indent=2)+'\n')
(OUT/'replay-summary.json').write_text(json.dumps(summary, indent=2)+'\n')
