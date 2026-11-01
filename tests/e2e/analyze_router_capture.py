#!/usr/bin/env python3
"""TEST ONLY routed PMTU analysis from closed, payload-free capture headers."""
import argparse
from collections import Counter
import json
from pathlib import Path
from analyze_native_capture import offsets, bound, first_gap


def analyze(root, family):
    rec=json.loads((root/('router-'+family+'.json')).read_text())
    obs={o['name']:json.loads(o['detail']) for o in rec['observations'] if o['name'] in ('capture','decode','transfer')}
    capture=obs['capture']; checks={Path(c['file']).stem:c for c in obs['decode']}
    rows={role:[json.loads(x) for x in (root/'headers'/rec['id']/(role+'.headers.jsonl')).read_text().splitlines()] for role in capture['points']}
    before=next(s['points'] for s in capture['samples'] if s['label']=='before-bulk')
    after=next(s['points'] for s in capture['samples'] if s['label']=='after-bulk')
    result=dict(transfer=obs['transfer'],capture_checks={},links={r:before[r]['links'] for r in before},
                sender_route_before=before['node']['routes']['peer_route'],sender_route_after=after['node']['routes']['peer_route'],
                ptb={},fragmentation_needed={},clock_offsets={})
    for role,points in rows.items():
        c=checks[role];p=capture['points'][role]
        complete=p['process_stopped'] and p['statistics_ok'] and p['kernel_drops']==0 and p['captured']==c['records']==len(points) and c['file_complete'] and c['header_errors']==c['snap_truncated_records']==0
        assert complete,role+' capture incomplete'
        result['capture_checks'][role]=dict(complete=complete,records=len(points),kernel_drops=p['kernel_drops'])
        off=offsets(capture,role);result['clock_offsets'][role]=off
        (root/'headers'/rec['id']/(role+'.bounds.jsonl')).write_text(''.join(json.dumps(bound(x,off),separators=(',',':'))+'\n' for x in points))
        result['ptb'][role]=[dict(packet=x['packet'],icmp=x['icmp'],ip_sha256=x['ip_sha256']) for x in points if x.get('protocol')==58 and x.get('icmp',{}).get('type')==2]
        result['fragmentation_needed'][role]=[x['packet'] for x in points if x.get('protocol')==1 and x.get('icmp',{}).get('type')==3 and x['icmp'].get('code')==4]
    target=before['node']['remote']['address']
    # Count each router interface separately. TTL/hop-limit changes prevent an
    # IP hash join across routing; unchanged egress packets can join the member.
    ingress_index=next(x['ifindex'] for x in before['router']['links'] if x['ifname']==('eth1' if family=='ipv6' else 'eth0'))
    egress_index=next(x['ifindex'] for x in before['router']['links'] if x['ifname']=='eth2')
    member_index=next(x['ifindex'] for x in before['member']['links'] if x['ifname']=='eth2')
    incoming=[x for x in rows['router'] if x.get('dst')==target and x['interface_index']==ingress_index and x['packet_type']!=4]
    outgoing=[x for x in rows['router'] if x.get('dst')==target and x['interface_index']==egress_index and x['packet_type']==4]
    received=[x for x in rows['member'] if x.get('dst')==target and x['interface_index']==member_index and x['packet_type']!=4]
    def fragments(points):
        return [x for x in points if x.get('mf') or x.get('fragment_offset') or any(e['type']==44 for e in x.get('extensions',[]))]
    oversize=[x for x in incoming if x['ip_length']>1280 and 'udp' in x]
    result['router_oversized_ingress']=[dict(packet=x['packet'],length=x['ip_length'],df=x.get('df'),identification=x.get('identification')) for x in oversize]
    result['router_ingress_fragments']=len(fragments(incoming))
    result['router_egress_fragments']=len(fragments(outgoing))
    available=Counter(x['ip_sha256'] for x in received)
    unmatched=[]
    for x in fragments(outgoing):
        if available[x['ip_sha256']]: available[x['ip_sha256']]-=1
        else: unmatched.append(x['packet'])
    result['egress_fragments_absent_at_member']=unmatched
    if family=='ipv4':
        result['router_fragment_groups']=[dict(ingress=x['packet'],identification=x['identification'],egress=[dict(packet=y['packet'],offset=y['fragment_offset'],mf=y['mf'],length=y['ip_length']) for y in outgoing if y.get('identification')==x['identification'] and y['src']==x['src']]) for x in oversize]
    port=obs['transfer']['bulk']['endpoint']['local_port']
    tx=[x for x in rows['origin'] if x.get('tcp',{}).get('src_port')==9446 and x['tcp']['dst_port']==port and x['packet_type']==4]
    rx=[x for x in rows['member'] if x.get('tcp',{}).get('src_port')==9446 and x['tcp']['dst_port']==port and x['packet_type']!=4]
    result['bulk_local_port']=port;result['first_missing_tcp_range']=first_gap(tx,rx)
    result['scope']='Virtual interface captures; GSO/GRO may coalesce inner TCP. No physical-wire frame count or authenticated ciphertext-to-inner mapping is inferred.'
    rec.update(id='router-analysis-'+family,title='D-137 routed PMTU header analysis',observations=[dict(name='analysis',detail=json.dumps(result))])
    (root/('router-analysis-'+family+'.json')).write_text(json.dumps(rec,indent=2)+'\n')
    print(family,obs['transfer']['bulk']['bytes'],obs['transfer']['bulk']['milliseconds'],'PTB',len(result['ptb']['router']),'oversize',len(oversize),'fragment egress',result['router_egress_fragments'],'unmatched',unmatched)

if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('run',type=Path);a=p.parse_args()
    for family in ('ipv6','ipv4'): analyze(a.run,family)
