#!/usr/bin/env python3
"""Exact producer/packet correlation. Every diagnostic loss remains a loss.

Packet capture timestamps are userspace read observations. Command brackets
bound kernel commits; neither an overlap nor a netlink receipt is a drop time.
"""
import json
from pathlib import Path
import statistics
import math
from collections import Counter
import sys


def ns(row, key):
    value = row[key]
    if not isinstance(value, str) or not value.isascii() or not value.isdecimal():
        raise ValueError('nanoseconds must be decimal strings: ' + key)
    return int(value)


def interval(row, start='start_ns', end='end_ns'):
    a, b = ns(row, start), ns(row, end)
    if not 0 < a <= b:
        raise ValueError('missing/unordered interval: ' + start)
    return a, b


def identity(row):
    return {k: row[k] for k in ('run_id', 'boot_id', 'sequence')}


def validate_operations(rows):
    seen, boots, spans, applies, verifiers = {}, {}, {}, [], []
    def span(name, a, b):
        if not 0 < a <= b:
            raise ValueError('unordered ' + name)
        spans.setdefault(name, []).append(b-a)
    for row in rows:
        run = row['run_id']
        seq = ns(row, 'sequence')
        if seq != seen.get(run, 0)+1:
            raise ValueError('missing/duplicate snapshot')
        seen[run] = seq
        if boots.setdefault(run, row['boot_id']) != row['boot_id']:
            raise ValueError('run changed boot')
        a, b = interval(row['anchor'], 'mono_before_ns', 'mono_after_ns')
        if not row['anchor']['wall'].endswith('Z'):
            raise ValueError('non-UTC anchor')
        if 'timing' not in row:
            kind = row['source']
            if kind in ('renewal', 'baseline'):
                start, end = interval(row[kind])
                if end > a:
                    raise ValueError(kind + ' exceeds anchor')
                span(kind, start, end)
                if kind == 'baseline' and (not row[kind]['cause'] or not isinstance(row[kind]['succeeded'], bool)):
                    raise ValueError('incomplete baseline writer evidence')
            elif kind == 'heartbeat':
                h = row[kind]
                if not 0 < ns(h, 'mono_ns') <= a or not isinstance(h['written'], bool):
                    raise ValueError('invalid heartbeat evidence')
                ns(h, 'policy_expires_mono_ns')
                ns(h, 'generation')
            else:
                raise ValueError('missing producer timing')
            continue
        t = row['timing']
        if t['version'] != 1 or t['clock'] != 'CLOCK_MONOTONIC':
            raise ValueError('unsupported timing clock/version')
        start, end = interval(t)
        if end > a:
            raise ValueError('operation exceeds anchor')
        span(t['operation'], start, end)
        previous = start
        for index, v in enumerate(t['verifications']):
            va, vb = interval(v)
            if not previous <= va <= vb <= end:
                raise ValueError('verifier outside operation or out of order')
            previous = vb
            span('verifier', va, vb)
            for label in ('nft_read', 'precision_read', 'tc_read'):
                r = v[label]
                x, y = ns(r, 'start_ns'), ns(r, 'end_ns')
                if x or y:
                    span(label, x, y)
                    if not va <= x <= y <= vb:
                        raise ValueError('executor read outside verifier')
                elif r.get('ok', r.get('read_ok', False)):
                    raise ValueError('successful read has no timing')
            if v['verified'] and not (v['nft_read']['ok'] and v['tc_read']['ok'] and
                                      v['precision_read']['read_ok'] and v['precision_read']['verified']):
                raise ValueError('verified without successful reads')
            verifiers.append(dict(identity(row), index=index, timing=v))
        if t['verifications'] and row.get('tc_sha256') != t['verifications'][-1]['tc_sha256']:
            raise ValueError('D-106 digest mismatch')
        if t['operation'] == 'apply':
            d = t['deadline']
            item = dict(identity(row), operation=t, lease=row.get('lease'), complete_guard=False)
            if ns(d, 'apply_start_ns') != start:
                raise ValueError('Apply entry mismatch')
            if d['guard_released'] and not d['install_ok']:
                raise ValueError('release without install')
            # Failed/partial stages stay retained. Only complete command flags
            # establish the outer and inner guard intervals.
            for name, left, right in (('install_command', 'install_start_ns', 'installed_by_ns'),
                                      ('release_command', 'release_start_ns', 'release_end_ns')):
                x, y = ns(d, left), ns(d, right)
                if x or y:
                    span(name, x, y)
                    if not start <= x <= y <= end:
                        raise ValueError('command outside Apply')
            if d['install_ok']:
                render, installed = ns(d, 'render_start_ns'), ns(d, 'installed_by_ns')
                span('render_to_installed', render, installed)
                item['install_bound_met'] = installed-render <= ns(d, 'install_bound_ns')
            if d['install_ok'] and d['guard_released']:
                render, install, installed, release, released = [ns(d, k) for k in
                    ('render_start_ns', 'install_start_ns', 'installed_by_ns', 'release_start_ns', 'release_end_ns')]
                if not start <= render <= install <= installed <= release <= released <= end:
                    raise ValueError('guard order')
                if not item['install_bound_met']:
                    raise ValueError('released guard exceeded install bound')
                span('outer_guard', install, released)
                span('inner_guard', installed, release)
                span('completed_guard_install_command', install, installed)
                item['complete_guard'] = True
            applies.append(item)
    if not rows or not applies or not verifiers:
        raise ValueError('missing Apply/verifier evidence')
    def distribution(values):
        ordered = sorted(values)
        return dict(count=len(values), median_ns=str(int(statistics.median(values))), worst_ns=str(max(values)),
                    **{name+'_ns': str(ordered[math.ceil(q*len(ordered))-1]) for name,q in [('p50',.5),('p90',.9),('p99',.99),('max',1)]})
    return dict(source_counts=dict(Counter(r['source'] for r in rows)),
                producer_rows=[r for r in rows if r['source'] in ('baseline','heartbeat')],
                failed_applies=[a for a in applies if not a['operation']['succeeded']],
                snapshots=len(rows), runs=[dict(run_id=r, boot_id=boots[r], snapshots=n) for r,n in seen.items()],
                brackets={k: distribution(v) for k,v in spans.items()},
                applies=applies, verifiers=verifiers,
                failed_operations=sum(not r['timing']['succeeded'] for r in rows if 'timing' in r),
                recovery_double_verifiers=[identity(r) for r in rows if len(r.get('timing',{}).get('verifications',[])) == 2])


