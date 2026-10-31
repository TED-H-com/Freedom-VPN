#!/usr/bin/env python3
"""TEST ONLY header-only native endpoint correlation; never reads raw payloads."""
import argparse
from collections import Counter
from datetime import datetime
from decimal import Decimal
import json
from pathlib import Path


def wall_ns(value):
    whole, _, fraction = value.removesuffix('Z').partition('.')
    return int(datetime.fromisoformat(whole+'+00:00').timestamp())*10**9 + int((fraction+'0'*9)[:9])


def clocks(value):
    if isinstance(value, dict):
        if 'mono_before_ns' in value:
            yield value
        else:
            for item in value.values():
                yield from clocks(item)
    elif isinstance(value, list):
        for item in value:
            yield from clocks(item)


def offsets(evidence, role):
    pairs = list(clocks(evidence['points'][role]))
    for sample in evidence['samples']:
        pairs.extend(clocks(sample['points'][role]))
    assert pairs and len({p['boot_id'] for p in pairs}) == 1
    lower = min(int(p['mono_before_ns'])-wall_ns(p['wall']) for p in pairs)
    upper = max(int(p['mono_after_ns'])-wall_ns(p['wall']) for p in pairs)
    assert upper-lower < 1000000, 'clock offset moved by >=1 ms; do not interpolate'
    return dict(boot_id=pairs[0]['boot_id'], pairs=len(pairs), offset_lower_ns=str(lower),
                offset_upper_ns=str(upper), measured_offset_span_ns=str(upper-lower),
                assumption='No intervening wall-clock step; empirical offset envelope, not hardware packet timestamps. Pcap microsecond quantization adds 1000 ns.')


def bound(row, offset):
    timestamp = int(Decimal(row['timestamp'])*10**9)
    return dict(packet=row['packet'], timestamp=row['timestamp'],
                mono_lower_ns=str(timestamp+int(offset['offset_lower_ns'])),
                mono_upper_ns=str(timestamp+int(offset['offset_upper_ns'])+1000))


def match_datagrams(sent, received):
    available = Counter(p['ip_sha256'] for p in received)
    sizes = {}
    missing = []
    for p in sent:
        label = str(p['ip_length'])
        if p.get('extensions'):
            ext = next((e for e in p['extensions'] if e['type'] == 44), None)
            if ext:
                label += ':fragment@'+str(ext['fragment_offset'])
        count = sizes.setdefault(label, dict(sent=0, arrived=0, absent=0))
        count['sent'] += 1
        if available[p['ip_sha256']]:
            available[p['ip_sha256']] -= 1
            count['arrived'] += 1
        else:
            count['absent'] += 1
            missing.append(p['packet'])
    return dict(sizes=sizes, absent_sender_packets=missing, unmatched_receiver_packets=sum(available.values()))


def first_gap(sent, received):
    # Compare byte-range coverage, not equal segment sizes (GRO/GSO may differ).
    ranges = sorted((r['tcp']['seq'], r['tcp']['seq']+r['tcp']['length']) for r in received if r['tcp']['length'] > 0)
    for a, b in sorted(set((r['tcp']['seq'], r['tcp']['seq']+r['tcp']['length']) for r in sent if r['tcp']['length'] > 0)):
        cursor = a
        for lo, hi in ranges:
            if hi <= cursor:
                continue
            if lo > cursor:
                break
            cursor = max(cursor, hi)
            if cursor >= b:
                break
        if cursor < b:
            next_received = min((lo for lo, _ in ranges if lo > cursor), default=b)
            return [cursor, min(b, next_received)]
    return None


