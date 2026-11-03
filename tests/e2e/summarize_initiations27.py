#!/usr/bin/env python3
"""Correct origin attribution of loopback copies without rewriting evidence."""
import argparse
import json
from pathlib import Path
from analyze_followup26 import notes
from analyze_followup27 import initiation_origin, stamp
from analyze_kernel_capture import publish


def main(run):
    source=json.loads((run/'transport-timeline.json').read_text())
    groups=json.loads(notes(source)['analysis'])['handshakes']
    rows=[]
    previous={}
    for group in groups:
        if group['wireguard']['type']!=1:continue
        origin=initiation_origin(group)
        row=dict(wireguard=group['wireguard'],origin=origin,response_hashes=[g['wireguard']['message_sha256'] for g in groups if g['wireguard']['type']==2 and g['wireguard']['receiver_index']==group['wireguard']['sender_index']])
        if origin:
            role=origin['role']
            if role in previous:row['since_previous_same_origin_ns']=str(stamp(origin)-previous[role])
            previous[role]=stamp(origin)
        rows.append(row)
    publish(run/'initiation-origins.json',source,'initiation-origins','Direction-qualified handshake initiation origins',dict(initiations=rows,scope='Supersedes the member_initiations convenience list in earlier measurement.json files, which also selected incoming loopback copies. Original packet/handshake timelines and all measured intervals remain unchanged. The pinned harness local WG port is 51820; SLL2 outgoing underlay or the local WG loopback source identifies origin.'))
    print(json.dumps(dict(run=run.name,origins=[dict(role=r['origin']['role'] if r['origin'] else None,at=r['origin']['timestamp'] if r['origin'] else None,interval_ns=r.get('since_previous_same_origin_ns')) for r in rows])))


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('run',type=Path)
    args=parser.parse_args();main(args.run)
