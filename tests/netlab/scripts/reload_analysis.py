#!/usr/bin/env python3
"""Correlate TEST ONLY TCP/UDP metadata with independent header captures and Apply.

Node any is authoritative for node ingress/egress. Origin/responder any traces
are additional namespace-local observations, not fields invented from a node
capture. Null means unobserved. No packet payload is captured or parsed.
"""
import argparse
from collections import Counter
from decimal import Decimal
import json
from pathlib import Path
import re
import statistics
import sys


def json_rows(path):
    return [json.loads(line) for line in path.read_text().splitlines()
            if line.startswith('{')]


def captures(path):
    return parse_capture(path.read_text(), path.with_suffix('.err').read_text())


def parse_capture(text, stats):
    rows = []
    pending = None
    for line in text.splitlines():
        head = re.match(r'^(\d+\.\d+)\s+(\S+)\s+(In|Out)\s+IP \(.*\bid (\d+),', line)
        if head:
            pending = dict(wall_ns=int(Decimal(head[1]) * 10**9),
                           interface=head[2], direction=head[3], ip_id=int(head[4]))
            continue
        body = re.match(r'\s*(\d+\.\d+\.\d+\.\d+)\.(\d+) > (\d+\.\d+\.\d+\.\d+)\.(\d+): UDP, length (\d+)', line)
        if body and pending:
            rows.append(dict(pending, source=body[1], source_port=int(body[2]),
                             destination=body[3], destination_port=int(body[4]), size=int(body[5])))
            pending = None
    n = re.search(r'(\d+) packets captured', stats)
    drops = re.search(r'(\d+) packets dropped by kernel', stats)
    if not n or not drops or int(drops[1]) != 0 or int(n[1]) != len(rows):
        raise ValueError(f'incomplete capture: parsed={len(rows)} stats={stats}')
    return rows


def guards(rows):
    out = []
    for row in rows:
        if row.get('event') != 'apply':
            continue
        install = [b for b in row['batches'] if b['kind'] == 'guard_install']
        release = [b for b in row['batches'] if b['kind'] == 'guard_release']
        if len(install) != 1 or len(release) != 1 or not install[0]['ok'] or not release[0]['ok']:
            raise ValueError('incomplete guard brackets')
        i, r = install[0], release[0]
        out.append(dict(generation=row['generation'], apply_start=row['start'], apply_end=row['end'],
                        install_start=i['start'], install_end=i['end'],
                        release_start=r['start'], release_end=r['end'],
                        lower_ns=int(r['start']['mono_ns']) - int(i['end']['mono_ns']),
                        upper_ns=int(r['end']['mono_ns']) - int(i['start']['mono_ns']),
                        render_to_installed_ns=int(row['deadline']['installed_by_ns']) - int(row['deadline']['render_start_ns']),
                        verified=row['verified']))
    return out


def matching_guards(at, windows):
    if at is None:
        return []
    return [w for w in windows if int(w['install_start']['wall_ns']) - 2_000
            <= at <= int(w['release_end']['wall_ns']) + 2_000]


def position(at, windows):
    if at is None:
        return 'unobserved', None
    matches = matching_guards(at, windows)
    if len(matches) > 1:
        return 'ambiguous', None
    if not matches:
        return 'outside', None
    w = matches[0]
    inside = int(w['install_end']['wall_ns']) + 2_000 <= at <= int(w['release_start']['wall_ns']) - 2_000
    return ('inside' if inside else 'boundary'), w['generation']


def guard_failures(at, windows):
    """The shared D-120 (i)/(ii) rules for UDP and D-161 TCP loss."""
    pos, generation = position(at, windows)
    matches = matching_guards(at, windows)
    w = matches[0] if len(matches) == 1 else None
    failures = []
    if pos not in ('inside', 'boundary'):
        failures.append('(i): ' + pos)
    if w:
        if w.get('verified') is not True:
            failures.append('(i): Apply not verified')
        if not 0 <= w['upper_ns'] <= 100_000_000:
            failures.append('(ii): guard upper bound > 100 ms or invalid')
        if not 0 <= w['render_to_installed_ns'] <= 100_000_000:
            failures.append('(ii): render-to-installed > 100 ms or invalid')
    return pos, generation, w, failures


