"""Validate local producer snapshots and correlate diagnostic packet windows.

No D-120 tolerance: losses remain losses even inside a successful guard bracket.
The historical isolation run has no producer snapshots and is not relabeled.
"""
import json
from pathlib import Path
import statistics
import sys


def ns(row, key):
    value = row[key]
    if not isinstance(value, str) or not value.isdigit():
        raise ValueError('nanoseconds must be decimal strings: '+key)
    return int(value)


def bracket(start, end):
    assert 0 < start <= end, (start, end)
    return end-start


def analyze(operations, renewal, expiry):
    seen, runs, spans, guards = {}, set(), {}, []
    def span(name, a, b):
        spans.setdefault(name, []).append(bracket(a, b))
    for row in operations:
        run = row['run_id']
        assert int(row['sequence']) == seen.get(run, 0)+1, 'missing snapshot'
        seen[run] = int(row['sequence'])
        runs.add((run, row['boot_id']))
        a = row['anchor']
        bracket(ns(a, 'mono_before_ns'), ns(a, 'mono_after_ns'))
        assert a['wall'].endswith('Z')
        if 'timing' not in row:
            assert row['source'] == 'renewal'
            bracket(ns(row['renewal'], 'start_ns'), ns(row['renewal'], 'end_ns'))
            continue
        t = row['timing']
        assert t['version'] == 1 and t['clock'] == 'CLOCK_MONOTONIC'
        span(t['operation'], ns(t, 'start_ns'), ns(t, 'end_ns'))
        assert ns(t, 'end_ns') <= ns(a, 'mono_before_ns')
        for v in t['verifications']:
            if v['verified']:
                assert v['nft_read']['ok'] and v['tc_read']['ok']
                assert v['precision_read']['read_ok'] and v['precision_read']['verified']
            span('verifier', ns(v, 'start_ns'), ns(v, 'end_ns'))
            assert ns(t,'start_ns') <= ns(v,'start_ns') <= ns(v,'end_ns') <= ns(t,'end_ns')
            for label in ['nft_read', 'precision_read', 'tc_read']:
                r = v[label]
                start, end = ns(r,'start_ns'), ns(r,'end_ns')
                if start or end:
                    span(label,start,end)
                    assert ns(v,'start_ns') <= start <= end <= ns(v,'end_ns')
                else:
                    assert not r.get('ok',r.get('read_ok',False))
        if t['verifications']:
            assert row['tc_sha256'] == t['verifications'][-1]['tc_sha256'], 'D-106 digest mismatch'
        if t['operation'] == 'apply':
            d = t['deadline']
            if d['guard_released']:
                assert d['install_ok'], 'release without successful install'
            if d['install_ok'] and d['guard_released']:
                start, installed, release, end = [ns(d,k) for k in ['install_start_ns','installed_by_ns','release_start_ns','release_end_ns']]
                assert ns(t,'start_ns') <= ns(d,'render_start_ns') <= start <= installed <= release <= end <= ns(t,'end_ns')
                span('install_command', start, installed)
                span('release_command', release, end)
                span('outer_guard', start, end)
                span('inner_guard', installed, release)
                span('render_to_installed', ns(d,'render_start_ns'), installed)
                assert installed-ns(d,'render_start_ns') <= ns(d,'install_bound_ns')
                guards.append(dict(sequence=row['sequence'], run_id=run, boot_id=row['boot_id'],
                                   start=start, installed=installed, release=release, end=end))
    assert len(runs) == len(seen), 'run changed boot'
    assert guards and any(r.get('timing',{}).get('operation') == 'verify' for r in operations)
    assert renewal['boot_id'] == expiry['boot_id'], 'packet traces changed boot'
    assert renewal['boot_id'] in {b for _, b in runs}, 'packets and operations have different boots'
    packets = renewal['packets']
    assert all(int(p.get('ingress_ns',0)) > 0 for p in packets), 'missing member ingress'
    assert all(s['drops'] == 0 for s in renewal['capture_stats']+expiry['capture_stats'])
    relevant = [g for g in guards if g['boot_id'] == renewal['boot_id']]
    losses = []
    for p in packets:
        if p['result'] == 'ok':
            continue
        start, end, ingress = [ns(p,k) for k in ['sent_ns','read_completion_ns','ingress_ns']]
        def matches(predicate):
            return [dict(run_id=g['run_id'], sequence=g['sequence']) for g in relevant if predicate(g)]
        losses.append(dict(packet=p, install_window_overlap=matches(lambda g: start <= g['installed'] and g['start'] <= end),
                           ingress_in_install=matches(lambda g: g['start'] <= ingress <= g['installed']),
                           ingress_in_outer_guard=matches(lambda g: g['start'] <= ingress <= g['end']),
                           ingress_in_inner_guard=matches(lambda g: g['installed'] <= ingress <= g['release'])))
    lo, hi = min(ns(p,'sent_ns') for p in packets), max(ns(p,'read_completion_ns') for p in packets)
    renewals = [r for r in operations if r.get('renewal',{}).get('new_lease') and lo <= ns(r['renewal'],'start_ns') <= hi]
    assert len(renewals) >= 2, 'two actual timed renewals required'
    assert all(r['boot_id'] == renewal['boot_id'] for r in renewals)
    assert ns(expiry,'deadline_ns') == ns(renewals[-1]['lease'],'node_deadline_ns'), 'expiry must use last accepted node deadline'
    d = ns(expiry,'deadline_ns')
    ps = expiry['packets']
    assert all(int(p.get('ingress_ns',0)) > 0 for p in ps)
    positives = [p for p in ps if int(p.get('egress_ns',0)) > 0]
    drops = [p for p in ps if not int(p.get('egress_ns',0))]
    assert len(positives) >= 3 and len(drops) >= 3
    last = max(ns(p,'egress_ns') for p in positives)
    first = min(ns(p,'sent_ns') for p in drops)
    assert last >= d-101_000_000 and first <= d+100_000_000
    assert all(ns(p,'egress_ns') < d and ns(p,'sent_ns') < first for p in positives)
    return dict(scope='TEST ONLY assembled agent/kernel/TLS fake control; 80ms dual-phase diagnostic; no D-120 tolerance',
                snapshots=len(operations), runs=[dict(run_id=r,boot_id=b) for r,b in sorted(runs)],
                brackets={k:dict(count=len(v),median_ns=str(int(statistics.median(v))),worst_ns=str(max(v))) for k,v in spans.items()},
                renewal=dict(probes=len(packets),delivered=sum(p['result']=='ok' for p in packets), losses=losses,
                             actual_renewals=renewals, loss_windows_overlapping_install=sum(bool(p['install_window_overlap']) for p in losses),
                             ingress_in_outer_guard=sum(bool(p['ingress_in_outer_guard']) for p in losses),
                             unresolved_ingress=sum(not p['ingress_in_outer_guard'] for p in losses)),
                expiry=dict(deadline_ns=str(d),last_positive_offset_ns=str(last-d),first_drop_offset_ns=str(first-d),
                            positives=len(positives),drops=len(drops),packet_predicate='pass'))


if __name__ == '__main__':
    directory = Path(sys.argv[1])
    result = analyze([json.loads(line) for line in (directory/'operations.jsonl').read_text().splitlines()],
                     json.loads((directory/'renewal.json').read_text()), json.loads((directory/'expiry.json').read_text()))
    (directory/'analysis.json').write_text(json.dumps(result,indent=2)+'\n')
    compact = dict(result, renewal={k:v for k,v in result['renewal'].items() if k not in ['losses','actual_renewals']})
    print(json.dumps(compact,indent=2))
