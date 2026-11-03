"""D-146 native TCP qualification from original, payload-free capture headers.

Uses the read-only owner decoder. No traffic, reconnect, MSS or MTU mutation.
A missing or ambiguous join never qualifies the receive-only row.
"""
import argparse
from collections import Counter
from decimal import Decimal
import hashlib
import json
from pathlib import Path
import sys

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
import decode_capture as decoder
import analyze_native_capture as native


def mss(packet):
    values = [o['mss'] for o in packet['tcp']['options'] if 'mss' in o]
    return values[0] if len(values) == 1 else None


def clamp_pair(pair, direction, device):
    rule, accept = pair['clamp'], pair['accept']
    expr, admitted = rule['expr'], accept['expr']
    key = {'tcp option': {'name': 'maxseg', 'field': 'size'}}
    flags = {'match': {'op': '==', 'left': {'&': [{'payload': {'protocol': 'tcp', 'field': 'flags'}}, 'syn']}, 'right': 'syn'}}
    field = 'saddr' if direction == 'upload' else 'daddr'
    return (rule['chain'] == accept['chain'] == 'forward' and
            pair['accept_index'] == pair['clamp_index'] + 1 and
            len(expr) == len(admitted) == 6 and expr[:3] == admitted[:3] and
            expr[0]['match']['left'] == {'meta': {'key': 'iifname'}} and
            expr[1]['match']['left'] == {'meta': {'key': 'oifname'}} and
            expr[2]['match']['left'] == {'payload': {'protocol': 'ip', 'field': field}} and
            all(e['match']['op'] == '==' for e in expr[:3]) and
            expr[3:] == [flags, {'match': {'op': '>', 'left': key, 'right': 1160}}, {'mangle': {'key': key, 'value': 1160}}] and
            admitted[3] == {'counter': ('u_' if direction == 'upload' else 'd_') + device.replace('-', '')} and
            admitted[-1] == {'accept': None})


