#!/usr/bin/env python3
"""Exact producer/packet correlation. Every diagnostic loss remains a loss.

Packet capture timestamps are userspace read observations. Command brackets
bound kernel commits; neither an overlap nor a netlink receipt is a drop time.
"""
import json
import gzip
from pathlib import Path
import statistics
import re
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


def read_operations(root):
    """Lossless gzip keeps the expanded diagnostic stream publishable."""
    path = root/'producer-operations.jsonl'
    if path.exists():
        with path.open() as stream:
            return [json.loads(line) for line in stream]
    with gzip.open(str(path)+'.gz', 'rt') as stream:
        return [json.loads(line) for line in stream]


def apply_context(item):
    context = {k:v for k,v in item.items() if k not in ('operation', 'lease')}
    commands = context['command_lifecycle']
    if commands['status'] == 'available':
        # Full raw records remain in the original producer and failed/nearest
        # Apply details. This inventory joins by identity + command index;
        # duplicating every cgroup sample here adds hundreds of MB, no evidence.
        context['command_lifecycle'] = dict(commands,
            source='producer-operations.jsonl (or its lossless .gz)',
            records=[{k:v for k,v in r.items() if k != 'record'} for r in commands['records']])
    return context


def correlated_apply(item):
    if item['operation']['succeeded'] and item['command_lifecycle']['status'] == 'available':
        return dict(apply_context(item), lease=item['lease'],
                    operation={k:v for k,v in item['operation'].items() if k != 'commands'})
    return item


HOST_COUNTERS = ('cpu_total_ticks', 'cpu_idle_ticks', 'cpu_iowait_ticks',
                 'procs_running', 'procs_blocked')
HOST_CONTENTION_RULE = ('TEST ONLY diagnostic: positive aggregate tick delta, zero idle and iowait '
                        'tick deltas, and more than one runnable process in both samples; '
                        'kernel aggregate, not cgroup entitlement or causal proof')


def host_context(row):
    """Validate the additive producer block; never borrow another Apply's host."""
    host = row.get('host')
    result = dict(status='unavailable', reason='host block absent', samples=host,
                  idle_percent=None, contention='unavailable', contention_rule=HOST_CONTENTION_RULE,
                  command_status=dict(failed_command=None, exit_code=None, signal=None))
    if host is None:
        return result
    if not isinstance(host, dict) or set(host) - {'before_apply', 'after_apply', 'failed_command', 'exit_code', 'signal'}:
        raise ValueError('invalid/unsanitized host block')
    if 'failed_command' in host and host['failed_command'] not in ('nft', 'tc', 'other'):
        raise ValueError('unsanitized failed command')
    for key, lo, hi in (('exit_code', -1, 255), ('signal', 1, 64)):
        if key in host and (type(host[key]) is not int or not lo <= host[key] <= hi):
            raise ValueError('invalid command status: ' + key)
    result['command_status'] = {k: host.get(k) for k in result['command_status']}
    if row['timing']['succeeded'] and any(k in host for k in result['command_status']):
        raise ValueError('successful Apply has failed command status')
    samples = [host[k] for k in ('before_apply', 'after_apply')]
    for s in samples:
        if not isinstance(s, dict) or set(s) != {'start_ns', 'end_ns', 'ok', *HOST_COUNTERS} or type(s['ok']) is not bool:
            raise ValueError('invalid host sample')
        for key in ('start_ns', 'end_ns', *HOST_COUNTERS):
            value = ns(s, key)
            if str(value) != s[key] or value > 18446744073709551615:
                raise ValueError('invalid host decimal: ' + key)
        if s['ok']:
            interval(s)
            if ns(s, 'cpu_idle_ticks') + ns(s, 'cpu_iowait_ticks') > ns(s, 'cpu_total_ticks'):
                raise ValueError('host CPU counters exceed total')
    before, after = samples
    start, end = interval(row['timing'])
    if ((before['ok'] and ns(before, 'end_ns') > start) or
        (after['ok'] and not end <= ns(after, 'start_ns') <= ns(after, 'end_ns') <= ns(row['anchor'], 'mono_before_ns'))):
        raise ValueError('host samples do not bracket their Apply')
    if not all(s['ok'] for s in samples):
        result['reason'] = 'producer host sample unavailable'
        return result
    deltas = {k: ns(after, k) - ns(before, k) for k in HOST_COUNTERS[:3]}
    if any(v < 0 for v in deltas.values()) or deltas['cpu_total_ticks'] == 0:
        result['reason'] = 'non-monotone or zero CPU tick delta'
        return result
    if deltas['cpu_idle_ticks'] + deltas['cpu_iowait_ticks'] > deltas['cpu_total_ticks']:
        raise ValueError('host CPU delta exceeds total')
    # Float is presentation only. The diagnostic decision uses exact integers.
    recorded = (deltas['cpu_idle_ticks'] == deltas['cpu_iowait_ticks'] == 0 and
                all(ns(s, 'procs_running') > 1 for s in samples))
    result.update(status='available', reason=None, tick_deltas={k: str(v) for k,v in deltas.items()},
                  idle_percent=100*deltas['cpu_idle_ticks']/deltas['cpu_total_ticks'],
                  contention='recorded' if recorded else 'not_established')
    return result