def classify_tcp(facts, windows, tcp_errors):
    """Pure D-161 assessment of checked observations; never read captures here."""
    facts = facts or dict(reads=[], requests=[], errors=['missing TCP capture'], connection_errors=[])
    reads = facts['reads']
    common = facts['errors'] + facts['connection_errors']
    requests = {r['sequence']: r for r in facts['requests']}
    guard_losses, losses_by_sequence = {}, {}
    for r in facts['requests']:
        losses = []
        for t in r['lost_transmissions']:
            pos, generation, _, failures = guard_failures(t['node_entry_ns'], windows)
            if t['node_entry_ns'] is None or t['node_exit_ns'] is not None:
                failures.append('lost transmission not dropped at node (entry required, exit absent)')
            row = dict(t, sequence=r['sequence'], classification=pos, generation=generation,
                       failures=failures)
            losses.append(row)
            if not facts['errors'] and not failures:
                guard_losses.setdefault(t['id'], dict(row, sequences=[]))['sequences'].append(r['sequence'])
        losses_by_sequence[r['sequence']] = losses
    attributed, unattributed, failures = [], [], []
    for error in tcp_errors:
        parts = error.split(':', 1)
        sequence = int(parts[0]) if len(parts) == 2 and parts[0].isdigit() else None
        reason = parts[-1]
        if reason != 'TimeoutError':
            failures.append(error)
            continue
        r = requests.get(sequence)
        lost = losses_by_sequence.get(sequence, [])
        conditions = list(common)
        if r is None:
            conditions.append('missing TCP request/echo capture facts')
        else:
            conditions.extend(r['errors'])
            if not r['delivery'] or not r['echo_arrival']:
                conditions.append('missing delivered request or in-order echo')
        if not lost:
            conditions.append('no captured lost transmission of stalled bytes')
        for t in lost:
            conditions.extend(f'transmission {t["id"]}: {f}' for f in t['failures'])
        row = dict(sequence=sequence, lost_transmissions=lost,
                   delivery=r['delivery'] if r else None, echo_arrival=r['echo_arrival'] if r else None)
        if conditions:
            unattributed.append(dict(row, failed_conditions=conditions))
            failures.append(f'{sequence}:TimeoutError unattributable: ' + '; '.join(conditions))
        else:
            attributed.append(row)
    # RST/FIN/second connection fails even when no application recv exposed it.
    failures.extend(facts['connection_errors'])
    stalls = [dict(sequence=r['sequence'], first_origin_ns=r['first_origin_ns'],
                   echo_arrival_ns=r['echo_arrival']['wall_ns'], stall_ns=r['stall_ns'])
              for r in facts['requests'] if r['stall_ns'] is not None]
    return dict(tcp_reads=len(reads), tcp_read_records=reads,
                tcp_timeouts=sum(r.get('outcome') == 'TimeoutError' for r in reads),
                tcp_guard_losses=list(guard_losses.values()), guard_attributed_timeouts=attributed,
                unattributed_timeouts=unattributed,
                longest_tcp_stall=max(stalls, key=lambda s: s['stall_ns']) if stalls and not facts['errors'] else None,
                tcp_evidence_errors=facts['errors'], tcp_failures=failures)


