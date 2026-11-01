#!/usr/bin/env python3
"""TEST ONLY exact synthetic DNS reply joins; never infer client acceptance."""
import argparse
from collections import Counter
import json
from pathlib import Path
from analyze_native_capture import offsets, bound


def analyze(root):
    source=json.loads((root/'native-dns-delivery.json').read_text())
    notes={o['name']:json.loads(o['detail']) for o in source['observations'] if o['name'] in ('capture','decode','rr','idna-udp','idna-tcp')}
    ev=notes['capture']; checks={Path(c['file']).stem:c for c in notes['decode']};packets={};captures={}
    for role,p in ev['points'].items():
        rows=[json.loads(x) for x in (root/'headers'/'native-dns-delivery'/(role+'.headers.jsonl')).read_text().splitlines()]
        check=checks[role]
        complete=p['process_stopped'] and p['statistics_ok'] and p['kernel_drops']==0 and p['captured']==check['records']==len(rows) and check['header_errors']==check['snap_truncated_records']==0 and check['file_complete']
        assert complete,role+' capture incomplete'
        off=offsets(ev,role)
        captures[role]=dict(complete=complete,records=len(rows),clock_offsets=off)
        (root/'headers'/'native-dns-delivery'/(role+'.bounds.jsonl')).write_text(''.join(json.dumps(bound(x,off),separators=(',',':'))+'\n' for x in rows))
        links={link['ifindex']:link['ifname'] for link in ev['samples'][0]['points'][role]['links']}
        packets[role]=[dict(bound(x,off),interface=links.get(x['interface_index'],'unknown'),packet_type=x['packet_type'],src=x['src'],dst=x['dst'],transport=x.get('udp',x.get('tcp')),dns=x['dns']) for x in rows if 'dns' in x]
    result=dict(captures=captures,queries={},scope='Exact DNS-message hashes prove observation at named interfaces, not socket delivery, parsing or cache insertion. Missing TCP messages are not attributed without stream reassembly.')
    for label in ('rr','idna-udp','idna-tcp'):
        value=notes[label]; case='rr' if label=='rr' else 'idna_positive'
        trace=value['dns_trace'];wire_id=value['dns_wire_id']
        joins=[]
        for receipt in trace['resolver_events']:
            digest=receipt.get('reply_sha256')
            joins.append(dict(receipt=receipt,packet_matches={role:[x for x in rows if x['dns']['message_sha256']==digest] for role,rows in packets.items()}))
        result['queries'][label]=dict(result=value,resolver_reply_joins=joins,
            observed_client_id_packets={role:[x for x in rows if x['dns']['fixture_case']==case and x['dns']['wire_id']==wire_id] for role,rows in packets.items()})
    source.update(id='dns-delivery-analysis',title='F12-04 exact DNS reply delivery joins',observations=[dict(name='analysis',detail=json.dumps(result))])
    (root/'dns-delivery-analysis.json').write_text(json.dumps(source,indent=2)+'\n')
    for label,query in result['queries'].items():
        print(label,query['result']['rcode'],query['result']['milliseconds'],[(j['receipt']['role'],{role:len(rows) for role,rows in j['packet_matches'].items()}) for j in query['resolver_reply_joins']])

if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('run',type=Path);a=p.parse_args();analyze(a.run)
