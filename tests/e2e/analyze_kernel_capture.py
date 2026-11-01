#!/usr/bin/env python3
"""Analyze a new, unpinned native first-selection run using exact private trace receipts."""
import argparse
import hashlib
import json
from pathlib import Path
from analyze_native_capture import analyze
from decode_kernel_trace import decode, token
from decode_native_debug import decode as debug


def receipt(raw, expected):
    assert str(len(raw)) == expected['bytes']
    assert hashlib.sha256(raw).hexdigest() == expected['sha256']


def publish(path, source, identity, title, value):
    record = dict(source, id=identity, title=title, observations=[dict(name='analysis', detail=json.dumps(value, separators=(',', ':')))])
    # Derived evidence is immutable too; never overwrite an existing result.
    with path.open('x') as output:
        output.write(json.dumps(record, indent=2)+'\n')


def main(run, private):
    source = json.loads((run/'initial-selection.json').read_text())
    notes = {o['name']: o['detail'] for o in source['observations']}
    if 'transfer' not in notes:
        return  # Preserve instrumentation failure without inventing a transfer.
    assert not (run/'kernel-path-analysis.json').exists(), 'already analyzed; immutable'
    capture = json.loads(notes['capture'])
    trace = json.loads(notes['kernel_trace'])
    for role, point in capture['points'].items():
        receipt((private/'initial-selection'/(role+'.pcap')).read_bytes(), point['private_pcap_receipt'])
    analysis = analyze(run, run/'initial-selection.json')
    result = dict(syns=len(analysis['member_syns']), capture=analysis, trace={}, samples=[])
    for role in trace['roles']:
        raw = (private/'kernel'/(role+'-trace.txt')).read_bytes()
        receipt(raw, trace['roles'][role]['trace_receipt'])
        errors = (private/'kernel'/(role+'-trace-errors.txt')).read_bytes()
        receipt(errors, trace['roles'][role]['error_receipt'])
        assert not errors and trace['roles'][role]['removed'] and trace['roles'][role]['stopped']
        result['trace'][role] = decode(raw)
        assert result['trace'][role]['unparsed_lines'] == 0, 'incomplete trace decode'
    for sample in capture['samples']:
        row = dict(label=sample['label'], points={})
        for role in ['member', 'node']:
            point = sample['points'][role]
            row['points'][role] = {k: point.get(k) for k in ['start', 'end', 'wireguard_endpoint', 'wireguard_peer_matched', 'latest_handshake_unix_seconds', 'native_latest_handshake_at', 'local', 'remote', 'uses_kernel_interface']}
            row['points'][role]['read_ok'] = {k: point['commands'].get(k, {}).get('ok', False) for k in ['snmp4', 'snmp6', 'nstat', 'conntrack6', 'wg_latest_handshakes', 'wg_endpoints']}
            row['points'][role]['snmp6'] = {k: point['kernel']['snmp6'].get(k) for k in ['Ip6InReceives', 'Ip6InHdrErrors', 'Ip6InAddrErrors', 'Ip6InDiscards', 'Ip6InDelivers', 'Udp6InDatagrams', 'Udp6InErrors', 'Udp6InCsumErrors', 'Udp6NoPorts']}
            row['points'][role]['snmp4'] = point['kernel'].get('snmp4', {})
            row['points'][role]['nstat4'] = {k: v for k, v in point['kernel']['nstat'].items() if k.startswith(('IpIn', 'IpExtIn', 'UdpIn', 'UdpNoPorts'))}
            row['points'][role]['nstat_hdr_errors'] = point['kernel']['nstat'].get('Ip6InHdrErrors')
            row['points'][role]['loopback_conntrack'] = point['kernel']['loopback_conntrack']
        result['samples'].append(row)
    result['loopback_source_packets'] = {role: [p for p in analysis['wireguard_headers'][role] if p['src'] == token('::1')] for role in ['member', 'node']}
    result['trace_scope'] = 'nftrace IDs join hook traversal, not pcap hashes. Packet shape/count and bounded counter deltas corroborate the pcap; no timestamp or ciphertext identity is invented for nft trace.'
    publish(run/'kernel-path-analysis.json', source, 'kernel-path-analysis', 'F11-01 native capture and member IPv6 input evidence', result)
    debug_record = json.loads((run/'native-debug-export.json').read_text())
    debug_notes = {o['name']: o['detail'] for o in debug_record['observations']}
    result = {}
    for role in ['member', 'node']:
        raw = (private/(role+'-debug.log')).read_bytes()
        expected = json.loads(debug_notes[role])
        receipt(raw, expected)
        result[role] = dict(expected, events=debug(raw))
    publish(run/'native-debug-events.json', debug_record, 'native-debug-events', 'Pinned native client allowlisted events', result)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--run', type=Path, required=True)
    parser.add_argument('--private', type=Path, required=True)
    args = parser.parse_args()
    main(args.run, args.private)