def classify_d120(datagrams, windows, tcp_errors=(), evidence_errors=(), reloads=100, tcp_facts=None):
    """Pure D-120 verdict. No I/O, retries, inferred ingress or missing-data pass.

    The harness separately proves complete sidecar/capture coverage and the
    fixed 100-reload campaign. Per-loss failures include global (iv)/(v).
    """
    errors = list(evidence_errors)
    tcp = classify_tcp(tcp_facts, windows, tcp_errors)
    tcp_failures = tcp['tcp_failures']
    if tcp_facts is not None:
        errors.extend('evidence: TCP: ' + e for e in tcp['tcp_evidence_errors'])
    if reloads != 100:
        errors.append('campaign: requires 100 reloads')
    if not datagrams:
        errors.append('evidence: no datagrams')
    if tcp_failures:
        errors.append('(iv): TCP failure: ' + ', '.join(tcp_failures))
    def success(d):
        return bool(d.get('read') and d['read'].get('outcome') == 'ok'
                    and d['read'].get('response_matches') is True)
    losses = [d for d in datagrams if not success(d)]
    total_losses = len(losses) + len(tcp['guard_attributed_timeouts'])
    if total_losses > 5:
        errors.append('(v): more than 5 UDP losses plus guard-attributed TCP timeouts per 100 reloads')
    rows = []
    for idx, d in enumerate(datagrams):
        if success(d):
            continue
        failures = []
        ingress = d.get('stages', {}).get('node_ingress')
        at = ingress['wall_ns'] if ingress else None
        pos, _, w, failures = guard_failures(at, windows)
        if not d.get('send') or not d.get('read'):
            failures.append('evidence: missing sidecar send/read')
        elif d['read'].get('outcome') != 'TimeoutError':
            failures.append('UDP failure is not a lost echo (timeout)')
        next_d = datagrams[idx + 1] if idx + 1 < len(datagrams) else None
        next_result = 'ok' if next_d and success(next_d) else ('missing' if next_d is None else 'failed')
        if next_result != 'ok':
            failures.append('(iii): next datagram ' + next_result)
        if tcp_failures:
            failures.append('(iv): TCP failure in run')
        if total_losses > 5:
            failures.append('(v): more than 5 UDP losses plus guard-attributed TCP timeouts per 100 reloads')
        rows.append(dict(sequence=d['sequence'], generation=w['generation'] if w else None,
                         bracket_start_ns=w['install_start']['wall_ns'] if w else None,
                         bracket_end_ns=w['release_end']['wall_ns'] if w else None,
                         upper_ns=w['upper_ns'] if w else None,
                         render_to_installed_ns=w['render_to_installed_ns'] if w else None,
                         classification=pos, next_datagram=next_result, failures=failures))
    passed = not errors and not any(r['failures'] for r in rows)
    return dict(passed=passed, verdict='AT10 PASS under D-120' if passed else 'AT10 FAIL',
                reloads=reloads, udp_losses=len(losses), errors=errors, losses=rows,
                total_losses=total_losses, **tcp)


def acceptance_evidence(result, stream_log):
    """Cross-check sidecar against the independently retained text verdicts."""
    events, windows, ds = result['events'], result['windows'], result['datagrams']
    errors, tcp_errors = list(result.get('capture_errors', [])), []
    ends = [e for e in events if e['event'] == 'trace_end']
    if len(ends) != 1 or events[-1] != ends[0]:
        errors.append('evidence: missing final sidecar boundary')
    n = ends[0]['sequence'] if len(ends) == 1 else 0
    logs = [line.split(' ', 1)[1] for line in stream_log.splitlines() if ' ' in line]
    if not logs or logs[0] != 'up' or len(logs) - 1 < n:
        errors.append('evidence: missing stream text verdicts')
    if [w['generation'] for w in windows] != [str(g) for g in range(1000, 1101)]:
        errors.append('evidence: requires initial Apply and generations 1001..1100')
    if any(not w['verified'] for w in windows):
        errors.append('evidence: unverified Apply')
    expected = []
    for seq in range(1, n + 1):
        group = [e for e in events if e['sequence'] == seq and e['event'] != 'trace_end']
        kinds = [e['event'] for e in group]
        if kinds not in (['tcp_send', 'tcp_read', 'udp_send', 'udp_read'], ['tcp_send', 'tcp_read']):
            errors.append(f'evidence: incomplete/duplicate sequence {seq}')
            continue
        tcp = group[1]
        if tcp.get('outcome') or tcp.get('size', 0) <= 0:
            reason = tcp.get('outcome', 'tcp-eof')
            tcp_errors.append(f'{seq}:{reason}')
            expected.append('fail ' + reason)
        elif len(group) != 4:
            errors.append(f'evidence: missing UDP record {seq}')
        else:
            read = group[3]
            expected.append('ok' if read.get('outcome') == 'ok' else 'fail ' + read.get('outcome', 'missing'))
    if any(e['sequence'] < 1 or e['sequence'] > n for e in events):
        errors.append('evidence: sequence outside sidecar boundary')
    if logs[1:n+1] != expected:
        errors.append('evidence: text/sidecar verdict mismatch')
    for d in ds:
        if not d['stages']['node_ingress'] or not d['stages']['origin_egress']:
            errors.append(f'evidence: missing trace record {d["sequence"]}')
    if result['summary']['incomplete_udp_reads']:
        errors.append('evidence: incomplete UDP reads')
    return classify_d120(ds, windows, tcp_errors, errors, tcp_facts=result.get('tcp'))


