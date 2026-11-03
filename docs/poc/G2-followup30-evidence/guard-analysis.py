#!/usr/bin/env python3
"""TEST ONLY exact positive guard log correlation and disclosed observation cost."""
import gzip
import hashlib
import json
from pathlib import Path
import re
import statistics
import sys

root=Path(sys.argv[1]);target=Path(sys.argv[2]);assert not target.exists()
sys.path.insert(0,str(Path(__file__).resolve().parents[3]/'tests/netlab/scripts'))
import tcp_analysis
def raw(name):
    p=root/name
    return p.read_text() if p.exists() else gzip.decompress(Path(str(p)+'.gz').read_bytes()).decode()
def lines(name):return [json.loads(s) for s in raw(name).splitlines() if s]
logs=lines('at10-guard.jsonl');applies=lines('at10-apply.jsonl');nodes=lines('at10-tcp-node.jsonl')
by_generation={a['generation']:a for a in applies}
KEY=('source','destination','source_port','destination_port','seq','ack','flags','ip_id','ip_length')
def same(a,b,keys=KEY):return all(a.get(k)==b.get(k) for k in keys)
rows=[]
for log in logs:
    if log.get('protocol')!=6 or 9001 not in (log.get('source_port'),log.get('destination_port')):continue
    apply=by_generation.get(log['generation'])
    if not apply:raise ValueError('log references an unknown generation')
    install=next(b for b in apply['batches'] if b['kind']=='guard_install')
    release=next(b for b in apply['batches'] if b['kind']=='guard_release')
    start=int(install['start']['wall_ns']);end=int(release['end']['wall_ns'])
    up=log['rule']=='iifname'
    match_keys=KEY if up else tuple(k for k in KEY if k!='destination')
    matches=[p for p in nodes if p['interface']==('fv-fh' if up else 'fv-ih') and p['direction']=='in' and same(p,log,match_keys) and (up or (p['destination']=='192.0.2.37' and log['destination']=='100.64.1.1')) and start<=int(p['wall_ns'])<=end]
    row=dict(log=log,ingress_matches=matches,install=install['start'],installed_by=install['end'],release_start=release['start'],release_end=release['end'])
    row['exact_rule_attribution']=len(matches)==1
    if len(matches)==1:
        p=matches[0];ns=int(p['wall_ns'])
        row['egress_matches']=[q for q in nodes if q['interface']==('fv-ih' if up else 'fv-fh') and q['direction']=='out' and same(q,p,tuple(k for k in KEY if k!=('source' if up else 'destination'))) and 0<=int(q['wall_ns'])-ns<100_000_000]
        row['timing_basis']='matched ingress bounds; pcap delivery time is not kernel verdict time'
        if 'kernel_wall_ns' in log:
            kernel=int(log['kernel_wall_ns'])
            row['kernel_timestamp_in_outer_bracket']=start-1000<=kernel<=end+1000
            row['kernel_bracket']='install' if kernel<=int(install['end']['wall_ns']) else 'installed_to_release' if kernel<int(release['start']['wall_ns']) else 'release'
            row['timing_basis']='NFLOG kernel timestamp, microsecond resolution, plus matched ingress'
        row['ingress_bracket']='install' if ns<=int(install['end']['wall_ns']) else 'installed_to_release' if ns<int(release['start']['wall_ns']) else 'release'
    rows.append(row)

def stats(values):
    values=sorted(values)
    return dict(count=len(values),min=min(values),median=statistics.median(values),p95=values[min(len(values)-1,int(.95*len(values)))],max=max(values),mean=statistics.mean(values))
times={k:[] for k in ('install_ns','release_ns','outer_guard_ns','installed_to_release_ns','producer_outer_guard_ns','added_script_bytes')}
snapshot_cost_ns=0
for a in applies:
    b={r['kind']:r for r in a['batches']}
    for batch in a['batches']:
        if hashlib.sha256(batch['test_only_script'].encode()).hexdigest()!=batch['test_only_sha256']:raise ValueError('executed script hash mismatch')
    for snap in a.get('test_only_tcp_snapshots',[]):
        for read in ('conntrack','socket'):
            snapshot_cost_ns+=int(snap[read]['end']['mono_ns'])-int(snap[read]['start']['mono_ns'])
    deadline=a['timing']['deadline']
    times['producer_outer_guard_ns'].append(int(deadline['release_end_ns'])-int(deadline['install_start_ns']))
    i,r=b['guard_install'],b['guard_release']
    times['install_ns'].append(int(i['end']['mono_ns'])-int(i['start']['mono_ns']))
    times['release_ns'].append(int(r['end']['mono_ns'])-int(r['start']['mono_ns']))
    times['outer_guard_ns'].append(int(r['end']['mono_ns'])-int(i['start']['mono_ns']))
    times['installed_to_release_ns'].append(int(r['start']['mono_ns'])-int(i['end']['mono_ns']))
    script=i['test_only_script']; original=re.sub(r' log prefix "f29g\d+-(?:iifname|oifname)" group 29 snaplen 128 queue-threshold 1','',script)
    if script.count('log prefix "f29g')!=2:raise ValueError('not exactly two instrumented guard rules')
    times['added_script_bytes'].append(len(script.encode())-len(original.encode()))
tcp=tcp_analysis.analyze(root)
missing=[]
for segment in tcp['segments']:
    incoming,outgoing=('node_in_fv-fh','node_out_fv-ih') if segment['direction']=='request' else ('node_in_fv-ih','node_out_fv-fh')
    if incoming not in segment['stages'] or outgoing in segment['stages']:continue
    packet=segment['stages'][incoming]['packet']
    matching=[r for r in rows if r['exact_rule_attribution'] and r['ingress_matches'][0]['packet']==packet]
    missing.append(dict(wall_ns=segment['wall_ns'],seq=segment['seq'],ack=segment['ack'],data_length=segment['data_length'],since_original_ns=segment.get('since_original_ns'),since_previous_overlapping_ns=segment.get('since_previous_overlapping_ns'),segment=segment['number'],direction=segment['direction'],kind=segment['kind'],
                        retransmission=segment['retransmission'],ingress=packet,
                        exact_guard_logs=[r['log']['packet'] for r in matching],attributed=len(matching)==1))
result=dict(test_only=True,historical_B1='FAIL',F27_01='unresolved: this window does not recover B1 packets',
    tcp_summary=tcp['summary'],missing_node_egresses=missing,all_missing_attributed=all(r['attributed'] for r in missing),
    timestamp_precision_ns=sorted(set(r['pcap_precision_ns'] for r in logs)),kernel_timestamp_present=sum('kernel_wall_ns' in r for r in logs),kernel_timestamp_absent=sum('kernel_wall_ns' not in r for r in logs),nflog_sequence_present=sum('nflog_sequence' in r for r in logs),log_ip_bytes=sum(r.get('ip_length',0) for r in logs),derived_log_bytes=len(raw('at10-guard.jsonl').encode()),log_count=len(logs),tcp_log_count=len(rows),exact_matches=sum(r['exact_rule_attribution'] for r in rows),
    logged_tcp_with_egress=sum(bool(r.get('egress_matches')) for r in rows),
    capture=json.loads(raw('at10-guard.meta.json')),tcp_snapshot_cost_ns=snapshot_cost_ns,cost={k:stats(v) for k,v in times.items()},
    cost_limit='Diagnostic logging only; incremental overhead is not isolated by a contemporaneous A/B. Production code unchanged.',logs=rows)
target.write_text(json.dumps(result,indent=2)+'\n');print(json.dumps({k:v for k,v in result.items() if k!='logs'},indent=2))
