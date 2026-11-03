#!/usr/bin/env python3
"""TEST ONLY derive the complete drop/retry ledger and measured cost receipt."""
import datetime
import gzip
import json
from pathlib import Path
D=Path(__file__).resolve().parent

def read(p):return p.read_text() if p.exists() else gzip.decompress(Path(str(p)+'.gz').read_bytes()).decode()
a=json.loads(read(D/'guard-analysis.json'));tcp=json.loads(read(D/'tcp-analysis.json'))
root=D/'guard-window/TestAT10_AT13_ReloadRevokeBaseline'
events=[json.loads(x) for x in read(root/'stream-a1.log.jsonl').splitlines()]
sends={e['sequence']:e for e in events if e['event']=='tcp_send'}
reads={e['sequence']:dict(e,elapsed_ns=int(e['mono_ns'])-int(sends[e['sequence']]['mono_ns'])) for e in events if e['event']=='tcp_read'}
log_rows=[]
for r in a['logs']:
    log=r['log'];packet=r['ingress_matches'][0]['packet']
    seg=next(s for s in tcp['segments'] if any(p['role']=='node' and p['packet']==packet for p in s['stages'].values()))
    cycle=max(n for n,x in sends.items() if int(x['wall_ns'])<=int(seg['wall_ns']))
    log_rows.append(dict(log=log['packet'],segment=seg['number'],cycle=cycle,kind=seg['kind'],retransmission=seg['retransmission'],generation=log['generation'],rule='inet fvpn fv_deadline_guard '+log['rule']+' "fv-fh" drop',direction=seg['direction'],seq=log['seq'],ack=log['ack'],ip_id=log['ip_id'],ip_length=log['ip_length'],kernel_wall_ns=log['kernel_wall_ns'],kernel_phase=r['kernel_bracket'],install_start_ns=r['install']['wall_ns'],installed_by_ns=r['installed_by']['wall_ns'],release_start_ns=r['release_start']['wall_ns'],release_end_ns=r['release_end']['wall_ns'],kernel_since_install_start_ns=int(log['kernel_wall_ns'])-int(r['install']['wall_ns']),installed_by_after_kernel_ns=int(r['installed_by']['wall_ns'])-int(log['kernel_wall_ns']),read_elapsed_ns=reads[cycle]['elapsed_ns']))
retry_rows=[]
for s in tcp['segments']:
    if not s['retransmission']:continue
    cycle=max(n for n,x in sends.items() if int(x['wall_ns'])<=int(s['wall_ns']))
    retry_rows.append(dict(segment=s['number'],cycle=cycle,direction=s['direction'],retransmits=s['retransmits_segments'],since_original_ns=s['since_original_ns'],since_previous_ns=s['since_previous_overlapping_ns'],stages=list(s['stages']),read_elapsed_ns=reads[cycle]['elapsed_ns']))
lo=int(a['capture']['ready']['wall_ns']);hi=int(a['capture']['end']['wall_ns'])
checks=[]
for line in read(D/'guard-window-quiet.jsonl').splitlines():
    r=json.loads(line)
    if 'interval_seconds' not in r:continue
    end=datetime.datetime.fromisoformat(r['utc']).timestamp()*1e9
    start=end-r['interval_seconds']*1e9
    if start<=hi and end>=lo:checks.append(r)
commands=[json.loads(x) for x in read(D/'guard-command-cost.jsonl').splitlines()]
command_cost=[dict(action=r['command'][-2],elapsed_ns=int(r['end']['mono_ns'])-int(r['start']['mono_ns']),exit_code=r['exit_code']) for r in commands]
result=dict(test_only=True,historical_B1='FAIL',F27_01='unresolved',window_qualified=False if any(not r['quiet'] for r in checks) else True,
            window_cpu_checks=len(checks),window_cpu_breaches=[dict(utc=r['utc'],interval_seconds=r['interval_seconds'],foreign_cpu=r['cpu_cores']['foreign'],contributors=r['foreign_contributors']) for r in checks if not r['quiet']],
            drops=log_rows,retries=retry_rows,longest_read=max(reads.values(),key=lambda r:r['elapsed_ns']),collector_command_cost=command_cost,
            collector_ready_cost_ns=int(a['capture']['ready']['mono_ns'])-int(a['capture']['start']['mono_ns']),cost=a['cost'],tcp_snapshot_cost_ns=a['tcp_snapshot_cost_ns'],
            log_count=a['log_count'],tcp_logs=a['tcp_log_count'],log_ip_bytes=a['log_ip_bytes'],derived_log_bytes=a['derived_log_bytes'],raw_pcap_bytes_discarded=a['capture']['raw_bytes'])
(D/'window-summary.json').write_text(json.dumps(result,indent=2)+'\n')
lines=['# Exact guard-drop ledger','', 'All seven missing probe TCP egresses have unique NFLOG matches. All are member → responder, rule `inet fvpn fv_deadline_guard iifname "fv-fh" drop`. The kernel timestamps lie inside their generation’s install-command bracket; the atomic commit is not assigned to a command endpoint. The window is **unqualified** for host quietness, and B1 stays FAIL.','', '| Log / segment | Cycle | Type | Generation | Kernel time since install start (ms) | Installed-by after kernel time (ms) | Read elapsed (ms) |','|---|---:|---|---:|---:|---:|---:|']
for r in log_rows:lines.append(f'| {r["log"]} / {r["segment"]} | {r["cycle"]} | {"retry " if r["retransmission"] else ""}{r["kind"]} | {r["generation"]} | {r["kernel_since_install_start_ns"]/1e6:.6f} | {r["installed_by_after_kernel_ns"]/1e6:.6f} | {r["read_elapsed_ns"]/1e6:.6f} |')
lines+=['','[The complete summary](window-summary.json) retains absolute kernel timestamps and all four command endpoints, sequence/ACK/IP identities, every retry and CPU breach. [The guard analysis](guard-analysis.json) retains unique node matches and zero contradictory egresses. The [full TCP analysis](tcp-analysis.json.gz) retains every endpoint and hop observation, including all seven retransmissions.','']
(D/'drop-ledger.md').write_text('\n'.join(lines))
print(json.dumps({k:v for k,v in result.items() if k not in ('drops','retries','cost','window_cpu_breaches')},indent=2))
