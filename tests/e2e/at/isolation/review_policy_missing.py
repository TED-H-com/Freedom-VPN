"""D-135/D-136 producer windows, retaining loss and recovery availability."""
import json
from evidence_io import read_json
from pathlib import Path
from producer_timing import identity, ns, packet_trace, read_operations, validate_operations


def windows(rows, streams):
    result=[]
    for run in dict.fromkeys(r['run_id'] for r in rows):
        process=[r for r in rows if r['run_id']==run]
        i=0
        while i<len(process):
            first=process[i]
            if first.get('baseline',{}).get('cause')!='policy_missing':
                i+=1;continue
            k=next((j for j in range(i+1,len(process)) if process[j].get('timing',{}).get('operation')=='apply' and process[j]['timing']['succeeded']),None)
            selected=process[i:k+1 if k is not None else len(process)]
            baselines=[r for r in selected if r['source']=='baseline']
            apply=process[k] if k is not None else None
            start=ns(first['baseline'],'start_ns'); end=ns(apply['timing'],'end_ns') if apply else None
            old=first.get('lease',{}); new=apply['timing']['candidate_generation'] if apply else None
            stale=bool(old.get('members',0)>0 and new and old['generation']!=new)
            # A later baseline can reopen the outage; do not skip it to find a
            # convenient serving heartbeat. These are later endpoints, not egress.
            following=[]
            recovery_limit = ns(process[-1]['anchor'], 'mono_after_ns') if 'anchor' in process[-1] else None
            if k is not None:
                for r in process[k+1:]:
                    if r['source']=='baseline':
                        recovery_limit=ns(r['baseline'], 'start_ns')
                        break
                    following.append(r)
            live=next((r for r in following if r['source']=='live' and r['timing']['succeeded'] and r['timing']['verified'] and r['timing']['candidate_generation']==new),None)
            heartbeat=next((r for r in following if r.get('heartbeat',{}).get('written') and r['heartbeat']['serving'] and r['heartbeat']['generation']==new),None)
            packets={}
            for name,trace in streams.items():
                inside=[p for p in trace if start<=ns(p,'ingress_ns') and (end is None or ns(p,'ingress_ns')<=end)]
                after=[p for p in trace if end is not None and ns(p,'ingress_ns')>end and ns(p,'egress_ns')>0 and (recovery_limit is None or ns(p,'egress_ns')<recovery_limit)]
                lost=[p for p in inside if p['result']!='ok' or not ns(p,'egress_ns')]
                guards=[r for r in process if r.get('timing',{}).get('operation')=='apply' and r['timing']['deadline']['install_ok'] and r['timing']['deadline']['guard_released']]
                losses=[dict(p,completed_guard_joins=[identity(g) for g in guards if ns(g['timing']['deadline'],'install_start_ns')<=ns(p,'ingress_ns')<=ns(g['timing']['deadline'],'release_end_ns')]) for p in lost]
                packets[name]=dict(coverage='sampled' if inside else 'unavailable: no ingress captured in this bracket',
                    ingresses=len(inside),losses=losses,ingress_losses_in_completed_guards=sum(bool(p['completed_guard_joins']) for p in losses),ingress_losses_outside_completed_guards=sum(not p['completed_guard_joins'] for p in losses),
                    first_post_apply_ingress_with_egress=after[0] if after else None)
            previous=next((r for r in reversed(process[:i]) if 'timing' in r and r['timing']['operation']=='apply'),None)
            result.append(dict(identity(first),baseline_rows=baselines,
                old_lease=old,next_successful_apply=dict(identity(apply),timing={a:b for a,b in apply['timing'].items() if a!='commands'}) if apply else None,
                accepted_renewals=[r for r in selected if r.get('renewal',{}).get('new_lease')],
                previous_apply=dict(identity(previous),succeeded=previous['timing']['succeeded']) if previous else None,
                failed_applies=[identity(r) for r in selected if r.get('timing',{}).get('operation')=='apply' and not r['timing']['succeeded']],
                baseline_start_ns=str(start),apply_end_ns=str(end) if end else None,
                baseline_to_apply_ns=str(end-start) if end else None,
                classification='policy_missing baseline during populated lease generation '+old['generation']+'→'+new+'; stale-lease path consistent with code' if stale else 'policy_missing; stale populated lease transition not established',
                stale_populated_lease_candidate=stale,cause_status='unproven',
                next_live=live,next_serving_heartbeat=heartbeat,packets=packets,
                recovery_observation_limit_ns=str(recovery_limit) if recovery_limit is not None else None,
                recovery='Apply plus later live verification and serving heartbeat observed' if live and heartbeat else 'later serving recovery unavailable within this baseline-free interval'))
            i=k+1 if k is not None else len(process)
    return result


def review(root):
    read=lambda name:read_json(root/name)
    rows=read_operations(root)
    validate_operations(rows)
    clock=read('producer-clock.json')
    renewal=packet_trace(read('renewal-probes.json'),read('renewal-packets.json'),clock)
    expiry=packet_trace(read('whole-node-probes.json')['probes'],read('whole-node-packets.json'),clock)
    result=dict(run_id=read('AT10.json')['run_id'],interpretation='D-136: ordinary stale-lease path installs baseline, ACKs, renews, then Applies/verifies/heartbeats. Only retained Apply-error recovery uses a verified deny-all candidate. policy_missing alone is not proof of generation mismatch. No finite outage ceiling. A later successful Apply is necessary; later checks and independent egress remain distinct evidence.',
                producer='producer-operations.jsonl (or its lossless .gz)',windows=windows(rows,dict(renewal=renewal,expiry=expiry)),
                all_losses=dict(renewal=sum(p['result']!='ok' or not ns(p,'egress_ns') for p in renewal),expiry=sum(p['result']!='ok' or not ns(p,'egress_ns') for p in expiry)),
                limitation='Source producer has no desired-accept/branch event. Stream coverage is only its captured interval, never continuous coverage of every baseline. No overlapping or missing packet is exempted.')
    (root/'policy-missing-analysis.json').write_text(json.dumps(result,indent=2)+'\n')
    return result


if __name__=='__main__':
    import sys
    review(Path(sys.argv[1]))
