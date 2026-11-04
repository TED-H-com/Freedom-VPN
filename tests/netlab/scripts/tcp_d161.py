"""TEST ONLY TCP header correlation, before the pure D-120/D-161 classifier.

No payload is decoded. Synthetic t<sequence> newline lengths locate byte ranges;
SYN sequence numbers anchor both streams. Exact echo identities come only from
the stream's existing match/stale metadata. Unsupported/ambiguous evidence fails
closed; timestamps never stand in for a missing packet observation.
"""
from collections import defaultdict
from decimal import Decimal
import re


ROUTES = {
    'request': [('origin', 'fv-wg0', 'Out'), ('node', 'fv-fh', 'In'),
                ('node', 'fv-ih', 'Out'), ('responder', 'fv-ii', 'In')],
    'echo': [('responder', 'fv-ii', 'Out'), ('node', 'fv-ih', 'In'),
             ('node', 'fv-fh', 'Out'), ('origin', 'fv-wg0', 'In')],
}


def parse_tcp(block):
    lines = block.splitlines()
    head = re.fullmatch(r'(\d+\.\d+)\s+(\S+)\s+(In|Out)\s+IP \(.*\bid (\d+), '
                        r'offset 0, flags \[DF\], proto TCP \(6\), length \d+\)', lines[0])
    body = re.fullmatch(
        r'\s*(\d+\.\d+\.\d+\.\d+)\.(\d+) > (\d+\.\d+\.\d+\.\d+)\.(\d+): '
        r'Flags \[([FSRP.UWE]*)\], (?:cksum 0x[0-9a-f]+ \([^)]*\), )?'
        r'(?:seq (\d+)(?::(\d+))?, )?(?:ack (\d+), )?win (\d+), '
        r'(?:options \[([^\]]*)\], )?length (\d+)\s*', lines[1]) if len(lines) == 2 else None
    if not head or not body:
        raise ValueError('unparsed TCP header')
    size = int(body[11])
    seq = int(body[6]) if body[6] else None
    end = int(body[7]) if body[7] else seq
    if size and (seq is None or end is None or end - seq != size):
        raise ValueError('unparsed TCP byte range (including sequence wrap)')
    ts = re.search(r'\bTS val (\d+) ecr (\d+)', body[10] or '')
    return dict(wall_ns=int(Decimal(head[1]) * 10**9), interface=head[2],
                direction=head[3], ip_id=int(head[4]), source=body[1], source_port=int(body[2]),
                destination=body[3], destination_port=int(body[4]), flags=body[5],
                seq_start=seq, seq_end=end, ack=int(body[8]) if body[8] else None,
                size=size, ts_val=int(ts[1]) if ts else None)


def checked_tcp(text, stats):
    # Reuse exactly the combined projection's count/drop/unparsed checks.
    from trace_split import checked_blocks
    _, blocks = checked_blocks(text, stats)
    return [parse_tcp(b) for b in blocks]


def complete_arrival(transmissions, low, high):
    """First complete in-order arrival of [low, high), including segmentation."""
    covered, used = low, []
    arrivals = sorted((t for t in transmissions if t['delivered']),
                      key=lambda t: t['far_side_ns'])
    intervals = []
    for t in arrivals:
        intervals.append((t['seq_start'], t['seq_end']))
        used.append(t['id'])
        for a, b in sorted(intervals):
            if a <= covered:
                covered = max(covered, b)
        if covered >= high:
            return dict(wall_ns=t['far_side_ns'], transmission=t, transmissions=used)
    return None


