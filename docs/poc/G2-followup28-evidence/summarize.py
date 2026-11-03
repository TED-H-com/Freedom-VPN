#!/usr/bin/env python3
"""TEST ONLY reproduce a readable ledger from the complete packet analysis."""
import collections
import gzip
import json
from pathlib import Path
import re

D=Path(__file__).resolve().parent

def read(path):
    return path.read_bytes() if path.exists() else gzip.decompress(Path(str(path)+'.gz').read_bytes())
a=json.loads(read(D/'tcp-analysis.json'))
events=[json.loads(x) for x in read(D/'window/reload/TestAT10_AT13_ReloadRevokeBaseline/stream-a1.log.jsonl').splitlines()]
sends={e['sequence']:e for e in events if e['event']=='tcp_send'}
reads={e['sequence']:e for e in events if e['event']=='tcp_read'}
starts={};offset=1
for seq in sorted(sends):
    starts[offset]=seq;offset+=len(f't{seq}\n'.encode())
expected={
 'request':['origin_out_fv-wg0','node_in_fv-fh','node_out_fv-ih','responder_in_fv-ii'],
 'reply':['responder_out_fv-ii','node_in_fv-ih','node_out_fv-fh','origin_in_fv-wg0']}
missing=[]; ledger=[]
for s in a['segments']:
    absent=[p for p in expected[s['direction']] if p not in s['stages']]
    if absent: missing.append(dict(segment=s['number'],direction=s['direction'],kind=s['kind'],absent=absent))
    if not s['retransmission']:continue
    original=a['segments'][s['retransmits_segments'][0]-1]
    cycle=starts.get(original['relative_seq'])
    ingress=s['stages'].get('node_in_fv-fh' if s['direction']=='request' else 'node_in_fv-ih')
    ledger.append(dict(segment=s['number'],cycle=cycle,original=original['number'],direction=s['direction'],kind=s['kind'],
        seq=s['seq'],ack=s['ack'],length=s['data_length'],
        after_original_ms=s['since_original_ns']/1e6,after_previous_ms=s['since_previous_overlapping_ns']/1e6,
        node_ingress=ingress,missing_stages=absent,
        read_elapsed_ms=(int(reads[cycle]['mono_ns'])-int(sends[cycle]['mono_ns']))/1e6 if cycle else None))
groups=[]
for original in sorted({r['original'] for r in ledger}):
    original_row=a['segments'][original-1];cycle=starts[original_row['relative_seq']]
    retries=[r for r in ledger if r['original']==original]
    successful=[r for r in retries if not r['missing_stages']]
    assert len(successful)==1
    groups.append(dict(cycle=cycle,original=original,retransmission_segments=[r['segment'] for r in retries],
        read_elapsed_ms=(int(reads[cycle]['mono_ns'])-int(sends[cycle]['mono_ns']))/1e6,
        recovered_ms=successful[0]['after_original_ms']))
rtos=[];snapshot_cost=[]
for s in a['snapshots']:
    text=s['reads']['socket']['output']
    match=re.search(r'\brto:(\d+(?:\.\d+)?)',text)
    backoff=re.search(r'\bbackoff:(\d+)',text)
    if match: rtos.append(dict(generation=s['generation'],phase=s['phase'],rto_ms=float(match[1]),backoff=int(backoff[1]) if backoff else 0))
    snapshot_cost.append(sum(int(s['reads'][k]['end']['mono_ns'])-int(s['reads'][k]['start']['mono_ns']) for k in ('conntrack','socket')))
result=dict(test_only=True,summary=a['summary'],missing_stage_segments=missing,retransmissions=ledger,affected_reads=groups,
    rto_samples=rtos,conntrack_states=sorted({r['state'] for s in a['snapshots'] for r in s['conntrack']}),
    conntrack_ids=sorted({r['id'] for s in a['snapshots'] for r in s['conntrack']}),
    empty_conntrack_snapshots=[dict(generation=s['generation'],phase=s['phase']) for s in a['snapshots'] if not s['conntrack']],
    all_read_max_ms=max((int(reads[n]['mono_ns'])-int(sends[n]['mono_ns']))/1e6 for n in reads),
    snapshot_read_cost_ms=dict(total=sum(snapshot_cost)/1e6,minimum=min(snapshot_cost)/1e6,maximum=max(snapshot_cost)/1e6),
    guard_upper_max_ms=max((w['release_end']-w['install_start'])/1e6 for w in a['windows']))
(D/'tcp-summary.json').write_text(json.dumps(result,indent=2)+'\n')
lines=['# Complete retransmission ledger','', 'TEST ONLY. All 11 retransmissions are member → responder data. Sequence/ACK values are absolute TCP header values. No TCP read failed. Every observation, including original emissions and return ACK/reply paths, remains in the complete analysis.','',
'| Retry segment | Application cycle | Original segment | Seq / ACK / bytes | Since original / previous (ms) | Node ingress bracket | Result | Read elapsed (ms) |',
'|---|---|---|---|---|---|---|---|']
for r in ledger:
    hits=', '.join(h['generation']+'/'+h['phase'] for h in r['node_ingress']['guard_positions']) or 'outside all brackets'
    outcome='no node egress / responder ingress' if r['missing_stages'] else 'all four request stages'
    lines.append(f"| {r['segment']} | {r['cycle']} | {r['original']} | {r['seq']} / {r['ack']} / {r['length']} | {r['after_original_ms']:.6f} / {r['after_previous_ms']:.6f} | {hits} | {outcome} | {r['read_elapsed_ms']:.6f} |")
lines += ['', 'There are seven affected reads. Eleven emissions are absent after node ingress: seven original data segments and four retransmissions. All eleven overlap an install/release command bracket. Successful retries traverse all four request stages, followed by replies reaching the origin. No SYN, SYN-ACK, reply, FIN or pure-ACK retransmission was observed. Command-boundary overlap plus a missing hop supports a guard-loss explanation, but does not expose the exact kernel commit or rule verdict.', '',
'Generation 1031 before/after: socket RTO 201 ms, pending bytes 0 → 4. After generation 1032: one retransmission, RTO still 201 ms. After generation 1033: two retransmissions, RTO 402 ms and backoff 1. Before generation 1035: bytes acknowledged and echo received, RTO back to 201 ms, three retransmissions total. Conntrack ID/state remain unchanged across these reads. The first retry is consistent with a [Linux v7.0 tail-loss probe](https://raw.githubusercontent.com/torvalds/linux/v7.0/net/ipv4/tcp_output.c); no kernel timer trace identifies its exact timer type. The later RTO/backoff change is directly present in ss output and is consistent with the [upstream retransmission timer](https://raw.githubusercontent.com/torvalds/linux/v7.0/net/ipv4/tcp_timer.c). These upstream sources describe a candidate mechanism; they are not an attestation of the distribution kernel build.', '',
'All flags, per-hop timestamps and install/release/verify classifications are in [the complete packet analysis](tcp-analysis.json.gz); [the derived summary](tcp-summary.json) includes every RTO sample and all missing-stage segments. See [the outcome and qualifications](overview.md).']
(D/'retransmissions.md').write_text('\n'.join(lines)+'\n')
print(json.dumps({k:result[k] for k in ('affected_reads','conntrack_states','empty_conntrack_snapshots','all_read_max_ms','snapshot_read_cost_ms','guard_upper_max_ms')},indent=2))