def validate_clock(actual, expected):
    # Namespace inodes differ between containers and across Docker restarts.
    # Linux timens offsets are relative to the initial namespace and immutable
    # once populated. Require exactly zero offsets in both clocks; do not add
    # a time tolerance or infer equivalence merely from similar wall samples.
    def offsets(row):
        fields = [line.split() for line in row['offsets'].splitlines()]
        if len(fields) != 2 or any(len(f) != 3 for f in fields):
            raise ValueError('missing clock offsets')
        return {f[0]: (int(f[1]), int(f[2])) for f in fields}
    if (actual['boot_id'] != expected['boot_id'] or actual['clock'] != expected['clock'] or
        actual['clock'] != 'CLOCK_MONOTONIC' or not actual['time_namespace'] or not expected['time_namespace'] or
        offsets(actual) != {'monotonic': (0,0), 'boottime': (0,0)} or offsets(actual) != offsets(expected)):
        raise ValueError('packet/producer boot or time offset mismatch')


def packet_trace(probes, captures, clock):
    ready = [r for r in captures if r.get('event') == 'ready']
    done = [r for r in captures if r.get('event') == 'complete']
    if len(ready) != 1 or len(done) != 1 or not done[0].get('statistics_available') or done[0].get('drops') != 0:
        raise ValueError('incomplete or dropped capture')
    validate_clock(ready[0]['trace_clock'], clock)
    incoming, outgoing = {}, {}
    for p in captures:
        target = incoming if p.get('interface') == 'fvmb0' else outgoing if p.get('interface') == 'fvup0' else None
        if target is not None:
            target.setdefault(p['packet_id'], ns(p['capture_read'], 'mono_ns'))
    seen, result = set(), []
    for p in probes:
        validate_clock(p['trace_clock'], clock)
        key = p['packet_id']
        if key in seen or key not in incoming:
            raise ValueError('duplicate probe or missing ingress')
        seen.add(key)
        a, b = ns(p['send'], 'mono_ns'), ns(p['read_completion'], 'mono_ns')
        interval(dict(start_ns=str(a), end_ns=str(b)))
        result.append(dict(packet_id=key, result=p['result'], sent_ns=str(a), read_completion_ns=str(b),
                           ingress_ns=str(incoming[key]), egress_ns=str(outgoing.get(key, 0))))
    if not result:
        raise ValueError('empty packet trace')
    return result


