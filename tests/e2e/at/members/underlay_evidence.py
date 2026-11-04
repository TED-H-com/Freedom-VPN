"""Read the existing namespace-wide capture; export only underlay headers.

No new capture hook, traffic, predicate or retry. Stable labels match the shared
native decoder. Exact-IP hash joins are observations, not decryption claims.
"""
from collections import Counter
import argparse
import hashlib
import json
from pathlib import Path
import re
import sys

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
import decode_capture as decoder


def packet_view(p, interfaces, ports):
    interface = interfaces.get(p.get('interface_index'))
    if not interface or interface in ('lo', 'fvnb0', 'fvmb0'):
        return None
    udp = p.get('udp', {})
    if p.get('protocol') not in (1, 58) and not (p.get('protocol') == 17 and ports.intersection((udp.get('src_port'), udp.get('dst_port')))):
        return None
    if any(not re.fullmatch(r'addr-[0-9a-f]{16}', p.get(k, '')) for k in ('src', 'dst')):
        return None
    if not re.fullmatch(r'[0-9]+\.[0-9]+', str(p.get('timestamp', ''))):
        return None
    out = {k: p[k] for k in ('packet', 'ip_version', 'ip_length', 'packet_type', 'interface_index', 'protocol') if isinstance(p.get(k), int)}
    out.update(timestamp=p['timestamp'], src=p['src'], dst=p['dst'], interface=interface,
               direction='egress' if p.get('packet_type') == 4 else 'ingress')
    if re.fullmatch(r'[0-9a-f]{64}', p.get('ip_sha256', '')):
        out['ip_sha256'] = p['ip_sha256']
    for field, keys in [('udp', ('src_port', 'dst_port', 'length')), ('icmp', ('type', 'code', 'mtu')), ('wireguard', ('type', 'length'))]:
        if field in p:
            out[field] = {k: p[field][k] for k in keys if isinstance(p[field].get(k), int)}
    return out


def join_direction(sender, receiver):
    counts = Counter(p['ip_sha256'] for p in receiver if 'ip_sha256' in p)
    joins = []
    for p in sender:
        h = p.get('ip_sha256')
        found = bool(h and counts[h])
        if found:
            counts[h] -= 1
        joins.append(dict(sender_packet=p['packet'], receiver_seen=found, fingerprint_available=bool(h)))
    return joins


def analyze(data, private):
    cap = data['capture']
    ports = {51820}  # fixture default; all observed endpoint ports are added
    interfaces = {'member': {}, 'node': {}}
    addresses = {'member': set(), 'node': set()}
    for s in cap['samples']:
        for role in interfaces:
            point = s['points'][role]
            opposite = 'node' if role == 'member' else 'member'
            for key, owner in [('local', role), ('remote', opposite)]:
                label = point.get(key, {}).get('address', '')
                if re.fullmatch(r'addr-[0-9a-f]{16}', label):
                    addresses[owner].add(label)
            for link in point.get('links', []):
                if re.fullmatch(r'(eth[0-9]+|fv[a-zA-Z0-9_-]*|lo)', link['ifname']):
                    interfaces[role][link['ifindex']] = link['ifname']
            for key in ('local', 'remote', 'wireguard_endpoint'):
                port = point.get(key, {}).get('port', '')
                if str(port).isdigit():
                    ports.add(int(port))
    result = dict(ports=sorted(ports), capture_id=cap['id'], points={}, directions={}, scope='All underlay interfaces in existing -i any SLL2 captures, UDP involving either WireGuard port plus ICMP/ICMPv6. Raw packet contents, keys and raw addresses are never exported. Exact-IP hash matches are per occurrence; a mismatch can include route header changes.')
    for role, links in interfaces.items():
        source = private/(role+'.pcap')
        receipt = cap['points'][role]['private_pcap_receipt']
        assert str(source.stat().st_size) == receipt['bytes']
        assert hashlib.sha256(source.read_bytes()).hexdigest() == receipt['sha256']
        headers = private/(role+'.f23-headers.jsonl')
        decoded = decoder.decode(source, headers, {}, stable_addresses=True, wireguard=True)
        packets = []
        with headers.open() as stream:
            for line in stream:
                p = packet_view(json.loads(line), links, ports)
                if p is not None:
                    packets.append(p)
        stats = cap['points'][role]
        result['points'][role] = dict(packets=packets, interfaces=links, captured=stats['captured'], kernel_drops=stats['kernel_drops'], statistics_ok=stats['statistics_ok'], process_stopped=stats['process_stopped'], decoded_records=decoded['records'], file_complete=decoded['file_complete'], header_errors=decoded['header_errors'], snap_truncated_records=decoded['snap_truncated_records'])
    for sender, receiver in [('member', 'node'), ('node', 'member')]:
        tx = [p for p in result['points'][sender]['packets'] if p['direction']=='egress' and p['src'] in addresses[sender] and p['dst'] in addresses[receiver]]
        rx = [p for p in result['points'][receiver]['packets'] if p['direction']=='ingress' and p['src'] in addresses[sender] and p['dst'] in addresses[receiver]]
        result['directions'][sender+'_to_'+receiver] = dict(sender_packets=tx, receiver_packets=rx, exact_ip_joins=join_direction(tx, rx))
    result['endpoint_labels'] = {role: sorted(values) for role, values in addresses.items()}
    return result


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--input',type=Path,required=True)
    p.add_argument('--private',type=Path,required=True)
    args=p.parse_args()
    print(json.dumps(analyze(json.loads(args.input.read_text()),args.private)))


if __name__ == '__main__':
    main()
