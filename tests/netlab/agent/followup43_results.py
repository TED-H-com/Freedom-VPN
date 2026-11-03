"""TEST ONLY offline f43 receipts; nearest-rank percentiles, no exclusion of failures."""
import hashlib
import json
import math
from pathlib import Path
import sys

METRICS = ['outer_guard', 'install_command', 'render_to_installed',
           'whole_serving_step', 'pre_apply_accounting']


def lines(path):
    return [json.loads(x) for x in path.read_text().splitlines()] if path.exists() else []


def read(path, default=None):
    return json.loads(path.read_text()) if path.exists() else default


def pin(path):
    return {'sha256': hashlib.sha256(path.read_bytes()).hexdigest(), 'bytes': path.stat().st_size}


def stats(values):
    values = sorted(values)
    return {'count': len(values), **{key: str(values[math.ceil(q * len(values)) - 1]) if values else None
            for key, q in [('p50_ns', .5), ('p90_ns', .9), ('p99_ns', .99), ('max_ns', 1)]}}


def evidence(path):
    streams = [path / sub / 'operations.jsonl' for sub in ['suite', 'assembled', 'interrupted']]
    rows = {}
    for stream in streams:
        for row in lines(stream):
            key = row['run_id'], row['sequence']
            if key in rows and row != rows[key]:
                raise ValueError('different observations for same run/sequence')
            rows[key] = row
    f36 = read(path / 'f36-result.json', {})
    for row in f36.get('operations', []):
        key = row['run_id'], row['sequence']
        if key in rows and row != rows[key]:
            raise ValueError('F36 observation collision')
        rows[key] = row
    values = {key: [] for key in METRICS}
    failures, guards, seen, gaps = [], [], {}, []
    for row in sorted(rows.values(), key=lambda r: (r['run_id'], int(r['sequence']))):
        identity = {key: row[key] for key in ['run_id', 'sequence', 'boot_id']}
        previous = seen.get(row['run_id'], 0)
        if int(row['sequence']) != previous + 1:
            gaps.append(dict(identity, previous=previous))
        seen[row['run_id']] = int(row['sequence'])
        timing = row.get('timing', {})
        if timing and not timing['succeeded']:
            failures.append(dict(identity, kind='operation', source=row['source'], timing={
                key: value for key, value in timing.items() if key not in ['commands', 'verifications']}))
        for index, command in enumerate(timing.get('commands', [])):
            if not command['ok']:
                failures.append(dict(identity, kind='command', index=index, command={
                    key: value for key, value in command.items() if key not in ['before', 'after']}))
        for source, flag in [('baseline', 'succeeded'), ('heartbeat', 'written')]:
            if source in row and not row[source][flag]:
                failures.append(dict(identity, kind=source, detail=row[source]))
        if timing.get('operation') != 'apply':
            continue
        d = timing['deadline']
        if not (d['install_ok'] and d['guard_released']):
            continue
        render, start, installed, release, end = [int(d[key]) for key in
            ['render_start_ns', 'install_start_ns', 'installed_by_ns', 'release_start_ns', 'release_end_ns']]
        assert 0 < render <= start <= installed <= release <= end
        values['outer_guard'].append(end - start)
        values['install_command'].append(installed - start)
        values['render_to_installed'].append(installed - render)
        guards.append(dict(identity, start=start, end=end))
    spans = lines(path / 'spans.jsonl') or lines(path / 'interrupted/operations.jsonl.f43-spans')
    for span in spans:
        start, end = int(span['start_ns']), int(span['end_ns'])
        assert 0 < start <= end
        if span['kind'] == 'whole_step':
            if span['serving_before'] or span['serving_after']:
                values['whole_serving_step'].append(end - start)
            if span['returned_error']:
                failures.append(dict(kind='step_return', detail=span))
        else:
            assert span['kind'] == 'pre_apply_accounting'
            # Failed/partial boundaries are kept, but have no Apply endpoint.
            if span['reached_apply']:
                values['pre_apply_accounting'].append(end - start)
            else:
                failures.append(dict(kind='pre_apply_aborted', detail=span))
    packets = {}
    expiry = read(path / 'assembled/expiry.json') or read(path / 'interrupted/expiry.json')
    if expiry:
        deadline = int(expiry['deadline_ns'])
        positives = [p for p in expiry['packets'] if int(p.get('egress_ns', 0))]
        drops = [p for p in expiry['packets'] if not int(p.get('egress_ns', 0))]
        last = max([int(p['egress_ns']) for p in positives], default=None)
        first = min([int(p['sent_ns']) for p in drops], default=None)
        packets['expiry'] = dict(positives=len(positives), drops=len(drops), deadline_ns=str(deadline),
            last_positive_offset_ns=str(last - deadline) if last else None,
            first_drop_offset_ns=str(first - deadline) if first else None,
            positives_at_or_after_deadline=sum(int(p['egress_ns']) >= deadline for p in positives),
            capture_stats=expiry['capture_stats'],
            predicate=bool(len(positives) >= 3 and len(drops) >= 3 and last >= deadline - 101000000
                           and first <= deadline + 100000000 and all(int(p['egress_ns']) < deadline
                           and int(p['sent_ns']) < first for p in positives)))
    renewal = read(path / 'assembled/renewal.json') or read(path / 'interrupted/renewal.json')
    if renewal:
        losses = [p for p in renewal['packets'] if p['result'] != 'ok']
        inside = sum(any(g['boot_id'] == renewal['boot_id'] and
                         g['start'] <= int(p.get('ingress_ns', 0)) <= g['end'] for g in guards) for p in losses)
        packets['renewal'] = dict(packets=len(renewal['packets']), losses=len(losses),
            loss_ingress_inside_guard=inside, loss_ingress_outside_guard=len(losses) - inside,
            capture_stats=renewal['capture_stats'], zero_loss_predicate=not losses)
    return dict(metrics={key: stats(value) for key, value in values.items()},
                values_ns={key: [str(n) for n in value] for key, value in values.items()},
                failed_operations=failures, snapshots=len(rows), spans=len(spans), sequence_gaps=gaps,
                packets=packets, binaries=read(path / 'binaries.json'), image=read(path / 'fixture-image.json'),
                raw_pins={str(p.relative_to(path)): pin(p) for p in sorted(path.rglob('*')) if p.is_file()})


