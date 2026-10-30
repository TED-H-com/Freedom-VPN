#!/usr/bin/env python3
"""Derive acceptance subsets from measured rows; keep prior evidence attributed."""
import json
import math
import hashlib
import re
from run import PUBLIC, PRIVATE, write_json, utc
from followup5_records import body_verdict


def percentiles(values):
    values=sorted(values)
    if not values:return {'samples':0,'p50_ms':None,'p95_ms':None,'p99_ms':None}
    return dict(samples=len(values),**{'p'+str(p)+'_ms':values[math.ceil(len(values)*p/100)-1] for p in (50,95,99)})


def admission_pass(live):
    cohort=live.get('simultaneous_cohort',{})
    if cohort!={'users':20,'authorized':60,'binding_devices':60}:return False
    if live.get('phase')=='profile':
        rows=live.get('native_scale',{}).get('peer_observations',[])
        return len(rows)==60 and all(r.get('identity_matched') and r.get('authorized') and not r.get('join_error') for r in rows)
    c=live.get('native_helper',{}).get('counts',{})
    return c=={'Users':20,'Peers':60,'IdentityMatches':60,'Authorized':60,'ScannedLogs':60,'ProtectedLogs':60,'SimultaneousAuthorized':60}


def isolation_pass(row):
    checks=row.get('checks',[])
    requests=row.get('requests',[])
    return len(checks)==4 and [r['after_anonymous_requests'] for r in checks]==[0,40,80,120] and all(r['admin_status']==200 and r['lease_status']==200 for r in checks) and len(requests)==120 and any(r['status']==429 and r['code']=='RATE_LIMITED' and r['retry_after'] for r in requests)


def scans_pass(streams,transient,expected):
    # Protected permissions alone cannot turn a positive key count into zero.
    return bool(streams) and all(s.get('read_ok') and s.get('known_secret_matches')==0 and s.get('device_flow_key_lines')==0 for s in streams) and len(transient)==expected and all(s.get('present') and s.get('mode')=='0:600' and s.get('device_flow_keys')==0 for s in transient)