def tcp_facts(events, captures, capture_errors=()):
    reads = [e for e in events if e['event'] == 'tcp_read']
    result = dict(reads=reads, requests=[], transmissions=[], errors=list(capture_errors),
                  connection_errors=[], capture_packets={k: len(v) for k, v in captures.items()})
    if capture_errors:
        return result
    errors, connection = result['errors'], result['connection_errors']
    ends = [e for e in events if e['event'] == 'trace_end']
    sends = [e for e in events if e['event'] == 'tcp_send']
    if len(ends) != 1 or not sends or not all('wall_ns' in e for e in events):
        errors.append('missing TCP sidecar boundary/timestamps')
        return result
    end = int(ends[0]['wall_ns'])
    paths = {p: (direction, index) for direction, route in ROUTES.items()
             for index, p in enumerate(route)}
    groups, ports, syns = defaultdict(lambda: defaultdict(list)), set(), defaultdict(set)
    for role, rows in captures.items():
        for row in rows:
            if row['wall_ns'] > end:
                continue
            path = (role, row['interface'], row['direction'])
            if path not in paths:
                errors.append('unexpected TCP collector interface/direction')
                continue
            direction, index = paths[path]
            request = direction == 'request'
            client = (row['source'], row['source_port']) if request else (row['destination'], row['destination_port'])
            server = (row['destination'], row['destination_port']) if request else (row['source'], row['source_port'])
            expected_client = '100.64.1.1' if path in (ROUTES['request'][:2] + ROUTES['echo'][2:]) else '192.0.2.37'
            ports.add(client[1])
            if client[0] != expected_client or server != ('203.0.113.10', 9001):
                connection.append('second or unexpected connection endpoint')
            if 'R' in row['flags']:
                connection.append('reset before trace end')
            if 'F' in row['flags']:
                connection.append('FIN before trace end')
            if 'S' in row['flags']:
                if row['wall_ns'] >= int(sends[0]['wall_ns']):
                    connection.append('second handshake after stream start')
                syns[direction].add(row['seq_start'])
            identity = (direction, row['ip_id'], row['seq_start'], row['seq_end'],
                        row['ack'], row['flags'], row['ts_val'])
            groups[identity][index].append(row)
    if len(ports) != 1 or any(len(syns[d]) != 1 or None in syns[d] for d in ROUTES):
        connection.append('missing handshake or second connection')
    # A single SYN/SYN-ACK must cross all four observations. A restart with a
    # different ISN, even on the same four-tuple, cannot be accepted.
    for direction in ROUTES:
        if not any(k[0] == direction and 'S' in k[5] and all(len(g[i]) == 1 for i in range(4))
                   for k, g in groups.items()):
            connection.append('incomplete handshake path')
    transmissions = result['transmissions']
    for key, hops in groups.items():
        sample = next(iter(hops.values()))[0]
        if not sample['size']:
            continue
        if any(len(rows) != 1 for rows in hops.values()):
            errors.append('ambiguous TCP transmission identity')
            continue
        if not hops[0]:
            errors.append('TCP transmission missing origin observation')
            continue
        times = [hops[i][0]['wall_ns'] if hops[i] else None for i in range(4)]
        observed = [t for t in times if t is not None]
        if any(b < a - 2_000 for a, b in zip(observed, observed[1:])):
            errors.append('TCP path timestamp order')
        if times[3] is not None and any(t is None for t in times):
            errors.append('delivered TCP transmission missing intermediate observation')
        transmissions.append(dict(id=len(transmissions) + 1, direction=key[0],
                                  seq_start=sample['seq_start'], seq_end=sample['seq_end'],
                                  ip_id=sample['ip_id'], origin_ns=times[0], node_entry_ns=times[1],
                                  node_exit_ns=times[2], far_side_ns=times[3], delivered=times[3] is not None))
    result['connection_errors'] = sorted(set(connection))
    if any(len(syns[d]) != 1 or None in syns[d] for d in ROUTES):
        return result
    bases = {d: next(iter(syns[d])) + 1 for d in ROUTES}
    offset = 0
    mapped = set()
    for send in sends:
        sequence = send['sequence']
        size = len(f't{sequence}\n'.encode())
        ranges = {d: (b + offset, b + offset + size) for d, b in bases.items()}
        relevant = [t for t in transmissions if t['seq_start'] < ranges[t['direction']][1]
                    and t['seq_end'] > ranges[t['direction']][0]]
        mapped.update(t['id'] for t in relevant)
        request = [t for t in relevant if t['direction'] == 'request']
        echo = [t for t in transmissions if t['direction'] == 'echo'
                and t['seq_start'] < ranges['echo'][1]]
        first = min((t['origin_ns'] for t in request), default=None)
        issues = []
        if first is None or first < int(send['wall_ns']) - 2_000:
            issues.append('missing or premature request origin transmission')
        delivery = complete_arrival(request, *ranges['request'])
        # Include all previous echo bytes: a higher sequence arriving out of
        # order cannot stand in for in-order delivery of the timed-out echo.
        echoed = complete_arrival(echo, bases['echo'], ranges['echo'][1])
        confirmed = [r for r in reads if r.get('response_sequence') == sequence
                     and r.get('echo_status') in ('match', 'stale')
                     and r.get('size') == size and int(r['wall_ns']) <= end]
        if not delivery or not echoed or not confirmed:
            issues.append('bytes/echo not delivered in order and confirmed before trace end')
        elif int(confirmed[0]['wall_ns']) < echoed['wall_ns'] - 2_000:
            issues.append('echo metadata precedes captured arrival')
        result['requests'].append(dict(sequence=sequence, ranges=ranges, first_origin_ns=first,
                                       delivery=delivery, echo_arrival=echoed, errors=issues,
                                       lost_transmissions=[t for t in relevant if not t['delivered']],
                                       stall_ns=echoed['wall_ns'] - first if echoed and first is not None else None))
        offset += size
    if mapped != {t['id'] for t in transmissions}:
        errors.append('TCP bytes outside sidecar stream')
    return result
