#!/usr/bin/env python3
"""Render measured Follow-up 5 subsets without promoting them to full AT passes."""
import hashlib
import datetime
import re
import json
from run import ROOT, PUBLIC, REQUIREMENTS, EXPECTED, utc, write_json


def counted_proof(flow):
    delta=flow.get('admitted_bytes_delta',{})
    return (flow.get('route_device')=='fvnb0' and flow.get('http_200') is True
            and flow.get('origin_authenticated') is True and flow.get('origin_probe_matches') is True
            and flow.get('origin_node_snat') is True and flow.get('counter_reads_ok') is True
            and int(delta.get('upload',0))>0 and int(delta.get('download',0))>0)


def body_verdict(rows,cap):
    sizes={r.get('bytes'):r for r in rows}
    if set(sizes)!={cap-1,cap,cap+1}:
        return 'blocked'
    return 'passed' if all(r.get('passed') is True for r in rows) else 'failed'


def normalize_times(value):
    if isinstance(value,dict):return {k:normalize_times(v) for k,v in value.items()}
    if isinstance(value,list):return [normalize_times(v) for v in value]
    if isinstance(value,str) and re.fullmatch(r'\d{4}-\d\d-\d\dT[\d:.]+[+-]\d\d:\d\d',value):
        try:return datetime.datetime.fromisoformat(value).astimezone(datetime.timezone.utc).isoformat().replace('+00:00','Z')
        except ValueError:pass
    return value