def qualify(data, packets, decoded):
    capture = data['capture']
    result = dict(schema='members-d146-v1', run_id=data['run_id'], source_commit=data['source_commit'],
                  row=data['row'], selected_before=data['selected_before'], selected_after=data['selected_after'],
                  rules=data['rules'], input_applied_at=data['input_applied_at'], bulk_started_at=data['bulk_started_at'],
                  bulk_ended_at=data['bulk_ended_at'], transfer=data['transfer'],
                  upload_predicate=data['upload_predicate'], original_byte_predicate=data['original_byte_predicate'],
                  smaller_mss_control='not exercised: the unchanged bulk fixture has no TCP_MAXSEG input; netlab 34 control stands',
                  udp_quic='receive-only no-PTB limitation under D-146; no TCP result upgrades UDP or QUIC',
                  capture_checks={}, endpoint_mss={}, no_ptb={}, checks={}, reasons=[])
    checks = result['checks']
    for role in ('member', 'node', 'origin'):
        p, d = capture['points'][role], decoded[role]
        complete = (p['statistics_ok'] and p['process_stopped'] and p['kernel_drops'] == 0 and
                    p['captured'] == d['records'] == len(packets[role]) and d['file_complete'] and
                    d['header_errors'] == d['snap_truncated_records'] == 0)
        result['capture_checks'][role] = dict(complete=complete, decode=d, captured=p['captured'], kernel_drops=p['kernel_drops'])
        result['no_ptb'][role] = dict(
            ipv4_fragmentation_needed=[p['packet'] for p in packets[role] if p.get('protocol') == 1 and p.get('icmp', {}).get('type') == 3 and p['icmp']['code'] == 4],
            ipv6_packet_too_big=[p['packet'] for p in packets[role] if p.get('protocol') == 58 and p.get('icmp', {}).get('type') == 2])
    checks['complete_captures'] = all(x['complete'] for x in result['capture_checks'].values())
    checks['no_ptb_any_capture_point'] = all(not ids for point in result['no_ptb'].values() for ids in point.values())
    before = next(s['points'] for s in capture['samples'] if s['label'] == 'before-bulk')
    after = next(s['points'] for s in capture['samples'] if s['label'] == 'after-bulk')
    rules = data['rules']
    pairs = rules.get('pairs', {})
    checks['serving_rules_before_new_handshake'] = (rules['available'] and rules['serving'] and
        all(direction in pairs and clamp_pair(pairs[direction], direction, data['device_id']) for direction in ('upload', 'download')))
    checks['both_byte_predicates'] = data['original_byte_predicate'] and data['upload_predicate']
    checks['selected_family_unchanged_direct'] = (data['selected_before']['chosen_family'] in ('ipv4', 'ipv6') and
        data['selected_before'] == data['selected_after'] and data['selected_before']['connected'] and
        data['selected_before']['connection_type'].lower() in ('p2p', 'direct'))
    # Broad window contains every command, but stops before any capture process.
    start = min(native.wall_ns(s['end']['wall']) for s in before.values())
    end = max(native.wall_ns(s['start']['wall']) for s in after.values())
    window = {role: [p for p in rows if start <= pcap_ns(p['timestamp']) <= end] for role, rows in packets.items()}
    port = data['transfer']['endpoint']['local_port']
    mi = next(p['ifindex'] for p in before['member']['links'] if p['ifname'] == 'fvnb0')
    def syns(role, sent, ack):
        return [p for p in window[role] if 'tcp' in p and (p['packet_type'] == 4) == sent and
                (role != 'member' or p['interface_index'] == mi) and
                p['tcp']['flags'] & 0x12 == (0x12 if ack else 2) and
                p['tcp']['src_port' if ack else 'dst_port'] == 9446 and
                p['tcp']['dst_port' if ack else 'src_port'] == port]
    ends = {'member_syn_sent': syns('member', True, False), 'origin_syn_received': syns('origin', False, False),
            'origin_synack_sent': syns('origin', True, True), 'member_synack_received': syns('member', False, True)}
    for name, rows in ends.items():
        result['endpoint_mss'][name] = [dict(packet=p['packet'], timestamp=p['timestamp'], tcp=p['tcp'], mss=mss(p)) for p in rows]
    # Retransmissions may repeat an advertisement; require one unambiguous ISN.
    checks['matched_handshake'] = all(ends.values())
    if checks['matched_handshake']:
        synseq = {p['tcp']['seq'] for name in ('member_syn_sent', 'origin_syn_received') for p in ends[name]}
        ackseq = {p['tcp']['seq'] for name in ('origin_synack_sent', 'member_synack_received') for p in ends[name]}
        checks['matched_handshake'] = (len(synseq) == len(ackseq) == 1 and
            all(p['tcp']['ack'] == (next(iter(synseq)) + 1) % 2**32 for name in ('origin_synack_sent', 'member_synack_received') for p in ends[name]))
    cutoff = max(native.wall_ns(data['input_applied_at']), native.wall_ns(rules['ended_at']))
    checks['handshake_after_mtu_and_serving_rules'] = (checks['matched_handshake'] and
        cutoff < native.wall_ns(data['bulk_started_at']) and
        all(pcap_ns(p['timestamp']) > cutoff for rows in ends.values() for p in rows))
    checks['both_advertisements_clamped'] = (checks['matched_handshake'] and
        all(mss(p) is not None and mss(p) > 1160 for name in ('member_syn_sent', 'origin_synack_sent') for p in ends[name]) and
        all(mss(p) == 1160 for name in ('origin_syn_received', 'member_synack_received') for p in ends[name]))
    # Selected raw outer IP datagrams match by hash and multiplicity, not time.
    for sender, receiver, label in [('node', 'member', 'reverse'), ('member', 'node', 'forward')]:
        s, r = before[sender], before[receiver]
        sdev, rdev = s['routes']['peer_route'][0]['dev'], r['routes']['peer_route'][0]['dev']
        si = next(p['ifindex'] for p in s['links'] if p['ifname'] == sdev)
        ri = next(p['ifindex'] for p in r['links'] if p['ifname'] == rdev)
        src, dst = r['remote']['address'], s['remote']['address']
        tx = [p for p in window[sender] if p.get('src') == src and p.get('dst') == dst and p['interface_index'] == si and p['packet_type'] == 4]
        rx = [p for p in window[receiver] if p.get('src') == src and p.get('dst') == dst and p['interface_index'] == ri and p['packet_type'] != 4]
        result[label] = dict(native.match_datagrams(tx, rx), sender_interface=sdev, receiver_interface=rdev,
            sender_count=len(tx), receiver_count=len(rx), sender_max_ip_length=max((p['ip_length'] for p in tx), default=0),
            receiver_max_ip_length=max((p['ip_length'] for p in rx), default=0),
            ip_families=sorted({p['ip_version'] for p in tx + rx}),
            sender_packets=[dict(packet=p['packet'], ip_length=p['ip_length'], ip_sha256=p['ip_sha256'], timestamp=p['timestamp']) for p in tx],
            receiver_packets=[dict(packet=p['packet'], ip_length=p['ip_length'], ip_sha256=p['ip_sha256'], timestamp=p['timestamp']) for p in rx])
    reverse = result['reverse']
    checks['every_reverse_datagram_fits_and_arrives'] = (reverse['sender_count'] > 0 and reverse['sender_count'] == reverse['receiver_count'] and
        not reverse['absent_sender_packets'] and reverse['unmatched_receiver_packets'] == 0 and
        0 < reverse['sender_max_ip_length'] <= 1280 and 0 < reverse['receiver_max_ip_length'] <= 1280 and
        reverse['ip_families'] == [4 if data['selected_before']['chosen_family'] == 'ipv4' else 6])
    result['reverse_tcp_payload_lengths'] = dict(Counter(str(p['tcp']['length']) for p in window['origin'] if p.get('tcp', {}).get('src_port') == 9446 and p['tcp']['dst_port'] == port and p['packet_type'] == 4 and p['tcp']['length'] > 0))
    result['scope'] = 'Selected endpoint datagrams in the original bulk command window; opaque IP hashes and multiplicity join sender to ingress. No encrypted datagram is assigned to an inner stream by timing. No relay, real-platform or extra-encapsulation qualification.'
    result['reasons'] = [key for key, ok in checks.items() if not ok]
    result['qualified'] = all(checks.values())
    return result


