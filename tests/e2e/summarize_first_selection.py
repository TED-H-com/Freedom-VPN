#!/usr/bin/env python3
"""TEST ONLY: derive a new immutable first-selection row from verified headers."""
import argparse
import json
from pathlib import Path
from analyze_kernel_capture import publish
from decode_kernel_trace import token


def elapsed(first, later):
    if first is None or later is None:
        return None
    return dict(lower_ns=str(int(later['mono_lower_ns'])-int(first['mono_upper_ns'])),
                upper_ns=str(int(later['mono_upper_ns'])-int(first['mono_lower_ns'])))


def data_around(packets, response):
    data = [p for p in packets if p['wireguard']['type'] == 4]
    if response is None:
        return dict(total_records=len(data), before=None, after=None, overlaps=None)
    before = sum(int(p['mono_upper_ns']) < int(response['mono_lower_ns']) for p in data)
    after = sum(int(p['mono_lower_ns']) > int(response['mono_upper_ns']) for p in data)
    return dict(total_records=len(data), before=before, after=after,
                overlaps=len(data)-before-after)


def summary(source, analysis):
    notes = {o['name']: o['detail'] for o in source['observations']}
    cap = analysis['capture']
    syns = cap['member_syns']
    start = syns[0] if syns else None
    origin = [p for p in cap['hops']['origin'] if p['tcp']['src_port'] == 20001 and p['packet_type'] != 4]
    node_syns = [p for p in cap['hops']['node'] if p['tcp']['src_port'] == 20001 and p['tcp']['flags'] == 2]
    origin_receipt = origin[0] if origin else None
    headers = cap['wireguard_headers']
    responses = [p for p in headers['node'] if p['wireguard']['type'] == 2 and p['packet_type'] == 4]
    loopback = [p for p in responses if p['src'] == token('::1')]
    first_response = responses[0] if responses else None
    first_loopback = loopback[0] if loopback else None
    result = dict(arm=notes.get('test_only_arm'), family=notes.get('test_only_family'),
                  status=source['status'], syns=len(syns), node_syn_records=len(node_syns),
                  first_member_syn=start, first_origin_packet=origin_receipt,
                  time_to_first_origin_packet=elapsed(start, origin_receipt),
                  origin_http_receipt=json.loads(notes['origin_arrival']),
                  http=cap['transfer']['http'], transfer_counters={k:cap['transfer'][k] for k in ['before','after']},
                  first_node_response=first_response, first_loopback_source_response=first_loopback,
                  loopback_source_response_records=len(loopback),
                  type4_around_first_response={r:data_around(headers[r],first_response) for r in ['member','node']},
                  type4_around_loopback_response={r:data_around(headers[r],first_loopback) for r in ['member','node']},
                  selected_peer_timeline={}, namespace_counters={}, loopback_response_member_joins=[])
    for response in loopback:
        digest=response['wireguard']['message_sha256']
        result['loopback_response_member_joins'].append(dict(node_packet=response['packet'],message_sha256=digest,
            member_packets=[p['packet'] for p in headers['member'] if p['wireguard']['message_sha256']==digest]))
    for role in ['member','node']:
        prior=None; timeline=[]
        for sample in analysis['samples']:
            p=sample['points'][role]
            state={k:p.get(k) for k in ['wireguard_endpoint','wireguard_peer_matched','latest_handshake_unix_seconds','uses_kernel_interface','local','remote']}
            if state!=prior:
                timeline.append(dict(label=sample['label'],start=p['start'],end=p['end'],**state));prior=state
        result['selected_peer_timeline'][role]=timeline
        result['namespace_counters'][role]={}
        for key in ['snmp4','snmp6','nstat4','nstat_hdr_errors']:
            result['namespace_counters'][role][key]={s['label']:s['points'][role].get(key) for s in [analysis['samples'][0],analysis['samples'][-1]]}
    result['scope']='Type-4 counts are structural capture records, including multiple proxy hops, not authenticated/unique data messages. Before/after use non-overlapping monotonic bounds. No response means null ordering, not zero data. Origin delay is first captured inner packet from first member SYN, separate from HTTP completion/receipt. Endpoints and handshakes are sampled, not continuous.'
    return result


def main(run):
    source=json.loads((run/'initial-selection.json').read_text())
    analysis=json.loads(json.loads((run/'kernel-path-analysis.json').read_text())['observations'][0]['detail'])
    publish(run/'first-selection-summary.json',source,'first-selection-summary','TEST ONLY bounded first-selection summary',summary(source,analysis))


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('run',type=Path)
    main(parser.parse_args().run)