def analyze(run, record):
    doc = json.loads(record.read_text())
    notes = {o['name']: o['detail'] for o in doc['observations']}
    evidence = json.loads(notes['capture'])
    decoded = json.loads(notes['decode'])
    role_order = ['member', 'node', 'origin']
    packets = {role: [json.loads(line) for line in (run/'headers'/doc['id']/(role+'.headers.jsonl')).read_text().splitlines()] for role in role_order}
    result = dict(id=doc['id'], run_id=doc['run_id'], source_commit=doc['source_commit'], status=doc['status'], clock_offsets={}, capture_checks={}, ptb={}, fragmentation_needed={})
    for role, check in zip(role_order, decoded):
        p = evidence['points'][role]
        complete = p['statistics_ok'] and p['process_stopped'] and p['kernel_drops'] == 0 and p['captured'] == check['records'] == len(packets[role]) and check['header_errors'] == 0 and check['file_complete'] and check['snap_truncated_records'] == 0
        result['capture_checks'][role] = dict(complete=complete, decoded=check['records'], captured=p['captured'], kernel_drops=p['kernel_drops'])
        assert complete, f'{doc["id"]}/{role}: incomplete capture'
        result['clock_offsets'][role] = offsets(evidence, role)
        result['ptb'][role] = [r['packet'] for r in packets[role] if r.get('protocol') == 58 and r.get('icmp', {}).get('type') == 2]
        result['fragmentation_needed'][role] = [r['packet'] for r in packets[role] if r.get('protocol') == 1 and r.get('icmp', {}).get('type') == 3 and r.get('icmp', {}).get('code') == 4]
    if doc['id'] == 'initial-selection':
        samples = evidence['samples']
        result['selection_samples'] = [s['label'] for s in samples]
        result['hops'] = {}
        before = next(s for s in samples if s['label'] == 'before-first-transfer')
        for role in role_order:
            indexes = {p['ifindex']: p['ifname'] for p in (before['points'][role]['links'] or [])}
            selected = [p for p in packets[role] if p.get('tcp', {}).get('src_port') == 20001 or p.get('tcp', {}).get('dst_port') == 20001]
            result['hops'][role] = [dict(bound(p, result['clock_offsets'][role]), interface=indexes.get(p['interface_index'], 'unknown'), packet_type=p['packet_type'], tcp=p['tcp']) for p in selected]
        result['member_syns'] = [p for p in result['hops']['member'] if p['interface'] == 'fvnb0' and p['packet_type'] == 4 and p['tcp']['flags'] == 2]
        result['transfer'] = json.loads(notes['transfer'])
        result['scope'] = 'Fresh first-use observation only. Historical five packets cannot be reconstructed. See inner/outer capture and readback bounds before assigning any fault.'
        return result
    before = next(s for s in evidence['samples'] if s['label'] == 'before-bulk')['points']
    after = next(s for s in evidence['samples'] if s['label'] == 'after-bulk')['points']
    start = min(wall_ns(s['end']['wall']) for s in before.values())
    stop = max(wall_ns(s['start']['wall']) for s in after.values())
    bulk = {role: [p for p in rows if start <= int(Decimal(p['timestamp'])*10**9) <= stop] for role, rows in packets.items()}
    result['selected_before'] = {r: dict(local=before[r]['local'], remote=before[r]['remote'], wireguard_endpoint=before[r]['wireguard_endpoint'], routes=before[r]['routes']) for r in ['member','node']}
    result['transfer'] = json.loads(notes['transfer'])
    for sender,receiver,label in [('node','member','reverse'),('member','node','forward')]:
        endpoint = before[sender]['remote']
        dev = before[sender]['routes']['peer_route'][0]['dev']
        index = next(p['ifindex'] for p in before[sender]['links'] if p['ifname'] == dev)
        rdev = before[receiver]['routes']['peer_route'][0]['dev']
        rindex = next(p['ifindex'] for p in before[receiver]['links'] if p['ifname'] == rdev)
        src, dst = before[receiver]['remote']['address'], endpoint['address']
        tx = [p for p in bulk[sender] if p.get('src') == src and p.get('dst') == dst and p['interface_index'] == index and p['packet_type'] == 4]
        rx = [p for p in bulk[receiver] if p.get('src') == src and p.get('dst') == dst and p['interface_index'] == rindex and p['packet_type'] != 4]
        result[label] = match_datagrams(tx, rx)
        result[label]['sender_interface'] = dev
        result[label]['receiver_interface'] = rdev
        result[label]['absent_bounds'] = [bound(p,result['clock_offsets'][sender]) for p in tx if p['packet'] in result[label]['absent_sender_packets']]
    origin = [p for p in bulk['origin'] if p.get('tcp', {}).get('src_port') == 9446 and p['packet_type'] == 4]
    received = [p for p in bulk['member'] if p.get('tcp', {}).get('src_port') == 9446 and p['packet_type'] != 4]
    gap = first_gap(origin, received)
    result['first_missing_tcp_range'] = gap
    if gap:
        result['missing_range_receipts'] = {}
        for role in role_order:
            rows = [p for p in bulk[role] if p.get('tcp',{}).get('src_port') == 9446 and p['tcp']['seq'] <= gap[0] < p['tcp']['seq']+p['tcp']['length']]
            index_names = {p['ifindex']:p['ifname'] for p in (before[role]['links'] or [])}
            result['missing_range_receipts'][role] = [dict(bound(p,result['clock_offsets'][role]), interface=index_names.get(p['interface_index'],'unknown'), packet_type=p['packet_type'], tcp=p['tcp']) for p in rows]
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('run', type=Path)
    args = parser.parse_args()
    rows = []
    for name in ['initial-selection','matrix4','matrix6','matrixDual','matrix1280','ipv6-asymmetric','ipv6-explicit1280','ipv6-explicit-repeat1','ipv6-explicit-repeat2']:
        if not (args.run/(name+'.json')).exists():
            continue
        rows.append(analyze(args.run, args.run/(name+'.json')))
    out = args.run/'headers'/'analysis.json'
    out.write_text(json.dumps(dict(schema='native-capture-analysis-v1', rows=rows),indent=2)+'\n')
    print(out)

if __name__ == '__main__':
    main()