COMMAND_PHASES = ('prepare', 'shape', 'install', 'release', 'verify', 'quarantine', 'revoke', 'rollback')
COMMAND_STAMPS = ('start_attempt', 'start_return', 'child_started_by', 'child_start_exact',
                  'cancel_observed', 'wait_return')


def command_lifecycle(row):
    """Validate detached command observations, always scoped to this Apply.

    Availability is authoritative. Start is a bracket, deadline is a conversion
    bracket, and resource deltas cannot establish scheduler causality.
    """
    commands = row['timing'].get('commands')
    result = dict(identity(row), status='unavailable', records=[], phases=[],
                  reason='command lifecycle records absent')
    if commands is None or commands == []:
        return result
    if not isinstance(commands, list):
        raise ValueError('commands must be an array')
    start, end = interval(row['timing'])

    def decimal(obj, key, signed=False):
        value = obj[key]
        if (not isinstance(value, str) or
            not re.fullmatch(r'-?(0|[1-9][0-9]*)' if signed else r'(0|[1-9][0-9]*)', value) or
            str(int(value)) != value or not -(2**63) <= int(value) < 2**63):
            raise ValueError('invalid lifecycle decimal: ' + key)
        return int(value)

    def boolean(obj, key):
        if type(obj[key]) is not bool:
            raise ValueError('invalid lifecycle boolean: ' + key)
        return obj[key]

    def stamp(s, deadline=False):
        if not isinstance(s, dict) or set(s) != {'ns', 'available'}:
            raise ValueError('invalid lifecycle stamp')
        n, available = decimal(s, 'ns', signed=deadline), boolean(s, 'available')
        if not available:
            if n != 0:
                raise ValueError('unavailable lifecycle stamp has a value')
            return None
        if not deadline and not start <= n <= end:
            raise ValueError('command stamp outside its Apply')
        return n

    def ordered(*values):
        known = [v for v in values if v is not None]
        if known != sorted(known):
            raise ValueError('unordered lifecycle stamps')

    def bracket(b, deadline=False):
        if not isinstance(b, dict) or set(b) != {'before', 'after'}:
            raise ValueError('invalid lifecycle bracket')
        a, b = stamp(b['before'], deadline), stamp(b['after'], deadline)
        ordered(a, b)
        return a, b

    def resources(r, phase):
        if not isinstance(r, dict) or set(r) - {'skip_reason'} != {'bracket', 'cgroup_v2_available', 'visible_ancestors_complete', 'host_ancestors_available', 'cgroups'}:
            raise ValueError('invalid lifecycle resource block')
        skipped = r.get('skip_reason')
        if skipped is not None and skipped not in ('install_bound', 'context_canceled', 'context_deadline_exceeded', 'context_other', 'unsupported_platform'):
            raise ValueError('invalid lifecycle resource skip reason')
        if skipped == 'install_bound' and phase != 'install':
            raise ValueError('install resource skip outside install phase')
        bounds = bracket(r['bracket'])
        for k in ('cgroup_v2_available', 'visible_ancestors_complete', 'host_ancestors_available'):
            boolean(r, k)
        groups = r['cgroups']
        if groups is None:
            groups = []  # Go's unavailable nil slice.
        if skipped and (any(b is not None for b in bounds) or groups or any(r[k] for k in ('cgroup_v2_available', 'visible_ancestors_complete', 'host_ancestors_available'))):
            raise ValueError('skipped resource sample carries observations')
        if not isinstance(groups, list) or len(groups) > 64:
            raise ValueError('invalid lifecycle cgroups')
        if not r['cgroup_v2_available'] and (groups or r['visible_ancestors_complete']):
            raise ValueError('unavailable cgroups have observations')
        for depth, group in enumerate(groups):
            if set(group) != {'depth', 'cpu_max', 'cpu_weight', 'cpu_stat', 'cpu_pressure'} or type(group['depth']) is not int or group['depth'] != depth:
                raise ValueError('invalid lifecycle cgroup depth/fields')
            for key in ('cpu_max', 'cpu_weight', 'cpu_stat', 'cpu_pressure'):
                field = group[key]
                if not isinstance(field, dict) or set(field) != {'available', 'value'}:
                    raise ValueError('invalid lifecycle resource field')
                available = boolean(field, 'available')
                value = field['value']
                if not isinstance(value, str) or len(value) > 16384 or (not available and value):
                    raise ValueError('invalid lifecycle resource value')
                if value and not re.fullmatch(r'[a-z0-9_=. \n]+', value):
                    raise ValueError('unsanitized lifecycle resource text')
        return bounds

    flags = ('started', 'ok', 'exit_available', 'signal_available', 'cpu_available', 'switches_available')
    counters = ('user_cpu_ns', 'system_cpu_ns', 'voluntary_switches', 'involuntary_switches')
    contexts = ('context_before', 'context_after', 'context_at_cancel')
    fields = {'kind', 'phase', 'context_deadline', *COMMAND_STAMPS, *flags, *counters,
              *contexts, 'exit_code', 'signal', 'before', 'after'}
    previous = start
    for index, c in enumerate(commands):
        if not isinstance(c, dict) or set(c) != fields:
            raise ValueError('invalid/unsanitized command lifecycle fields')
        if c['kind'] not in ('nft', 'tc', 'conntrack', 'ip', 'other') or c['phase'] not in COMMAND_PHASES:
            raise ValueError('invalid command kind/phase')
        for k in flags:
            boolean(c, k)
        for k in contexts:
            if c[k] not in ('none', 'canceled', 'deadline_exceeded', 'other', 'unavailable') or (k != 'context_at_cancel' and c[k] == 'unavailable'):
                raise ValueError('invalid command context state')
        for k in counters:
            decimal(c, k)
        for k, lo, hi in (('exit_code', -1, 255), ('signal', 0, 64)):
            if type(c[k]) is not int or not lo <= c[k] <= hi:
                raise ValueError('invalid lifecycle exit status')
        for flag, keys in (('exit_available', ('exit_code',)), ('signal_available', ('signal',)),
                           ('cpu_available', counters[:2]), ('switches_available', counters[2:])):
            if not c[flag] and any(int(c[k]) != 0 for k in keys):
                raise ValueError('unavailable command status has a value')
        times = {k: stamp(c[k]) for k in COMMAND_STAMPS}
        bracket(c['context_deadline'], deadline=True)
        before, after = resources(c['before'], c['phase']), resources(c['after'], c['phase'])
        ordered(previous, *before, times['start_attempt'], times['start_return'], times['wait_return'], *after)
        previous = next((v for v in reversed(after) if v is not None), times['wait_return'] or times['start_return'] or previous)
        ordered(times['start_attempt'], times['child_start_exact'], times['child_started_by'], times['wait_return'])
        ordered(times['start_attempt'], times['cancel_observed'], times['wait_return'], *after)
        if c['started']:
            if c['child_started_by'] != c['start_return']:
                raise ValueError('child-start bound differs from Start return')
        elif c['ok'] or any(times[k] is not None for k in ('child_started_by', 'child_start_exact', 'wait_return', 'cancel_observed')) or any(c[k] for k in flags[2:]):
            raise ValueError('unstarted command has child/exit observations')
        if c['context_at_cancel'] == 'unavailable' and times['cancel_observed'] is not None:
            raise ValueError('cancel stamp without callback context')
        if c['ok'] and ((c['exit_available'] and c['exit_code'] != 0) or (c['signal_available'] and c['signal'] != 0)):
            raise ValueError('successful command has failed exit status')
        d = row['timing']['deadline']
        if c['phase'] in ('install', 'release'):
            left, right = ('install_start_ns', 'installed_by_ns') if c['phase'] == 'install' else ('release_start_ns', 'release_end_ns')
            a, b = interval(d, left, right)
            for value in (*before, *times.values(), *after):
                if value is not None and not a <= value <= b:
                    raise ValueError('lifecycle outside its phase command bracket')
        result['records'].append(dict(identity(row), index=index, record=c))
    for phase in COMMAND_PHASES:
        indices = [i for i, c in enumerate(commands) if c['phase'] == phase]
        if indices:
            result['phases'].append(dict(phase=phase, command_indices=indices,
                description='; '.join(f"{commands[i]['kind']}: started={commands[i]['started']}, ok={commands[i]['ok']}, "
                    f"exit={commands[i]['exit_code'] if commands[i]['exit_available'] else 'unavailable'}, "
                    f"signal={commands[i]['signal'] if commands[i]['signal_available'] else 'unavailable'}, "
                    f"context={commands[i]['context_before']}→{commands[i]['context_after']}, "
                    f"cancel={commands[i]['context_at_cancel']}" for i in indices)))
    result.update(status='available', reason=None)
    return result


