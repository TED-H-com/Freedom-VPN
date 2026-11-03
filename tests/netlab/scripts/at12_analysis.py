#!/usr/bin/env python3
"""TEST ONLY attribution ledger; it never relaxes the AT12 assertion."""
import gzip
import json
from pathlib import Path
import sys

DEVICE='019a0000-0000-7000-8000-0000000000a1'
COUNTER='u_'+DEVICE.replace('-','')


def read(path):
    if path.exists(): return path.read_text()
    return gzip.decompress(Path(str(path)+'.gz').read_bytes()).decode()


def analyze(root):
    before=json.loads(read(root/'before-observation.json'))
    mid=json.loads(read(root/'mid-observation.json'))
    result=dict(test_only=True, historical_f28_attribution='No original all-traffic capture exists; new packets cannot identify historical packet IDs.', phases={})
    for name,doc in [('before',before),('mid',mid)]:
        counters=doc['counter_document']['counters']; samples=doc['counter_document']['samples']
        selected=[s for s in samples if s['device_id']==DEVICE and s['direction']=='upload']
        if len(selected)!=1 or any(selected[0][k]!=counters[COUNTER][k] for k in ('packets','bytes')):
            raise ValueError('sample/counter mismatch or ambiguous sample')
        result['phases'][name]=dict(counter=counters[COUNTER],read_start_ns=str(doc['counter_start_ns']),read_end_ns=str(doc['counter_end_ns']))
    result['counter_deltas']={k:{f:int(v[f])-int(before['counter_document']['counters'][k][f]) for f in ('bytes','packets')}
        for k,v in mid['counter_document']['counters'].items() if v!=before['counter_document']['counters'].get(k)}
    lower,upper=before['counter_start_ns'],mid['counter_end_ns']
    inside_lower,inside_upper=before['counter_end_ns'],mid['counter_start_ns']
    roles={}; counts={}
    for role in ('origin','node','responder'):
        meta=json.loads(read(root/('at12-all-'+role+'.meta.json')))
        if not meta['complete']: raise ValueError('incomplete capture: '+role)
        rows=[json.loads(s) for s in read(root/('at12-all-'+role+'.jsonl')).splitlines()]
        if len(rows)!=meta['read_packets']: raise ValueError('capture count mismatch')
        roles[role]=rows; counts[role]=len(rows)
    result['capture_counts']=counts
    uploads=[r for r in roles['node'] if r['interface']=='fv-fh' and r['packet_type']==0 and r.get('source')=='100.64.1.1' and lower<=int(r['wall_ns'])<=upper]
    def identity(row):
        return tuple(row.get(k) for k in ('protocol','destination','source_port','destination_port','seq','ack','flags','ip_id','ip_length'))
    egress={}
    for r in roles['node']:
        if r['interface']=='fv-ih' and r['packet_type']==4:
            egress.setdefault(identity(r),[]).append(r)
    ledger=[]
    for r in uploads:
        row=dict(ingress=r,inside_both_reads=inside_lower<int(r['wall_ns'])<inside_upper)
        matches=[v for v in egress.get(identity(r),[]) if 0<=int(v['wall_ns'])-int(r['wall_ns'])<1_000_000_000]
        row['node_egresses']=matches
        row['origin_copies']=[v for v in roles['origin'] if identity(v)==identity(r) and v.get('source')==r['source'] and 0<=int(r['wall_ns'])-int(v['wall_ns'])<100_000_000]
        row['responder_copies']=[v for v in roles['responder'] if identity(v)==identity(r) and 0<=int(v['wall_ns'])-int(r['wall_ns'])<100_000_000]
        # Evidence of a prior tuple, not an automatic exemption for its bytes.
        row['prior_origin_packets']=[v['packet'] for v in roles['origin'] if int(v['wall_ns'])<lower and v.get('source')==r['source'] and all(v.get(k)==r.get(k) for k in ('protocol','destination','source_port','destination_port'))]
        ledger.append(row)
    result['upload_ledger']=ledger
    result['observed_egress_packets']=sum(bool(r['node_egresses']) for r in ledger)
    result['observed_egress_bytes']=sum(r['ingress']['ip_length'] for r in ledger if r['node_egresses'])
    result['read_boundary_ingresses']=[r['ingress']['packet'] for r in ledger if not r['inside_both_reads']]
    return result


if __name__=='__main__':
    result=analyze(Path(sys.argv[1])); dest=Path(sys.argv[2]); assert not dest.exists()
    dest.write_text(json.dumps(result,indent=2)+'\n')
    print(json.dumps({k:v for k,v in result.items() if k!='upload_ledger'},indent=2))
