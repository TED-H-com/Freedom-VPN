#!/usr/bin/env python3
"""Separate underlay loopback-source data from handshake-response evidence."""
import argparse
from collections import Counter
import json
from pathlib import Path
from analyze_followup26 import notes
from analyze_followup27 import stamp
from analyze_kernel_capture import publish
from followup27_io import read


def main(run):
    source=json.loads((run/'measurement.json').read_text())
    capture=json.loads(notes(json.loads(read(run/'native-session.json')))['capture'])
    timeline=json.loads(notes(json.loads((run/'transport-timeline.json').read_text()))['analysis'])
    packets={}
    groups={}
    for role in ('member','node'):
        indexes={link['ifindex']:link['ifname'] for s in capture['samples'] for link in s['points'][role]['links'] or []}
        packets[role]=[dict(p,interface=indexes.get(p.get('interface_index'),'unknown')) for p in
                       (json.loads(line) for line in read(run/f'headers/native-session/{role}.headers.jsonl').splitlines())]
        for p in packets[role]:
            wg=p.get('wireguard',{})
            if p['interface']=='lo' or not wg.get('source_loopback'):continue
            g=groups.setdefault(wg['message_sha256'],dict(wireguard=wg,hops=[]))
            g['hops'].append(dict(role=role,**p))
    rows=sorted(groups.values(),key=lambda g:min(stamp(p) for p in g['hops']))
    for g in rows:
        receiver=g['wireguard'].get('receiver_index')
        g['prior_response_hashes']=[h['wireguard']['message_sha256'] for h in timeline['handshakes'] if h['wireguard']['type']==2 and h['wireguard']['sender_index']==receiver and min(stamp(p) for p in h['hops'])<=min(stamp(p) for p in g['hops'])]
    following=[]
    for receiver in sorted({g['wireguard']['receiver_index'] for g in rows if g['wireguard']['type']==4}):
        last=max(stamp(p) for g in rows if g['wireguard'].get('receiver_index')==receiver for p in g['hops'])
        for role in ('node','member'):
            p=next((p for p in packets[role] if stamp(p)>last and p['interface']!='lo' and p.get('wireguard',{}).get('type')==4 and p['wireguard'].get('receiver_index')==receiver and not p['wireguard']['source_loopback']),None)
            if p:following.append(dict(role=role,**p))
    delta=sum(int(c['changes'].get('Ip6InHdrErrors',{}).get('delta','0')) for c in timeline['error_counter_changes'] if c['role']=='member')
    value=dict(messages=rows,types=dict(Counter(str(g['wireguard']['type']) for g in rows)),joined_both_ends=sum({p['role'] for p in g['hops']}=={'member','node'} for g in rows),member_ipv6_header_error_increment=str(delta),counter_windows=timeline['error_counter_changes'],following_non_loopback_data=following,
               scope='Same opaque WireGuard bytes join node and member underlay captures. Header-error increments are bounded observations, not exact drop callsites. Type 4 data is distinct from a type 2 handshake response; this artifact does not infer a handshake retry or historical cause.')
    publish(run/'loopback-underlay.json',source,'loopback-underlay','All loopback-source underlay packets and independent error-counter windows',value)
    print(json.dumps(dict(run=run.name,messages=len(rows),types=value['types'],joined=value['joined_both_ends'],header_error_increment=str(delta))))


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('run',type=Path)
    args=parser.parse_args();main(args.run)
