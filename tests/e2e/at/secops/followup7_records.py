#!/usr/bin/env python3
"""Record measured subsets without upgrading missing transport/operator evidence."""
import hashlib
import json
import math
import subprocess
from run import ROOT, PUBLIC, PRIVATE, write_json, utc, REQUIREMENTS, EXPECTED
from followup5_records import body_verdict
from followup6_records import percentiles, isolation_pass


def protected_verdict(pre,post):
    return len(pre)==60 and len(post)==60 and all(s.get('present') and s.get('mode')=='0:600' for s in pre) and all(s.get('scan_ok') and s.get('deletion_proven') and not s.get('post_join_present') for s in post)


def cutoff_verdict(row):
    samples=row.get('samples',[])
    return row.get('before_fault',{}).get('counted') is True and row.get('signed_node_duration_seconds',float('inf'))<=60 and row.get('worker_pause_ok') is True and row.get('control_stop_ok') is True and row.get('agent_kill_command_ok') is True and any(s.get('request_start_after_deadline') for s in samples) and not any(s.get('request_start_after_deadline') and s.get('http_200') for s in samples) and row.get('recovery_flow',{}).get('counted') is True


def telemetry_summary(record):
    samples=record.get('samples',[])
    cpus=[s['cpu_percent_one_core'] for s in samples if 'cpu_percent_one_core' in s]
    return {'samples':len(samples),'cadence_seconds':record.get('cadence_seconds'),'errors':record.get('errors'),
            'cpu_percent_p95':sorted(cpus)[math.ceil(len(cpus)*.95)-1] if cpus else None,
            **{key+'_max':max((s[key] for s in samples),default=None) for key in ('memory_bytes','open_fds','conntrack_entries')}}