def main():
    run=json.loads((PUBLIC/'followup6-run.json').read_text())
    lives={phase:json.loads((PUBLIC/f'followup6-{phase}-live.json').read_text()) for phase in ('profile','helper')}
    common={'schema_version':1,'follow_up':6,'source_revision':run['source_revision'],'started_at':run['started_at'],'finished_at':run.get('finished_at'),'hardware':'shared development host, not candidate hardware','commands':[p.get('live',{}).get('command',[]) for p in run['phases']],'runs':{k:{field:v.get(field) for field in ('run_id','started_at','finished_at','source_revision','probe_source_sha256','policy_revision','node_incarnation','image_digests')} for k,v in lives.items()}}
    common['build_digests']=[{'arch':arch,'kind':'final compiled acceptance test binary; per-phase executed probe source hash remains separate','sha256':hashlib.sha256((PRIVATE/'build'/f'{arch}.test').read_bytes()).hexdigest()} for arch in ('amd64','arm64') if (PRIVATE/'build'/f'{arch}.test').exists()]
    phase_records={}
    all_scans={}
    for phase,live in lives.items():
        reads=[r for rows in live.get('control_reads',{}).get('rows',[]) for r in rows]
        peers=live.get('native_scale',{}).get('peer_observations',[])
        phase_records[phase]={'status':'passed' if admission_pass(live) else 'blocked','cohort':live.get('simultaneous_cohort'),
          'login_latency':percentiles([r['elapsed_ms'] for r in live.get('login_measurements',[])]),'admission_latency':percentiles([r['elapsed_ms'] for r in peers]),
          'admission_wall_seconds':live.get('native_scale',{}).get('elapsed_seconds') if phase=='profile' else live.get('native_helper',{}).get('elapsed_seconds'),
          'helper_counts':live.get('native_helper',{}).get('counts'),
          'control_read_latency':percentiles([r['elapsed_ms'] for r in reads]),
          'control_read_status':'passed' if len(reads)==600 and all(r.get('status')==200 and r.get('valid_page') and r.get('devices')==3 and r.get('foreign_devices')==0 for r in reads) and percentiles([r['elapsed_ms'] for r in reads])['p95_ms']<500 else 'blocked','reads_expected':600,'reads_observed':len(reads),'reads_200':sum(r['status']==200 for r in reads),'foreign_devices':sum(r.get('foreign_devices',0) for r in reads),
          'exact_three_device_pages':sum(r.get('valid_page') and r['devices']==3 for r in reads),
          'gateway_before':live.get('gateway_before'),'gateway_after':live.get('gateway_after'),'gateway_worker_count_qualification':'Profile used ps, absent in the image; its worker count and mixed stderr tokens are invalid. The labeled TCP state/FD values remain valid. Helper uses /proc cmdline; manual /proc check returned 2. The separate mid-run sampler used the original ps command, so all its worker zeros are unavailable.',
          'counted_peer_traffic':False,'node_process_cgroup_telemetry':None,
          'individual_helper_latencies':'not exposed by ScaleNativeMembers; only sequential profile has per-peer admission samples'}
        transient=[r['transient_log_scan'] for r in peers] if phase=='profile' else list(live.get('helper_transient_observations',{}).values())
        streams=live.get('secret_scan',[])
        logdir=PRIVATE/('followup6-'+phase+'-'+run['driver_run_id']+'-logs') if run.get('driver_run_id') else PRIVATE/('followup6-'+phase+'-logs')
        gateway_path=logdir/'gateway_stdout_stderr.log'
        gateway_raw=gateway_path.read_bytes() if gateway_path.exists() else b''
        gateway_diagnostic={'read_ok':gateway_path.exists(),'bytes':len(gateway_raw),'device_flow_key_lines':sum(bool(re.search(rb'user_code=|verification_uri|device_code=',line)) for line in gateway_raw.splitlines()),'connection_exhaustion_lines':sum(b'worker_connections are not enough' in line for line in gateway_raw.splitlines()),'query_values_published':False,'source':'protected complete gateway stdout/stderr capture before teardown'}
        all_scans[phase]={'status':'passed' if scans_pass(streams,transient,60) else 'open',
          'gateway_diagnostic':gateway_diagnostic,'retained_streams':streams,'known_canaries':live.get('known_secret_canaries'),'audit_rows':live.get('audit_rows_scanned'),
          'transient_present':sum(s['present'] for s in transient),'transient_protected':sum(s['present'] and s['mode']=='0:600' for s in transient),
          'transient_key_lines':sum(s['device_flow_keys'] for s in transient),'transient':transient,
          'transient_scope':'exact JoinNativeMember returned scan before deletion' if phase=='profile' else 'independent watcher via exported scanner; helper also reports aggregate pre-delete count but does not expose per-stream keys',
          'transient_stream_pattern':f'fvpn-w4-secops-member-{ "secops-uNN-pN" if phase=="profile" else "secops6-uNN-pN" }:/var/lib/vpn-node/e2e/netbird-up.log',
          'gateway':[s for s in streams if s['stream']=='gateway stdout/stderr']}
    capacity=dict(common,id='SECOPS-10',expected={'users':20,'peers':60,'read_p95_ms_less_than':500},status='passed' if all(r['status']=='passed' for r in phase_records.values()) else 'blocked',actual=phase_records,qualification='Admission subset only; no serving node or per-peer counted traffic in either phase. Prior two-device counted run remains history/follow-up-5/capacity.json.')
    write_json(PUBLIC/'capacity.json',capacity)
    write_json(PUBLIC/'native-scale.json',capacity)
    scans=dict(common,id='SECOPS-16',status='passed' if all(r['status']=='passed' for r in all_scans.values()) else 'open',expected='Every retained and deleted transient stream has zero secret canaries and recognized device-flow keys; protected mode alone does not prove this.',actual=all_scans,values_published=False)
    write_json(PUBLIC/'secret-scan.json',scans)
    helper=lives['helper']; rate=helper.get('rate_classes',{}); isolation=helper.get('anonymous_isolation',{})
    limiter=dict(common,id='AT28-f17',status='passed' if isolation_pass(isolation) and body_verdict(rate.get('usage_body_limits',[]),8<<20)=='passed' and rate.get('ledger',{}).get('actual_rows')==1 else 'blocked',expected='Anonymous saturation leaves admin and certified lease unaffected; usage cap remains 8 MiB and duplicate row count one',actual={'anonymous_isolation':isolation,'rate_classes':rate,'trusted_proxy':helper.get('trusted_proxy_row')})
    write_json(PUBLIC/'limiter.json',limiter)
    write_json(PUBLIC/'live.json',dict(common,evidence={k:f'followup6-{k}-live.json' for k in lives},scope='Follow-up 6 index; prior complete live run is history/follow-up-5/live.json'))
    for at in ('AT28','AT29','AT30','AT31','AT32'):
        old=json.loads((PUBLIC/'history/follow-up-5'/f'{at}.json').read_text())
        actual={'AT28':['limiter.json','secret-scan.json'],'AT29':['history/follow-up-5/deadlines.json'],'AT30':['history/follow-up-5/pitr.json'],'AT31':['capacity.json'],'AT32':['scope-review.json']}[at]
        gap={'AT28':'SECOPS-16 positive protected transient streams or incomplete stream evidence, complete resource matrix, live trusted-proxy row pending owner follow-up 18.',
             'AT29':'Finite grace prepare option; short node-lease setter blocked on contract revision 9 D-065; established TCP/UDP and guest clock rollback remain.',
             'AT30':'Unchanged: backups sources did not change since #165; real node archive and independent provider/peer reconciliation remain.',
             'AT31':'60-peer admission does not prove counted traffic per peer or mixed relay/UDP load. Acceptance owner still needs node process/cgroup telemetry; physical hardware/platform rows need operator run.',
             'AT32':'Three physical enrollment origins and member platforms remain needs operator run; no new implementation scope claim.'}[at]
        doc=dict(common,id=at,requirements=old.get('requirements'),expected=old.get('expected'),status='blocked',actual={'follow_up_6':actual,'historical_subset':f'history/follow-up-5/{at}.json'},blockers=[gap])
        write_json(PUBLIC/f'{at}.json',doc)
        (PUBLIC/f'{at}.md').write_text(f'# {at} — Follow-up 6\n\nFull status: **blocked**.\n\n{gap}\n\nCurrent evidence: '+', '.join(f'[{p}]({p})' for p in actual)+f'. Prior results remain [attributed history](history/follow-up-5/{at}.json).\n')
    def display(value):
        return 'not exposed' if value is None else f'{value:.3f}'
    rows=[]
    for phase,r in phase_records.items():
        c=r['cohort'] or {}; q=r['control_read_latency']; a=r['admission_latency']
        rows.append(f"| {phase} | {c.get('users')}/{c.get('authorized')} | {display(r['admission_wall_seconds'])} | {display(a['p50_ms'])} / {display(a['p95_ms'])} / {display(a['p99_ms'])} | {r['reads_200']}/{r['reads_expected']} | {q['p50_ms']} / {q['p95_ms']} / {q['p99_ms']} | {r['foreign_devices']} |")
    summary='# Follow-up 6 — measured admission and control diagnostics\n\nMeasured main `'+run['source_revision'][:7]+'`; devstack #170 and control #169. This is a shared development host, not candidate hardware.\n\n| Phase | Users/devices simultaneously authorized | Admission wall seconds | Per-peer admission p50/p95/p99 ms | Reads HTTP 200 | Read p50/p95/p99 ms | Foreign devices |\n|---|---|---|---|---|---|---|\n'+'\n'.join(rows)+'\n\nThe first profile is unchanged sequential 20×3. The second uses exported `ScaleNativeMembers` (three peers per user concurrently); the helper does not return individual admission latencies, hence nulls. Both use grants and product admission, no projected-row writes. Gateway socket/FD inventory is in [capacity.json](capacity.json) and [42 timed samples](gateway-inventory.json). The initial profile/observer worker count used unavailable `ps`, so those zeros are invalid; the helper uses `/proc` and observes two workers. Final established TCP / FD counts were 398 / 441 (profile) and 385 / 429 (helper). Socket TIME_WAIT entries are separate and are not live worker connections. Read windows are 20 readers × 30 seconds at one request per second.\n\nSECOPS-10 admission subset: **'+capacity['status']+'**. Full AT31 stays blocked: per-peer counted serving traffic, concurrent sampling/policy/relay/UDP, node process/cgroup telemetry, and candidate hardware are not measured here. The previous counted two-device evidence remains [history](history/follow-up-5/capacity.json).\n\nAT28 limiter subset: **'+limiter['status']+'**. [Limiter records](limiter.json) retain anonymous refusals with the intervening admin and certified lease responses, metrics and the 8 MiB boundary. This is one actual address, not an IPv6 table-fill simulation. Control follow-up 18 was not merged on measured main.\n\nSECOPS-16: **'+scans['status']+'**. [Scan evidence](secret-scan.json) includes protected pre-delete streams; a positive recognized device-flow key count keeps the all-stream finding open even when gateway diagnostics are value-free. No key values or matched lines are public.\n\nAT29: finite-grace preparation and D-065 setter remain requested. AT30: unchanged; backup sources did not change since #165. All full AT28–AT32 statuses remain blocked with their exact scope in the case records. See [requests](requests.md) and [run instructions](RUN.md).\n'
    (PUBLIC/'summary.md').write_text(summary)
    (PUBLIC/'capacity.md').write_text(summary)
    (PUBLIC/'limiter.md').write_text('# AT28 limiter rows — Follow-up 6\n\nMeasured subset: **'+limiter['status']+'**. See [record](limiter.json), including the complete request/response table, metrics and unchanged usage cap−1/cap/cap+1 boundaries. Anonymous saturation is one real address. No claim of 10,000 principal-table exhaustion or trusted-proxy behavior is made.\n')
    write_json(PUBLIC/'followup6-verdict.json',{'checked_at':utc(),'admission_subset':capacity['status'],'limiter_subset':limiter['status'],'all_stream_scan':scans['status'],'full_case_status':'blocked'})


if __name__=='__main__':main()