# Pcap fractions are decimal strings, never binary floats.
def pcap_ns(value):
    return int(Decimal(value) * 10**9)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--input', type=Path, required=True)
    p.add_argument('--private', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    args = p.parse_args()
    data = json.loads(args.input.read_text())
    args.out.mkdir(parents=True, exist_ok=True)
    (args.out/'input.json').write_text(json.dumps(data, indent=2)+'\n')
    try:
        packets, decoded = {}, {}
        for role in ('member', 'node', 'origin'):
            source = args.private/(role+'.pcap')
            receipt = data['capture']['points'][role]['private_pcap_receipt']
            assert source.stat().st_size == int(receipt['bytes']) and hashlib.sha256(source.read_bytes()).hexdigest() == receipt['sha256'], 'capture receipt mismatch'
            output = args.out/(role+'.headers.jsonl')
            decoded[role] = decoder.decode(source, output, {}, stable_addresses=True, wireguard=True)
            packets[role] = [json.loads(line) for line in output.read_text().splitlines()]
        result = qualify(data, packets, decoded)
    except (AssertionError, KeyError, StopIteration, ValueError, OSError, TypeError):
        result = dict(qualified=False, run_id=data['run_id'], row=data['row'], reasons=['incomplete_or_unjoinable_evidence'],
                      scope='Original input and decoded headers retained; unavailable evidence never upgrades a receive-only row.')
    (args.out/'analysis.json').write_text(json.dumps(result, indent=2)+'\n')
    (args.out/'SHA256SUMS').write_text(''.join(hashlib.sha256(f.read_bytes()).hexdigest()+'  '+f.name+'\n' for f in sorted(args.out.iterdir()) if f.is_file() and f.name != 'SHA256SUMS'))
    print(json.dumps(result))


if __name__ == '__main__':
    main()