def verdict_table(verdict):
    lines = [verdict['verdict'],
             'datagram | generation | bracket start ns | bracket end ns | upper ms | render-installed ms | classification | next datagram | failures']
    for r in verdict['losses']:
        upper = f"{r['upper_ns']/1e6:.6f}" if r['upper_ns'] is not None else '-'
        render = f"{r['render_to_installed_ns']/1e6:.6f}" if r['render_to_installed_ns'] is not None else '-'
        lines.append(f"{r['sequence']} | {r['generation']} | {r['bracket_start_ns']} | {r['bracket_end_ns']} | {upper} | {render} | {r['classification']} | {r['next_datagram']} | {', '.join(r['failures']) or 'none'}")
    if not verdict['losses']:
        lines.append('(no UDP losses)')
    lines.append(f"TCP reads: {verdict['tcp_reads']}; TCP timeouts: {verdict['tcp_timeouts']}; "
                 f"guard-attributed timeouts: {len(verdict['guard_attributed_timeouts'])}; "
                 f"combined losses: {verdict['total_losses']}")
    lines.append('longest TCP stall: ' + json.dumps(verdict['longest_tcp_stall'], sort_keys=True))
    for key in ('tcp_read_records', 'tcp_guard_losses', 'guard_attributed_timeouts', 'unattributed_timeouts'):
        for row in verdict[key]:
            lines.append(key + ': ' + json.dumps(row, sort_keys=True))
    lines.extend(verdict['errors'])
    return '\n'.join(lines) + '\n'