def collect(root):
    base = root / '.agent/f43/timing'
    samples = lines(base / 'samples.jsonl')
    out = dict(scope='TEST ONLY same-host D-133/D-138; timestamp-only overlays; nearest-rank percentiles.',
               sources=read(base / 'sources.json'), window=read(base / 'window.json'),
               overlays=read(base / 'overlays/pins.json'),
               reservations=lines(base / 'reservations.jsonl'), releases=lines(base / 'releases.jsonl'),
               completion=read(base / 'completion.json'), legs=[])
    for row in read(base / 'results.json', []):
        ev = evidence(root / row['evidence'])
        # Include preflight for refused legs and successful entry. Windows are
        # isolated by the actual reserve/release receipts, not repeated labels.
        reserved = next(r['utc'] for r in out['reservations'] if r['window'] == row['window'] and r['exit'] == 0)
        released = next(r['utc'] for r in out['releases'] if r['window'] == row['window'])
        selected = [s for s in samples if s['label'] == row['leg'] and reserved <= s['utc'] <= released]
        public_samples = [{k: v for k, v in s.items() if k != 'foreign_suites'} |
                          {'foreign_suite_count': len(s['foreign_suites'])} for s in selected]
        out['legs'].append(dict(row, measurement=ev, qualification_samples=public_samples))
    out['pooled'] = {}
    for arm in ['A', 'B']:
        rows = [r for r in out['legs'] if r['arm'] == arm and r['qualified']]
        out['pooled'][arm] = dict(qualified_legs=[dict(leg=r['leg'], window=r['window']) for r in rows],
            metrics={key: stats([int(n) for r in rows for n in r['measurement']['values_ns'][key]]) for key in METRICS},
            failed_operations=[dict(leg=r['leg'], window=r['window'], **failure) for r in rows
                               for failure in r['measurement']['failed_operations']])
    out['monitoring_pins'] = {p.name: pin(p) for p in sorted(base.glob('*.jsonl'))}
    out['monitoring_limit'] = ('Successive /proc PID/start samples cannot reconstruct CPU for processes '
                               'that start and exit entirely between samples; departed PIDs are retained.')
    return out


def markdown(out):
    rows = ['# Follow-up 43 timing results — 2026-11-03', '',
            'TEST ONLY. All times below are milliseconds. Unqualified legs are retained and excluded from pooling.', '',
            '| Leg | Window | Qualified | Scenarios | Outer guard p99 / max | Serving Step p99 / max | Failed operations |',
            '| --- | --- | --- | --- | --- | --- | --- |']
    def pair(metrics, key):
        m = metrics[key]
        return ' / '.join(f'{int(m[k]) / 1e6:.3f}' if m[k] is not None else '—' for k in ['p99_ns', 'max_ns'])
    for r in out['legs']:
        m = r['measurement']
        rows.append(f"| {r['leg']} | {r['window']} | {r['qualified']}: {r.get('reason', '')} | "
                    f"{len(r.get('passed_scenarios', []))}/13 | {pair(m['metrics'], 'outer_guard')} | "
                    f"{pair(m['metrics'], 'whole_serving_step')} | {len(m['failed_operations'])} |")
    for label, metrics in [(f"Window {r['window']} {r['leg']}", r['measurement']['metrics']) for r in out['legs']] + [
            (f'Qualified pooled {arm}', value['metrics']) for arm, value in out['pooled'].items()]:
        rows += ['', f'## {label}', '', '| Span | n | p50 | p90 | p99 | max |', '| --- | --- | --- | --- | --- | --- |']
        for key, m in metrics.items():
            cells = [f'{int(m[k]) / 1e6:.3f}' if m[k] is not None else '—' for k in ['p50_ns', 'p90_ns', 'p99_ns', 'max_ns']]
            rows.append(f"| {key} | {m['count']} | " + ' | '.join(cells) + ' |')
    rows += ['', 'The JSON companion retains every failed operation, packet predicates, source/binary/fixture pins,',
             'qualification samples, refusals and releases. Only qualified legs enter the arm pools.',
             'Attribution requires repeated direction in each window and a code explanation; see the outcome record.', '']
    return '\n'.join(rows)


if __name__ == '__main__':
    root = Path.cwd()
    out = collect(root)
    raw = json.dumps(out, indent=2) + '\n'
    for forbidden in [str(root), str(Path.home())]:
        if forbidden in raw:
            raise ValueError('private location in public result')
    suffix = sys.argv[1] if len(sys.argv) > 1 else 'timing-results'
    dest = root / 'docs/node' / ('followup43-' + suffix + '-2026-11-03')
    dest.with_suffix('.json').write_text(raw)
    dest.with_suffix('.md').write_text(markdown(out))
    print(json.dumps({'legs': len(out['legs']), 'qualified': {a: len(p['qualified_legs']) for a, p in out['pooled'].items()}}))