def main():
    run=json.loads((PUBLIC/'followup7-run.json').read_text())
    live=json.loads((PUBLIC/'followup7-live.json').read_text()) if (PUBLIC/'followup7-live.json').exists() else {}
    common={'schema_version':1,'follow_up':7,'started_at':run['started_at'],'finished_at':run.get('finished_at'),'source_revision':run['source_revision'],'application_base':'fb3ea32',
            'run_id':live.get('run_id'),'driver_run_id':run['driver_run_id'],'hardware':'shared development host; not candidate hardware',
            'commands':[run.get('up',{}).get('command'),run.get('live',{}).get('command'),run.get('down',{}).get('command')],
            'probe_source_sha256':live.get('probe_source_sha256'),'policy_revision':live.get('policy_revision'),
            'node_incarnation':live.get('node_incarnation'),'agent_node_incarnation':live.get('agent_node_incarnation'),
            'image_digests':live.get('image_digests',[]),'build_digests':[{'name':f.name,'sha256':hashlib.sha256(f.read_bytes()).hexdigest()} for f in (PRIVATE/'build').glob('*') if f.is_file()],
            'previous_run':'history/follow-up-6/','projector_owned_rows_written':live.get('projector_owned_rows_written',[])}
    def record(name,expected,actual,status,**extra):
        doc=dict(common,expected=expected,actual=actual,status=status,**extra)
        write_json(PUBLIC/(name+'.json'),doc)
        return doc
    native=live.get('native_scale',{})
    cohort=live.get('simultaneous_cohort',{})
    peerrows=native.get('peer_observations',[])
    admission=cohort=={'users':20,'authorized':60,'binding_devices':60} and len(peerrows)==60 and all(r.get('authorized') and r.get('identity_matched') and not r.get('join_error') for r in peerrows)
    measured=live.get('counted_capacity',{})
    counted=sum(all(int(v.get(k,0))>0 for k in ('upload','download')) for v in measured.get('admitted_deltas',{}).values())
    capacity=record('capacity',{'members':20,'devices':60,'read_p95_ms_less_than':500,'policy_seconds_at_most':60,'usage_seconds_at_most':120,'telemetry_cadence_seconds':5},
                    {'cohort':cohort,'admission_status':'passed' if admission else 'blocked','admission_wall_seconds':native.get('elapsed_seconds'),
                     'admission_latency':percentiles([r['elapsed_ms'] for r in peerrows]),'counted_peers':counted,'counted_capacity':measured,
                     'admission_node_telemetry':live.get('admission_node_telemetry'),'traffic_node_telemetry':live.get('traffic_node_telemetry'),
                     'telemetry_summary':{k:telemetry_summary(live.get(k,{})) for k in ('admission_node_telemetry','traffic_node_telemetry')},
                     'gateway_before':live.get('gateway_before'),'gateway_after':live.get('gateway_after'),'two_exit_traffic':live.get('two_exit_traffic')},'blocked',
                    qualification='Measured dev-host subsets; forced relay, established UDP and candidate-hardware certification are separate requirements.')
    record('native-scale','20 users ×3 simultaneously authorized peers with native identity read-back',native,'passed' if admission else 'blocked',simultaneous_cohort=cohort)
    record('counted-flow','Selected product exit, fvnb0, HTTPS200, authenticated node SNAT and exact-device byte growth',measured.get('initial_flows',[]),'passed' if len(measured.get('initial_flows',[]))==60 and all(r.get('counted') for r in measured['initial_flows']) else 'blocked')
    telemetry=record('node-telemetry','Container CPU/memory/FD/conntrack samples at a fixed five-second cadence',{k:live.get(k) for k in ('admission_node_telemetry','traffic_node_telemetry')},'passed' if all(live.get(k,{}).get('samples') and live[k].get('errors')==0 for k in ('admission_node_telemetry','traffic_node_telemetry')) else 'blocked')
    rate=live.get('rate_classes',{})
    proxy=live.get('trusted_proxy',{})
    limiter_ok=proxy.get('status')=='passed' and isolation_pass(live.get('anonymous_isolation',{})) and body_verdict(rate.get('usage_body_limits',[]),8<<20)=='passed' and rate.get('ledger',{}).get('actual_rows')==1
    limiter=record('limiter','Anonymous address A429 / address B enrollment replay200 through one trusted gateway; direct forged XFF cannot evade A429; admin and certified lease unaffected; 8 MiB unchanged',{'trusted_proxy':proxy,'anonymous_isolation':live.get('anonymous_isolation'),'rate_classes':rate},'passed' if limiter_ok else 'blocked')
    d=live.get('deadline_matrix',{})
    rows={kind:{'status':'passed' if cutoff_verdict(d.get(kind,{})) else 'blocked','row':d.get(kind)} for kind in ('paid','grant','lease')}
    restored=d.get('fleet_restore_86400',{})
    restore_ok=restored.get('put_status')==200 and restored.get('readback',{}).get('max_offline_lease_seconds')==86400
    deadlines=record('deadlines','D-065 short signed lease, earliest service/lease cutoff, finite grace and new-policy recovery; restore86400',d,'blocked',subset_verdicts=rows,default_restored=restore_ok,
                     qualification='Repeated HTTPS with worker frozen excludes central peer withdrawal; established TCP/UDP and guest clock rollback remain distinct.')
    pre=[r.get('transient_log_scan',{}) for r in peerrows]
    artifacts=live.get('protected_login_artifacts',{})
    post=artifacts.get('post_delete',[])
    streams=live.get('secret_scan',[])
    ordinary_ok=bool(streams) and all(s.get('read_ok') and s.get('known_secret_matches')==0 and s.get('device_flow_key_lines')==0 for s in streams)
    protected_ok=protected_verdict(pre,post)
    scans=record('secret-scan','D-066: zero canaries in ordinary logs/captures/readable files; protected transient credentials require owner/mode and deletion evidence',
                 {'ordinary_streams':streams,'ordinary_status':'passed' if ordinary_ok else 'blocked','protected_artifacts':{
                     'class':'protected transient credential artifacts, excluded from ordinary-log zero-canary criterion',
                     'status':'passed' if protected_ok else 'blocked','count':len(pre),'owner':'uid0','required_mode':'0600','recognized_key_line_count':sum(s.get('device_flow_keys',0) for s in pre),'pre_delete':pre,'post_delete':post},
                  'known_canaries':live.get('known_secret_canaries'),'audit_rows_scanned':live.get('audit_rows_scanned'),
                  'bulk_helper_per_stream':{'status':'blocked','owner':'w3-devstack-e2e follow-up 12','reason':'NativeScale returns aggregate scanned/protected counts; final per-stream scanner result not exported on measured base'}},'blocked',
                 criterion_change={'decision':'D-066','supersedes':'history/follow-up-6/secret-scan.json all-stream zero-key criterion','historical_results_preserved':True})
    gaps={
      'AT28':'Bulk ScaleNativeMembers per-stream attribution remains blocked on devstack follow-up 12; full resource-family ownership matrix remains beyond this limiter/log subset.',
      'AT29':'Finite-grace provider past_due truth, established member TCP/UDP transport export and disposable guest clock rollback remain unproven. See each cutoff verdict and the measured fleet restoration.',
      'AT30':'Unchanged since Follow-up 6: original isolated PITR remains valid historical evidence; real retained-node replay plus independently current provider/peer/revocation reconciliation remain.',
      'AT31':'See counted peers, latency and telemetry below; forced-relay mixed transport and candidate hardware remain unproven. Second-exit actual precondition is recorded, never inferred from publication.',
      'AT32':'Physical three-origin, physical arm64 and member-platform runs remain operator work; no production capacity or SLA claim follows from these lab numbers.'}
    paths={'AT28':['limiter.json','secret-scan.json'],'AT29':['deadlines.json'],'AT30':['history/follow-up-6/pitr.json'],'AT31':['capacity.json','node-telemetry.json'],'AT32':['scope-review.json']}
    for at in gaps:
        record(at,EXPECTED[at],{'evidence':paths[at],'historical_subset':f'history/follow-up-6/{at}.json'},'blocked',id=at,requirements=REQUIREMENTS[at],blockers=[gaps[at]])
        (PUBLIC/(at+'.md')).write_text(f'# {at} — Follow-up 7\n\nFull status: **blocked**.\n\n{gaps[at]}\n\nEvidence: '+', '.join(f'[{v}]({v})' for v in paths[at])+'.\n')
    record('live','Current full probe and immutable preceding evidence',{'probe':'followup7-live.json','prior':'history/follow-up-6/live.json'},'blocked')
    write_json(PUBLIC/'cleanup.json',dict(common,inventory=run.get('inventory_after'),status='passed' if run.get('inventory_after') and all(v['exit_code']==0 and v['count']==0 for v in run['inventory_after'].values()) else 'blocked'))
    perf=capacity['actual'];lat=perf['admission_latency'];t=perf['telemetry_summary']['traffic_node_telemetry']
    summary=f'''# Follow-up 7 — measured lab subsets

Application base `fb3ea32` (#195); harness `{run['source_revision'][:7]}`; run `{live.get('run_id')}`. Shared development host, **not candidate hardware**. Source, build/image digests, policy revision, incarnations, UTC times and commands are in each JSON record.

| Measurement | Actual |
|---|---|
| Simultaneous native cohort | {cohort.get('users')} users / {cohort.get('authorized')} devices / {cohort.get('binding_devices')} bindings |
| Sequential admission wall time | {native.get('elapsed_seconds')} s |
| Admission p50 / p95 / p99 | {lat['p50_ms']} / {lat['p95_ms']} / {lat['p99_ms']} ms |
| Counted peers during traffic | {counted} / 60 |
| Read p50 / p95 / p99 | {measured.get('read_p50_ms')} / {measured.get('read_p95_ms')} / {measured.get('read_p99_ms')} ms |
| Read statuses / wrong member responses | {measured.get('read_status_counts')} / {measured.get('wrong_member_responses')} |
| Node telemetry samples / errors / cadence | {t['samples']} / {t['errors']} / {t['cadence_seconds']} s |
| Node CPU p95 (100%=one core) | {t['cpu_percent_p95']}% |
| Peak memory / open FDs / conntrack | {t['memory_bytes_max']} bytes / {t['open_fds_max']} / {t['conntrack_entries_max']} |
| Proxy and body-limit subset | {limiter['status']} |
| Paid / grant / lease cutoff and recovery subsets | {rows['paid']['status']} / {rows['grant']['status']} / {rows['lease']['status']} |
| Fleet maximum restored to 86400 via API | {restore_ok} |
| Ordinary log scan / protected artifact lifecycle | {'passed' if ordinary_ok else 'blocked'} / {'passed' if protected_ok else 'blocked'} |

[Capacity](capacity.json) contains per-device bytes, fixed load profile, policies, usage watermarks, gateway inventory and the second-exit attempt. [Node telemetry](node-telemetry.json) contains the full fixed-cadence series. [Deadlines](deadlines.json) retains every fault sample and signed bound. [Limiter](limiter.json) retains exact response sequences and metrics. [D-066 scan](secret-scan.json) separates protected credentials from ordinary streams. The previous run is preserved byte-for-byte under [history/follow-up-6](history/follow-up-6/archive-index.json).

All full AT28–AT32 cases remain **blocked** for the requirements listed in their records; physical hardware/platform/clock rows are **needs operator run**. SECOPS-04's API/config inputs and SECOPS-10's admission facility are delivered; this does not turn unmeasured transports or a refused provider state into passes. AT30 is unchanged; no new physical restore run or RTO/RPO claim. See [requests](requests.md) and [run instructions](RUN.md).
'''
    (PUBLIC/'summary.md').write_text(summary)
    (PUBLIC/'capacity.md').write_text(summary)
    for name,doc in [('limiter',limiter),('deadlines',deadlines),('secret-scan',scans),('node-telemetry',telemetry)]:
        (PUBLIC/(name+'.md')).write_text(f'# {name} — Follow-up 7\n\nStatus: **{doc["status"]}** for the stated scope.\n\n{doc["expected"]}\n\n[Exact measured records]({name}.json), [summary](summary.md), [remaining requests](requests.md).\n')
    (PUBLIC/'counted-flow.md').write_text('# Counted flow — Follow-up 7\n\nThe ordered 60-peer proofs are in [counted-flow.json](counted-flow.json); timed byte deltas and any missing precondition are in [capacity.json](capacity.json). No API-only forwarding claim.\n')
    write_json(PUBLIC/'followup7-verdict.json',{'checked_at':utc(),'blocked':True,'go_live_status':run.get('live',{}).get('status'),'admission':'passed' if admission else 'blocked','counted_peers':counted,'limiter':limiter['status'],'ordinary_scan':'passed' if ordinary_ok else 'blocked','protected_artifacts':'passed' if protected_ok else 'blocked','cutoffs':{k:v['status'] for k,v in rows.items()},'fleet_default_restored':restore_ok})


if __name__=='__main__':main()
