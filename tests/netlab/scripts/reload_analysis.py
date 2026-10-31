#!/usr/bin/env python3
"""Correlate TEST ONLY UDP metadata with independent header captures and Apply.

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


def position(at, windows):
    if at is None:
        return 'unobserved', None
    for w in windows:
        if int(w['install_end']['wall_ns']) + 2_000 <= at <= int(w['release_start']['wall_ns']) - 2_000:
            return 'inside', w['generation']
    for w in windows:
        if int(w['install_start']['wall_ns']) - 2_000 <= at <= int(w['release_end']['wall_ns']) + 2_000:
            return 'boundary', w['generation']
    return 'outside', None


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
    cap = {role: captures(Path(str(trace_base) + '-' + role + '.txt'))
           for role in ('origin', 'node', 'responder')}
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
    return dict(test_only=True, summary=summary, windows=windows, datagrams=datagrams)


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('--stream', type=Path, required=True)
    p.add_argument('--apply', type=Path, required=True)
    p.add_argument('--trace-base', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    args = p.parse_args()
    result = analyze(args.stream, args.apply, args.trace_base)
    args.output.write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(result['summary'], indent=2))