def distribution(values):
    values = sorted(values)
    if not values:
        return None
    return dict(min=min(values), median=statistics.median(values),
                p95=values[max(0, (95 * len(values) + 99)//100 - 1)], max=max(values))


def analyze(stream, apply, trace_base):
    events = json_rows(stream)
    windows = guards(json_rows(apply))
    if not windows:
        raise ValueError('no Apply records')
    cap, capture_errors = {}, []
    for role in ('origin', 'node', 'responder'):
        try:
            cap[role] = captures(Path(str(trace_base) + '-' + role + '.txt'))
        except (OSError, ValueError):
            cap[role] = []
            capture_errors.append('evidence: ' + role + ': missing or incomplete UDP capture')
    sends = [e for e in events if e['event'] == 'udp_send']
    reads = {e['sequence']: e for e in events if e['event'] == 'udp_read'}
    datagrams = []
    for idx, sent in enumerate(sends):
        read = reads.get(sent['sequence'])
        start = int(sent['send_wall_ns'])
        # There is only one outstanding UDP send per stream. Attribute late
        # capture arrivals until the next send; preserve mismatched reads.
        end = int(sends[idx+1]['send_wall_ns']) if idx+1 < len(sends) else (int(read['wall_ns']) if read else start) + 200_000_000
        def hop(role, iface, direction, request):
            rows = [r for r in cap[role] if start - 2_000 <= r['wall_ns'] < end
                    and r['interface'] == iface and r['direction'] == direction
                    and (r['destination_port'] == 9000 if request else r['source_port'] == 9000)]
            if len(rows) > 1:
                raise ValueError(f'ambiguous datagram sequence={sent["sequence"]} {role} {iface} {direction}')
            return rows[0] if rows else None
        stages = dict(origin_egress=hop('origin','fv-wg0','Out',True),
                      node_ingress=hop('node','fv-fh','In',True),
                      node_egress=hop('node','fv-ih','Out',True),
                      responder_ingress=hop('responder','fv-ii','In',True),
                      responder_egress=hop('responder','fv-ii','Out',False),
                      node_reply_ingress=hop('node','fv-ih','In',False),
                      node_reply_egress=hop('node','fv-fh','Out',False),
                      origin_reply_ingress=hop('origin','fv-wg0','In',False))
        pos, gen = position(stages['node_ingress']['wall_ns'] if stages['node_ingress'] else None, windows)
        send_pos, send_gen = position(start, windows)
        nearest = min(windows, key=lambda w: abs(start-int(w['install_start']['wall_ns'])))
        datagrams.append(dict(sequence=sent['sequence'],send=sent,read=read,stages=stages,
                              ingress_guard_position=pos,ingress_guard_generation=gen,
                              send_guard_position=send_pos,send_guard_generation=send_gen,
                              nearest_guard_generation=nearest['generation'],
                              send_minus_nearest_install_start_ns=start-int(nearest['install_start']['wall_ns']),
                              read_wait_guard_generations=[w['generation'] for w in windows if read and
                                  int(w['install_start']['wall_ns']) <= int(read['wall_ns']) and
                                  int(w['release_end']['wall_ns']) >= start]))
    timeouts = [d for d in datagrams if d['read'] and d['read'].get('outcome') == 'TimeoutError']
    summary = dict(apply_count=len(windows),verified_applies=sum(w['verified'] for w in windows),
                   datagrams=len(datagrams),udp_timeouts=len(timeouts),
                   timeouts_by_node_ingress=dict(Counter(d['ingress_guard_position'] for d in timeouts)),
                   timeouts_by_send=dict(Counter(d['send_guard_position'] for d in timeouts)),
                   timeouts_without_node_egress=sum(d['stages']['node_egress'] is None for d in timeouts),
                   tcp_timeouts=sum(e['event']=='tcp_read' and e.get('outcome')=='TimeoutError' for e in events),
                   unobserved_node_ingress=sum(d['stages']['node_ingress'] is None for d in datagrams),
                   incomplete_udp_reads=sum(d['read'] is None for d in datagrams),
                   mismatched_echoes=sum(d['read'] is not None and d['read'].get('response_matches') is False for d in datagrams),
                   guard_lower_ns=distribution([w['lower_ns'] for w in windows]),
                   guard_upper_ns=distribution([w['upper_ns'] for w in windows]),
                   render_to_installed_ns=distribution([w['render_to_installed_ns'] for w in windows]),
                   guard_upper_over_100ms=sum(w['upper_ns']>100_000_000 for w in windows),
                   install_over_100ms=sum(w['render_to_installed_ns']>100_000_000 for w in windows),
                   capture_packets={k:len(v) for k,v in cap.items()},
                   max_wall_mono_elapsed_difference_ns=max(abs(
                       (int(w['apply_end']['wall_ns'])-int(w['apply_start']['wall_ns']))-
                       (int(w['apply_end']['mono_ns'])-int(w['apply_start']['mono_ns']))) for w in windows))
    from tcp_d161 import checked_tcp, tcp_facts
    from trace_split import udp_projection
    tcp_captures, tcp_capture_errors = {}, []
    for role in ('origin', 'node', 'responder'):
        path = Path(str(trace_base) + '-' + role + '-headers.txt')
        try:
            text, stats = path.read_text(), path.with_suffix('.err').read_text()
            tcp_captures[role] = checked_tcp(text, stats)
            projected, _, _ = udp_projection(text, stats)
            if projected != Path(str(trace_base) + '-' + role + '.txt').read_text():
                tcp_capture_errors.append(role + ': combined/UDP projection mismatch')
        except FileNotFoundError:
            tcp_capture_errors.append(role + ': missing TCP capture')
        except (OSError, ValueError) as exc:
            tcp_capture_errors.append(role + ': incomplete or unparsed TCP capture: ' + str(exc))
    tcp = tcp_facts(events, tcp_captures, tcp_capture_errors)
    assessment = classify_tcp(tcp, windows, [f'{e["sequence"]}:{e["outcome"]}' for e in events
                                           if e['event'] == 'tcp_read' and e.get('outcome')])
    summary.update({k: assessment[k] for k in ('tcp_reads', 'tcp_guard_losses',
                    'guard_attributed_timeouts', 'longest_tcp_stall', 'unattributed_timeouts')})
    return dict(test_only=True, summary=summary, windows=windows, datagrams=datagrams, events=events, tcp=tcp, capture_errors=capture_errors)


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('--stream', type=Path, required=True)
    p.add_argument('--apply', type=Path, required=True)
    p.add_argument('--trace-base', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--d120', action='store_true')
    p.add_argument('--stream-log', type=Path)
    args = p.parse_args()
    try:
        result = analyze(args.stream, args.apply, args.trace_base)
        if args.d120:
            if args.stream_log is None:
                raise ValueError('D-120 requires the stream text verdict log')
            result['d120'] = acceptance_evidence(result, args.stream_log.read_text())
    except (OSError, ValueError, KeyError, TypeError) as exc:
        if not args.d120:
            raise
        result = dict(d120=classify_d120([], [], evidence_errors=[f'evidence: {exc}']))
    args.output.write_text(json.dumps(result, indent=2) + '\n')
    if args.d120:
        print(verdict_table(result['d120']), end='')
        sys.exit(0 if result['d120']['passed'] else 1)
    print(json.dumps(result['summary'], indent=2))