def correlate(packet, validated):
    start, end, ingress = [ns(packet, k) for k in ('sent_ns','read_completion_ns','ingress_ns')]
    def overlap(a,b):
        return a <= end and start <= b
    result = dict(packet=packet)
    predicates = {
        'apply_window_overlap': lambda t,d: overlap(ns(t,'start_ns'), ns(t,'end_ns')),
        'install_window_overlap': lambda t,d: ns(d,'install_start_ns') > 0 and overlap(ns(d,'install_start_ns'), ns(d,'installed_by_ns')),
        'release_window_overlap': lambda t,d: ns(d,'release_start_ns') > 0 and overlap(ns(d,'release_start_ns'), ns(d,'release_end_ns')),
        'ingress_in_install': lambda t,d: 0 < ns(d,'install_start_ns') <= ingress <= ns(d,'installed_by_ns'),
        'ingress_in_release': lambda t,d: 0 < ns(d,'release_start_ns') <= ingress <= ns(d,'release_end_ns'),
        'ingress_in_outer_guard': lambda t,d: 0 < ns(d,'install_start_ns') <= ingress <= ns(d,'release_end_ns'),
        'ingress_in_inner_guard': lambda t,d: 0 < ns(d,'installed_by_ns') <= ingress <= ns(d,'release_start_ns'),
    }
    for label, predicate in predicates.items():
        result[label] = [identity(a) for a in validated['applies'] if
            (label == 'apply_window_overlap' or a['complete_guard']) and predicate(a['operation'], a['operation']['deadline'])]
    result['verifier_window_overlap'] = [dict(identity(v), index=v['index']) for v in validated['verifiers']
                                        if overlap(*interval(v['timing']))]
    if validated['applies']:
        # Full nearest Apply carries its render, install/release, both verifiers
        # and every executor read. References above disambiguate other overlaps.
        result['nearest_apply'] = min(validated['applies'], key=lambda a: abs(ingress-ns(a['operation'],'start_ns')))
        prior = [a for a in validated['applies'] if ns(a['operation'],'end_ns') <= ingress]
        result['last_completed_apply'] = max(prior, key=lambda a: ns(a['operation'],'end_ns')) if prior else None
    prior = [v for v in validated['verifiers'] if ns(v['timing'],'end_ns') <= ingress]
    result['last_completed_verifier'] = max(prior, key=lambda v: ns(v['timing'],'end_ns')) if prior else None
    producers = validated.get('producer_rows', [])
    baselines = [r for r in producers if r['source'] == 'baseline']
    result['baseline_window_overlap'] = [r for r in baselines if overlap(*interval(r['baseline']))]
    prior = [r for r in baselines if ns(r['baseline'], 'end_ns') <= ingress]
    result['last_completed_baseline'] = max(prior, key=lambda r: ns(r['baseline'], 'end_ns')) if prior else None
    # The anchor follows the actual heartbeat write; its mono sample alone
    # would not prove that the write had completed before this packet read.
    prior = [r for r in producers if r['source'] == 'heartbeat' and r['heartbeat']['written']
             and ns(r['anchor'], 'mono_after_ns') <= ingress]
    result['last_written_heartbeat'] = max(prior, key=lambda r: ns(r['anchor'], 'mono_after_ns')) if prior else None
    result['classification'] = ('guard-correlated; loss retained' if result['ingress_in_outer_guard']
                                else 'outside completed guard brackets; unresolved')
    return result