def main():
    live=normalize_times(json.loads((PUBLIC/'live.json').read_text()))
    if live.get('follow_up')!=5:
        raise SystemExit('no Follow-up 5 live record; refusing stale evidence')
    if counted_proof(live.get('selected_exit_flow',{})):
        live.setdefault('pre_selection_probe',live.get('flow_probe'))
        live['flow_probe']=live['selected_exit_flow']
        live['counted_flow']=True
        live.pop('serving_blocker',None)
        live['serving_flow_positive']={'status':'passed','evidence':'selected_exit_flow','supersedes':'pre-selection gate; devstack #153 explicit exit selection'}
    write_json(PUBLIC/'live.json',live)
    builds=[]
    for path in sorted((ROOT/'.agent/e2e-build').glob('fvpn-*')):
        if path.is_file():builds.append({'name':path.name,'sha256':hashlib.sha256(path.read_bytes()).hexdigest(),'kind':'prepared application binary'})
    common={'schema_version':1,'follow_up':5,'run_id':live['run_id'],'source_revision':live['source_revision'],'probe_source_sha256':live.get('probe_source_sha256'),'started_at':live['started_at'],'finished_at':live['finished_at'],'policy_revision':live.get('policy_revision'),'node_incarnation':live.get('agent_node_incarnation'),'secondary_node_incarnation':live.get('node_incarnation'),'image_digests':live.get('image_digests',[]),'build_digests':builds,'commands':['FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 make -f mk/at-secops.mk at-secops-followup5'],'hardware':'shared development host; not candidate hardware'}
    flow=live.get('selected_exit_flow',{})
    flowrec=dict(common,id='counted-flow',status='passed' if counted_proof(flow) else 'blocked',expected='Selected API exit → fvnb0 → HTTPS 200 → authenticated node SNAT receipt → exact-device byte growth',actual=flow)
    write_json(PUBLIC/'counted-flow.json',flowrec)
    rates=live.get('rate_classes',{})
    rows=rates.get('usage_body_limits',[])
    other=live.get('other_body_boundaries',[])
    classes={name:body_verdict([r for r in other if r['class']==name],1<<20) for name in ['browser','agent','heartbeat','lease','enrollment','webhook']}
    usage=body_verdict(rows,8<<20)
    lim=dict(common,id='SECOPS-15',status='passed' if usage=='passed' and all(x=='passed' for x in classes.values()) else 'blocked',usage_status=usage,other_class_status=classes,expected='At cap−1 and cap: size accepted then documented validation/identity/domain response; cap+1: 413 PAYLOAD_TOO_LARGE',actual={'rate_classes':rates,'other_body_boundaries':other,'login':live.get('concurrent_sign_in')},history='followup4-limiter.json; previous usage failure superseded only by a passed current usage triplet')
    write_json(PUBLIC/'limiter.json',lim)
    matrix=live.get('deadline_matrix',{})
    # A closed path before the fault, loss while still valid, missing deadline
    # samples, or absent recovery cannot pass because a timeout loop completed.
    if (PUBLIC/'fault-injection.json').exists():
        matrix['worker_fault_guard']=json.loads((PUBLIC/'fault-injection.json').read_text())
    for kind in ('paid','grant'):
        row=matrix.get(kind,{})
        if row.get('status')=='passed repeated-HTTPS subset':
            after=[s for s in row.get('samples',[]) if s.get('request_start_after_service_end')]
            if not counted_proof(row.get('before_fault',{})) or not after:
                row['evaluated_status']='blocked'
            elif not row.get('control_lost_agent_alive',{}).get('http_200') or any(s.get('http_200') for s in after) or counted_proof(row.get('after_control_restored_expired',{})):
                row['evaluated_status']='failed'
            elif not counted_proof(row.get('recovery_flow',{})):
                row['evaluated_status']='blocked: recovery not proven'
            else:
                row['evaluated_status']='passed repeated-HTTPS subset'
            source=row.get('recovery_source_intervals',{})
            end=row.get('expected_service_end')
            row['original_source_end_unchanged']=(source.get('paid_through')==end) if kind=='paid' else any(g.get('valid_until')==end for g in source.get('grants',[]))
    deadline=dict(common,id='AT29-deadline-subsets',status='blocked',expected=EXPECTED['AT29'],actual=matrix,qualification='Finite paid/grant repeated-HTTPS fault subsets only; inspect each actual status. Grace, short node lease, established TCP/UDP and guest clock rollback remain distinct.')
    write_json(PUBLIC/'deadlines.json',deadline)
    capacity=dict(common,id='AT31-counted-subset',status='blocked',expected=EXPECTED['AT31'],actual=live.get('counted_capacity',{}),native_scale=live.get('native_scale',live.get('native_scale_gate')),qualification='Per-device counted bytes on the admitted cohort only. Full 20×3/second-node/forced-relay/UDP/candidate-hardware profile is not inferred.')
    node_resources=[x for x in capacity['actual'].get('resources',[]) if x.get('Name')=='fvpn-w4-secops-node']
    unavailable=sum(x.get('MemUsage')=='0B / 0B' for x in node_resources)
    capacity['actual']['resource_quality']={'node_docker_samples':len(node_resources),'unavailable_node_samples':unavailable,'status':'unavailable' if unavailable or not node_resources else 'observed','qualification':'0B / 0B is unavailable telemetry, never evidence of a zero-memory node; no candidate-hardware capacity inference.'}
    capacity['actual']['measurement_window_note']='Requests run for the fixed 300 seconds after 60-second warmup; the enclosing collector includes final resource collection and readback. Byte deltas include diagnostic policy requests and protocol overhead.'
    capacity['actual']['usage_accounting_scope']='Stored contiguous checkpoint and receipt/sample counts are observed completeness checks, not a claim about unobserved events. First member also has the separately identified node-B synthetic usage event.'
    write_json(PUBLIC/'capacity.json',capacity)
    revoke=dict(common,id='F3-02',expected='Next post-revoke observation retains coherent applied generation and block reasons; independently eligible second member still forwards',actual=live.get('post_revoke',{}))
    actual=revoke['actual'];revoke['status']='passed' if actual.get('request_status')==202 and actual.get('next_observations') and counted_proof(actual.get('second_before',{})) and counted_proof(actual.get('second_after',{})) and not counted_proof(actual.get('revoked_after',{})) else 'blocked'
    write_json(PUBLIC/'post-revoke.json',revoke)
    # Preserve the preceding publication regression run, without relabeling its
    # operation-step observations as measurements from this incarnation.
    publication=dict(common,id='SECOPS-11',status='historical halves retained; current populated serving probe',history='followup4-publication.json',actual={k:live.get(k) for k in ('publication_request','publication_operation','publication_observation_after','node_after_publication','selected_exit_flow')})
    write_json(PUBLIC/'publication.json',publication)
    refs={'AT28':['limiter.json','live.json','secret-scan.json'],'AT29':['counted-flow.json','deadlines.json','post-revoke.json'],'AT30':['pitr.json','pitr-source.json','pitr-compatibility.json'],'AT31':['capacity.json','counted-flow.json','limiter.json'],'AT32':['scope-review.json','requests.md']}
    gaps={'AT28':['Complete resource-family/refusal and every transient log coverage are not promoted from these scoped retests.','SECOPS-16 exhaustion retest requires devstack follow-up 10; previous finding remains open until that run.'],
          'AT29':['Finite paid/grant fault evidence is scoped to the measured transport; see deadlines.json.','Finite grace configuration hook, short node-lease input, guest rollback, established TCP/UDP remain unmeasured.'],
          'AT30':['No new physical restore in this follow-up. Original PITR subset remains historical; full native archive/provider/peer reconciliation is still blocked.'],
          'AT31':['Only the actually admitted counted cohort is measured; no 60-device, two-node, forced-relay or candidate-hardware certification.'],
          'AT32':['Physical three-source/platform/hardware comparisons remain needs operator run; current documentation mismatches are in scope-review.json.follow_up_5.']}
    if usage!='passed':gaps['AT28'].append('SECOPS-15 usage cap triplet did not pass.')
    if not counted_proof(flow):
        gaps['AT29'].insert(0,'Ordered counted-flow prerequisite not satisfied; no fault success claimed.')
        gaps['AT31'].insert(0,'Ordered counted-flow prerequisite not satisfied; no capacity success claimed.')
    cap=capacity['actual']
    actuals={'AT28':{'usage_boundary':usage,'other_class_boundaries':classes,'login':live.get('concurrent_sign_in',{}),'go_probe_failed':live.get('go_test_failed')},
             'AT29':{'counted_flow':flowrec['status'],'paid':matrix.get('paid',{}).get('evaluated_status',matrix.get('paid',{}).get('status')),'grant':matrix.get('grant',{}).get('evaluated_status',matrix.get('grant',{}).get('status')),'post_revoke':revoke['status'],'full_matrix':False},
             'AT31':{k:cap.get(k) for k in ('native_peers','reads','read_status_counts','wrong_member_responses','read_p50_ms','read_p95_ms','read_p99_ms','admitted_deltas','policy_changes','usage_visibility')},
             'AT32':{'review':'scope-review.json.follow_up_5','physical_platform_runs':'needs operator run','new_documentation_findings':['DOC15']}}
    for at in refs:
        rec=dict(common,id=at,requirements=REQUIREMENTS[at],expected=EXPECTED[at],status='blocked',actual=actuals.get(at,{}),evidence=refs[at],blockers=gaps[at])
        if at=='AT30':
            old=json.loads((PUBLIC/'AT30.json').read_text())
            old['follow_up_5']={'reviewed_at':utc(),'source_revision':common['source_revision'],'new_restore':False,'note':gaps[at][0]}
            rec=old
        write_json(PUBLIC/(at+'.json'),rec)
        measured={'AT28':'Measured usage and all other class body triplets: '+lim['status']+'.','AT29':'Counted selected exit: '+flowrec['status']+'; paid/grant repeated-HTTPS cutoff and recovery: '+str(actuals['AT29']['paid'])+' / '+str(actuals['AT29']['grant'])+'.','AT30':'No new physical restore; original PITR subset remains historical.','AT31':'Measured '+str(cap.get('native_peers'))+' native devices, '+str(cap.get('reads'))+' portal reads; p95 '+str(cap.get('read_p95_ms'))+' ms.','AT32':'Prior DOC01–14 fixes retained; DOC15 requests historical attribution of obsolete blockers.'}[at]
        (PUBLIC/(at+'.md')).write_text('# '+at+': blocked\n\n'+EXPECTED[at]+'\n\n'+measured+'\n\n'+'\n'.join('- '+g for g in gaps[at])+'\n\nEvidence: '+', '.join('['+x+']('+x+')' for x in refs[at])+'.\n')
    for name,rec in [('counted-flow',flowrec),('deadlines',deadline),('capacity',capacity),('post-revoke',revoke)]:
        (PUBLIC/(name+'.md')).write_text('# '+rec['id']+': '+rec['status']+'\n\n'+rec['expected']+'\n\n'+rec.get('qualification','See the ordered expected/actual gates and counters.')+'\n\n[JSON evidence]('+name+'.json), source `'+common['source_revision']+'`, run `'+str(common['run_id'])+'`.\n')
    lines=['# SECOPS-15 body boundary retest: '+lim['status'],'','Usage at 8 MiB; other route classes at 1 MiB. D-060 burst checks retain thresholds and metrics.','', '| Class | Bytes | Expected | Actual | Code |','|---|---:|---:|---:|---|']
    for name,items in [('usage',rows)]+[(name,[r for r in other if r['class']==name]) for name in classes]:
        for r in items:lines.append(f"| {name} | {r['bytes']} | {r['expected_status']} | {r['actual_status']} | {r['code']} |")
    lines+=['','Other-class cap tests include an actual node-A observation under node-B identity (403), invalid enrollment credentials, unsigned webhook, and whitespace on a bodyless bootstrap route (400). These are size-path tests, not successful forged observations. The usage body is schema-valid with legal trailing JSON whitespace. Duplicate ledger evidence is retained.','','[Current JSON](limiter.json); [superseded Follow-up 4 measurement](followup4-limiter.json).','']
    (PUBLIC/'limiter.md').write_text('\n'.join(lines))
    streams=live.get('secret_scan',[])
    scan=dict(common,id='secret-scan',status='blocked',scope='Inventoried retained streams only; removed transient native login logs excluded',streams=streams,known_secret_canaries=live.get('known_secret_canaries'),audit_rows=live.get('audit_rows_scanned'),known_canary_status='passed' if streams and all(s.get('read_ok') and s.get('known_secret_matches')==0 for s in streams) else 'blocked',exhaustion_history='followup4-secret-scan.json; no negative finding retirement without repeated exhaustion diagnostic')
    write_json(PUBLIC/'secret-scan.json',scan)


if __name__=='__main__':main()