def apply_diagnostic(row):
    t, d = row['timing'], row['timing']['deadline']
    host = host_context(row)
    commands = command_lifecycle(row)
    checks = t['verifications']
    verification = ('passed' if checks[-1]['verified'] else 'failed') if checks else 'unavailable'
    stage = None
    if not t['succeeded']:
        stage = ('pre_install' if not ns(d, 'install_start_ns') else
                 'install' if not d['install_ok'] else
                 'release' if not d['guard_released'] else
                 'verification' if verification == 'failed' else
                 'post_verification' if verification == 'passed' else 'unavailable')
    render, installed, bound = (ns(d, k) for k in ('render_start_ns', 'installed_by_ns', 'install_bound_ns'))
    elapsed = installed-render if render and installed and bound else None
    overrun = elapsed > bound if elapsed is not None else None
    if overrun and not t['succeeded']:
        stage = 'install'
    label = 'unavailable' if host['status'] == 'unavailable' else 'no recorded install-overrun/contention conjunction'
    if stage == 'install' and overrun and not d['guard_released'] and host['contention'] == 'recorded':
        label = 'fail-closed install overrun with recorded host contention'
    return dict(host_context=host, command_lifecycle=commands, failure_stage=stage, verification_status=verification,
                install_phase_description=next((p['description'] for p in commands['phases'] if p['phase'] == 'install'), 'unavailable'),
                render_to_installed_ns=str(elapsed) if elapsed is not None else None,
                install_bound_met=not overrun if overrun is not None else None,
                install_overrun=overrun, diagnostic_label=label,
                cause_status='unproven' if not t['succeeded'] else 'not_applicable')


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
        if 'host' in row and (row['source'] != 'apply' or row.get('timing', {}).get('operation') != 'apply'):
            raise ValueError('host context outside Apply')
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
        if 'commands' in t and (t['operation'] != 'apply' or row['source'] != 'apply'):
            raise ValueError('command lifecycle outside Apply')
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
            render, installed = ns(d, 'render_start_ns'), ns(d, 'installed_by_ns')
            if installed:
                if not start <= render <= ns(d, 'install_start_ns') <= installed <= end or ns(d, 'install_bound_ns') <= 0:
                    raise ValueError('invalid render/install bracket or bound')
            item.update(apply_diagnostic(row))
            if d['install_ok']:
                span('render_to_installed', render, installed)
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
                span('completed_guard_render_to_installed', render, installed)
                item['complete_guard'] = True
            applies.append(item)
    if not rows or not applies:
        raise ValueError('missing Apply evidence')
    def distribution(values):
        ordered = sorted(values)
        return dict(count=len(values), median_ns=str(int(statistics.median(values))), worst_ns=str(max(values)),
                    **{name+'_ns': str(ordered[(q*len(ordered)+99)//100-1]) for name,q in [('p50',50),('p90',90),('p99',99),('max',100)]})
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
        result['nearest_apply'] = correlated_apply(min(validated['applies'], key=lambda a: abs(ingress-ns(a['operation'],'start_ns'))))
        prior = [a for a in validated['applies'] if ns(a['operation'],'end_ns') <= ingress]
        result['last_completed_apply'] = correlated_apply(max(prior, key=lambda a: ns(a['operation'],'end_ns'))) if prior else None
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
    rows = read_operations(root)
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
                  apply_contexts=[apply_context(a) for a in validated['applies']],
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
                              losses=expiry_losses,
                              pre_deadline_losses=sum(p['expiry_region']=='before deadline' for p in expiry_losses),
                              literal_zero_pre_deadline_loss=not any(p['expiry_region']=='before deadline' for p in expiry_losses)),
                  interpretation='Outer bracket is install-start through release-end; inner is installed-by through release-start. Capture reads and command overlaps do not locate the kernel commit/drop instant. All loss counts remain unchanged.')
    (root/'producer-timing-analysis.json').write_text(json.dumps(result,indent=2)+'\n')
    compact = dict({k:v for k,v in result.items() if k not in ('producer_rows','failed_applies','apply_contexts')}, renewal={k:v for k,v in result['renewal'].items() if k not in ('losses','actual_renewals')},
                   expiry={k:v for k,v in result['expiry'].items() if k not in ('losses','last_accepted_renewal')})
    print(json.dumps(compact,indent=2))
    return result


def replay_first_attempt(root):
    """Read the immutable attempt; write only a new, separately dated review."""
    source = root/'history/follow-up-10-attempt-1'
    rows = read_operations(source)
    v = validate_operations(rows)
    clock = json.loads((source/'producer-clock.json').read_text())
    packets = packet_trace(json.loads((source/'renewal-probes.json').read_text()),
                           json.loads((source/'renewal-packets.json').read_text()), clock)
    losses = [correlate(p, v) for p in packets if p['result'] != 'ok']
    result = dict(run_id=json.loads((source/'AT10.json').read_text())['run_id'],
                  review='Follow-up 11: agent-core 29 consumer replay; original archive unchanged',
                  source='history/follow-up-10-attempt-1/producer-operations.jsonl',
                  snapshots=v['snapshots'], source_counts=v['source_counts'], brackets=v['brackets'],
                  failed_applies=v['failed_applies'],
                  host_context='unavailable', cause_status='unproven',
                  verification_rule='No verifier records means unavailable, not failed verification; baseline cause names do not supply verifier records.',
                  renewal=dict(probes=len(packets), delivered=sum(p['result']=='ok' for p in packets),
                               loss_count=len(losses), ingress_in_outer_guard=sum(bool(p['ingress_in_outer_guard']) for p in losses),
                               unresolved_ingress=sum(not p['ingress_in_outer_guard'] for p in losses), losses=losses),
                  expiry='unavailable: no current whole-node window was executed in this attempt')
    if any(a['host_context']['status'] != 'unavailable' for a in v['applies']):
        raise ValueError('historical replay unexpectedly contains host samples')
    (root/'first-attempt-host-review.json').write_text(json.dumps(result, indent=2)+'\n')
    return result


if __name__ == '__main__':
    if sys.argv[1:] == ['--replay-first-attempt']:
        replay_first_attempt(Path('docs/e2e/at/isolation'))
    else:
        analyze(Path(sys.argv[1] if len(sys.argv)>1 else 'docs/e2e/at/isolation'))