def analyze(root):
    def read(name):
        return json.loads((root/name).read_text())
    rows = [json.loads(line) for line in (root/'producer-operations.jsonl').read_text().splitlines()]
    if 'measurement unavailable' in (root/'agent.log.txt').read_text():
        raise ValueError('producer reported a failed measurement write')
    validated = validate_operations(rows)
    clock = read('producer-clock.json')
    if {r['boot_id'] for r in rows} != {clock['boot_id']}:
        raise ValueError('producer changed boot')
    after = read('producer-clock-after-restart.json')
    validate_clock(dict(after, clock='CLOCK_MONOTONIC'), clock)
    if after['measurement_run_id'] not in {r['run_id'] for r in rows}:
        raise ValueError('live process clock has no producer run')
    for name in ('renewal-packets.json', 'whole-node-packets.json'):
        ready = [r for r in read(name) if r.get('event') == 'ready']
        if len(ready) != 1 or ready[0]['trace_clock']['time_namespace'] != after['time_namespace']:
            raise ValueError('node packet recorder differs from live agent namespace')
    renewal = packet_trace(read('renewal-probes.json'), read('renewal-packets.json'), clock)
    expiry_doc = read('whole-node-probes.json')
    expiry = packet_trace(expiry_doc['probes'], read('whole-node-packets.json'), clock)
    verdict = read('deadline-verdict.json')
    deadline = ns(verdict, 'deadline_mono_ns')
    lo, hi = min(ns(p,'sent_ns') for p in renewal), max(ns(p,'read_completion_ns') for p in renewal)
    renewals = [r for r in rows if r.get('renewal',{}).get('new_lease') and lo <= ns(r['renewal'],'start_ns') <= hi]
    if len(renewals) < 2:
        raise ValueError('fewer than two actual signed renewals')
    if {r['run_id'] for r in renewals} != {after['measurement_run_id']}:
        raise ValueError('renewals outside sampled live producer process')
    before_expiry = [r for r in rows if r.get('renewal',{}).get('new_lease') and ns(r['renewal'],'end_ns') < min(ns(p,'sent_ns') for p in expiry)]
    if not before_expiry or ns(before_expiry[-1]['lease'],'node_deadline_ns') != deadline:
        raise ValueError('expiry differs from last accepted producer deadline')
    if any(r.get('renewal',{}).get('new_lease') and min(ns(p,'sent_ns') for p in expiry) <= ns(r['renewal'],'end_ns') <= max(ns(p,'read_completion_ns') for p in expiry) for r in rows):
        raise ValueError('new lease during expiry capture')
    losses = [correlate(p,validated) for p in renewal if p['result'] != 'ok']
    expiry_losses = [correlate(p,validated) for p in expiry if p['result'] != 'ok' or ns(p,'egress_ns') == 0]
    for p in expiry_losses:
        at = ns(p['packet'],'ingress_ns')
        p['ingress_offset_ns'] = str(at-deadline)
        p['expiry_region'] = 'at/after deadline' if at >= deadline else 'before deadline'
    wire = read('tc-evidence-index.json')['reads']
    digests = {v['timing']['tc_sha256'] for v in validated['verifiers']}
    valid_wire = [r for r in wire if r['digest_matches'] and r['tc_evidence']['sha256'] in digests]
    during = [a for a in validated['applies'] if lo <= ns(a['operation'],'start_ns') <= hi]
    result = dict(run_id=read('AT10.json')['run_id'], validation='pass',
                  scope='Assembled real central/native NetBird; 80ms dual-phase diagnostic; no D-120 tolerance.',
                  historical=dict(run_id='7779d83c-2f78-4699-ac88-8fcc59e8cf87', probes=1396, losses=17,
                                  limitation='No exact producer timestamps in that run; never retrospectively classified.',
                                  evidence='history/follow-up-8/continuity-analysis.json'),
                  clock=clock, snapshots=validated['snapshots'], runs=validated['runs'], brackets=validated['brackets'],
                  clock_after_restart=after, source_counts=validated['source_counts'],
                  producer_rows=validated['producer_rows'], failed_applies=validated['failed_applies'],
                  clock_basis='Same kernel boot, exact zero monotonic/boottime offsets to the initial namespace. Namespace inodes are retained, not assumed identical across Docker restarts or member containers.',
                  capture_scope='Source node timing retained across every process restart; clone wire bodies remain separate and do not supply source timing.',
                  failed_operations=validated['failed_operations'], recovery_double_verifiers=validated['recovery_double_verifiers'],
                  wire_digests=dict(reads=len(wire), recompute_ok=sum(r['digest_matches'] for r in wire), matched_producer_digest=len(valid_wire)),
                  renewal=dict(probes=len(renewal), delivered=sum(p['result']=='ok' for p in renewal), loss_count=len(losses), losses=losses,
                               actual_renewals=renewals, applies=len(during),
                               complete_guards=sum(a['complete_guard'] for a in during),
                               loss_windows_overlapping_install=sum(bool(p['install_window_overlap']) for p in losses),
                               ingress_in_outer_guard=sum(bool(p['ingress_in_outer_guard']) for p in losses),
                               unresolved_ingress=sum(not p['ingress_in_outer_guard'] for p in losses)),
                  expiry=dict(deadline_ns=str(deadline), last_accepted_renewal=before_expiry[-1], packet_verdict= {k:v for k,v in verdict.items() if k!='packets'},
                              losses=expiry_losses),
                  interpretation='Outer bracket is install-start through release-end; inner is installed-by through release-start. Capture reads and command overlaps do not locate the kernel commit/drop instant. All loss counts remain unchanged.')
    (root/'producer-timing-analysis.json').write_text(json.dumps(result,indent=2)+'\n')
    compact = dict({k:v for k,v in result.items() if k not in ('producer_rows','failed_applies')}, renewal={k:v for k,v in result['renewal'].items() if k not in ('losses','actual_renewals')},
                   expiry={k:v for k,v in result['expiry'].items() if k not in ('losses','last_accepted_renewal')})
    print(json.dumps(compact,indent=2))
    return result


if __name__ == '__main__':
    analyze(Path(sys.argv[1] if len(sys.argv)>1 else 'docs/e2e/at/isolation'))
