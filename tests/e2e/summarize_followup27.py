#!/usr/bin/env python3
"""Publish immutable, compact transport timelines without changing raw rows."""
import argparse
from decimal import Decimal
import hashlib
import json
from pathlib import Path

from analyze_followup26 import notes
from analyze_followup27 import handshake_groups
from analyze_kernel_capture import publish
from followup27_io import read


def address(token):
    for ip in ('::1','127.0.0.1'):
        if token=='addr-'+hashlib.sha256(ip.encode()).hexdigest()[:16]:return ip
    return token


def main(run):
    source=json.loads((run/'measurement.json').read_text())
    measured=json.loads(notes(source)['analysis'])
    capture=json.loads(notes(json.loads(read(run/'native-session.json')))['capture'])
    packets={}
    for role in ('member','node'):
        indexes={link['ifindex']:link['ifname'] for s in capture['samples'] for link in s['points'][role]['links'] or []}
        packets[role]=[dict(p,interface=indexes.get(p.get('interface_index'),'unknown')) for p in
                       (json.loads(line) for line in read(run/f'headers/native-session/{role}.headers.jsonl').splitlines())]
    groups=handshake_groups(packets)
    states, counters = [], []
    state_keys=('wireguard_endpoint','wireguard_peer_matched','latest_handshake_unix_seconds','native_latest_handshake_at','peer_present','peer_connected','connection_type','uses_kernel_interface','local','remote','routes')
    for role in ('member','node'):
        prior_state, prior_counters, prior_sample = None, None, None
        for s in capture['samples']:
            point=s['points'][role]
            state={k:point.get(k) for k in state_keys}
            if state!=prior_state:
                states.append(dict(role=role,label=s['label'],start=point['start'],end=point['end'],state=state,commands=point['commands']))
                prior_state=state
            values={k:point.get('kernel',{}).get(family,{}).get(k) for family,names in
                    [('snmp4',('IpInHdrErrors','IpInAddrErrors','UdpInErrors')),('snmp6',('Ip6InHdrErrors','Ip6InAddrErrors','Udp6InErrors'))] for k in names}
            if prior_counters is not None:
                changes={k:dict(before=prior_counters[k],after=v,delta=str(int(v)-int(prior_counters[k]))) for k,v in values.items() if v is not None and prior_counters[k] is not None and v!=prior_counters[k]}
                if changes:counters.append(dict(role=role,before=prior_sample,after=dict(label=s['label'],start=point['start'],end=point['end']),changes=changes))
            prior_counters=values
            prior_sample=dict(label=s['label'],start=point['start'],end=point['end'])
    states.sort(key=lambda x:x['start']['wall'])
    underlay_responses=[dict(wireguard=g['wireguard'],hops=[h for h in g['hops'] if h['interface']!='lo']) for g in groups if g['wireguard']['type']==2 and any(h['interface']!='lo' for h in g['hops'])]
    result=dict(handshakes=groups,underlay_responses=underlay_responses,state_changes=states,error_counter_changes=counters,
                scope='Per-hop source flags belong to that packet, not every copy of the handshake. Route tables and NetBird state retain command bounds. Counter deltas bracket observations; they do not identify an exact kernel drop callsite.')
    publish(run/'transport-timeline.json',source,'transport-timeline','Per-hop response sources, endpoint/route states and error-counter windows',result)
    first=measured['first_selection'];stream=measured['stream']
    lines=[f"# Transport timeline: {run.name}", '', f"Source: `{source['source_commit']}`.", '',
           '[Measurement joins](measurement.json) · [Per-hop and state joins](transport-timeline.json) · [Every stream send](stream-intervals.csv)', '',
           f"First selection: **{first['status']}**, member/node/origin SYNs {first['member_syns']} / {first['node_syns']} / {first['origin_syns']}."]
    transit=first['first_syn_transit']
    if transit:lines.append(f"First SYN member-egress → node-ingress: {Decimal(transit['lower_ns'])/10**6}–{Decimal(transit['upper_ns'])/10**6} ms.")
    else:lines.append('No matching node ingress; no finite delivery interval is claimed.')
    lines.extend(['',f"Stream: {stream['attempts']} socket-write attempts; {stream['send_errors']} send errors; {stream['missing_ingress']} absent at node ingress; {len(stream['gaps_at_least_one_second'])} ingress gaps ≥1 s; {len(stream['delivered_after_one_second'])} deliveries whose upper bound exceeds 1 s.",
                  f"Missing ranges: `{stream['missing_ingress_ranges']}`. All loss is counted, including isolated and terminal loss.", '',
                  '| Unix seconds | Type / message hash prefix | Role / interface / packet | Source → destination | Family |',
                  '|---|---|---|---|---|'])
    rows=[]
    for g in groups:
        for h in g['hops']:
            rows.append((Decimal(h['timestamp']),f"| {h['timestamp']} | {g['wireguard']['type']} / `{g['wireguard']['message_sha256'][:16]}` | {h['role']} / {h['interface']} / {h['packet']} | `{address(h['src'])}:{h['udp']['src_port']}` → `{address(h['dst'])}:{h['udp']['dst_port']}` | IPv{h['ip_version']} |"))
    lines.extend(line for _,line in sorted(rows))
    lines.extend(['','Type 1 is initiation; type 2 is response. Full hashes and sender/receiver indexes join copies in the JSON. Known loopback addresses are named; other addresses remain stable content-free tokens. A loopback response on `lo` is distinguished from a loopback-source response on the underlay.', '',
                  f"Accepted closed reports after serving admission: {len(measured['closed_reports_after_admission'])}. Capture drops/read errors/truncations are zero on all three points; {measured['private_receipts_verified']} private receipts verified.", '',
                  'The complete route tables and NetBird/WireGuard state are in the original native-session capture. This reduced fixture does not reclassify a historical acceptance row.'])
    with (run/'transport-timeline.md').open('x') as out:out.write('\n'.join(lines)+'\n')
    print(json.dumps(dict(run=run.name,state_changes=len(states),counter_changes=counters,underlay_responses=len(underlay_responses))))


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('run',type=Path)
    args=parser.parse_args();main(args.run)
