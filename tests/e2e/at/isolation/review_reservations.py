#!/usr/bin/env python3
"""AT13 pre-Apply reservation brackets and receipt/recovery joins (core 34)."""
import json
from evidence_io import read_json
from pathlib import Path
from producer_timing import read_operations, identity
from candidate_evidence import wall_ns


def elapsed_from_wall(row, mono, wall):
    a=row['anchor']; w=wall_ns(a['wall']); receipt=wall_ns(wall)
    return dict(lower_ns=str(w+int(mono)-int(a['mono_after_ns'])-receipt),
                upper_ns=str(w+int(mono)-int(a['mono_before_ns'])-receipt))


def bracket(row):
    t=row['timing'];before=row.get('host',{}).get('before_apply',{})
    if not before.get('ok'):return dict(available=False)
    start=int(before['end_ns']);end=int(t['start_ns'])
    if end<start:raise ValueError('pre-Apply bracket inverted')
    return dict(available=True,start_ns=str(start),end_ns=str(end),duration_ns=str(end-start),
        contains='conntrack reservation plus common setup when queued addresses exist; not isolated transaction latency')


def review(root):
    read=lambda name:read_json(root/name)
    rows=read_operations(root);wire=read('receiver-capture.json')['transactions']
    applies=[r for r in rows if r.get('timing',{}).get('operation')=='apply']
    indexed={(r['run_id'],r['sequence']):r for r in applies}
    inventory=[];pending=[]
    for r in applies:
        t=r['timing'];b=bracket(r)
        commands=[dict(command_index=i,phase=c['phase'],kind=c['kind'],ok=c['ok'],start_attempt=c['start_attempt'],wait_return=c['wait_return']) for i,c in enumerate(t.get('commands') or []) if c['phase']=='revoke']
        item=dict(identity(r),candidate_generation=t['candidate_generation'],succeeded=t['succeeded'],verified=t['verified'],pre_apply=b,
            pending_evidence='nonempty batch proved by original revoke commands' if commands else 'no revoke command observed; pre-install failure cannot exclude queued work',revoke_commands=commands,
            install_start_ns=t['deadline']['install_start_ns'],installed_by_ns=t['deadline']['installed_by_ns'],apply_end_ns=t['end_ns'])
        inventory.append(item)
        if not commands:continue
        # First retained desired delivery of this candidate before this Apply.
        deliveries=[(i,w) for i,w in enumerate(wire) if w.get('status')==200 and 'desired-state' in w.get('path','') and
            (w.get('response') or {}).get('signed',{}).get('config_generation')==t['candidate_generation'] and wall_ns(w['response_at'])<=wall_ns(r['anchor']['wall'])]
        join=dict(item,desired_delivery=None)
        if deliveries:
            i,w=deliveries[0]; receipt=w['response_at']
            join['desired_delivery']=dict(transaction_index=i,request_received_at=w['received_at'],response_completed_at=receipt,
                signed_generation=w['response']['signed']['config_generation'],
                receipt_to_pre_apply_start=elapsed_from_wall(r,b['start_ns'],receipt) if b['available'] else None,
                receipt_to_apply_start=elapsed_from_wall(r,t['start_ns'],receipt),
                receipt_to_restricting_install_end=elapsed_from_wall(r,t['deadline']['installed_by_ns'],receipt) if int(t['deadline']['installed_by_ns']) else None,
                receipt_to_apply_end=elapsed_from_wall(r,t['end_ns'],receipt))
            baselines=[p for p in rows if p['run_id']==r['run_id'] and p['source']=='baseline' and
                int(p['baseline']['end_ns'])<=int(t['start_ns']) and int(elapsed_from_wall(p,p['baseline']['start_ns'],receipt)['lower_ns'])>=0]
            if baselines:
                p=baselines[0]
                join['first_baseline_after_receipt']=dict(identity(p),baseline=p['baseline'],receipt_to_baseline_start=elapsed_from_wall(p,p['baseline']['start_ns'],receipt),receipt_to_baseline_end=elapsed_from_wall(p,p['baseline']['end_ns'],receipt))
        pending.append(join)
    recovery=[]
    policy=read('policy-missing-analysis.json')
    for window in policy['windows']:
        a=window.get('next_successful_apply')
        if not a:continue
        raw=indexed[(a['run_id'],a['sequence'])]
        b=bracket(raw);cmds=[i for i,c in enumerate(raw['timing'].get('commands') or []) if c['phase']=='revoke']
        recovery.append(dict(run_id=window['run_id'],baseline_sequence=window['sequence'],old_lease=window['old_lease'],
            baseline_start_ns=window['baseline_start_ns'],baseline_to_apply_ns=window['baseline_to_apply_ns'],next_apply=identity(raw),
            pre_apply=b,revoke_command_indices=cmds,contains_proven_reservation=bool(cmds),
            next_live=identity(window['next_live']) if window.get('next_live') else None,
            next_serving_heartbeat=identity(window['next_serving_heartbeat']) if window.get('next_serving_heartbeat') else None,
            packets=window['packets']))
    # Exact same-monotonic-clock overlap of Apply work with the packet windows.
    packet_windows={}
    for name,file in [('whole_node','whole-node-probes.json'),('renewal','renewal-probes.json')]:
        ps=read(file);ps=ps['probes'] if isinstance(ps,dict) else ps
        lo=min(int(p['send']['mono_ns']) for p in ps);hi=max(int(p['read_completion']['mono_ns']) for p in ps)
        selected=[i for i in inventory if i['pre_apply'].get('available') and int(i['pre_apply']['start_ns'])<=hi and int(i['apply_end_ns'])>=lo]
        packet_windows[name]=dict(start_ns=str(lo),end_ns=str(hi),applies=len(selected),nonempty_batches=[{k:i[k] for k in ('run_id','boot_id','sequence','pre_apply','revoke_commands')} for i in selected if i['revoke_commands']],failed_applies=[i for i in selected if not i['succeeded']],
            scope='A pure renewal/expiry Step adds no reservation. Nonempty batches are identified by revoke commands; every pre-Apply bracket is retained, including failures whose queue is not exported.')
    out=dict(run_id=read('AT13.json')['run_id'],bracket_read='host.before_apply.end_ns → timing.start_ns',
        scope='Agent-core 34 reservation precedes Apply and the restricting install; never hidden inside guard percentiles. All Apply pre-brackets retained. Desired proxy response completion is a delivery observation, not exact node parse/accept time. Anchor intervals preserve clock uncertainty. A baseline can revoke before the subsequent reservation; recovery still includes it.',
        applies=inventory,nonempty_batches=pending,recovery_windows=recovery,packet_windows=packet_windows,
        revoke_request=read('revocation-request.json'),restart=read('restart-lifecycle.json'),
        limitation='Queued-address identities are not exported before a failed install. No absence claim is made for those failures. Mechanism benchmarks are not an assembled pass, and no transaction duration is inferred by subtracting an unrelated run.')
    (root/'pre-apply-reservation-review.json').write_text(json.dumps(out,indent=2)+'\n')
    return out


if __name__=='__main__':
    import sys
    r=review(Path(sys.argv[1]));print(json.dumps(dict(run_id=r['run_id'],applies=len(r['applies']),nonempty_batches=len(r['nonempty_batches']),recovery_windows=len(r['recovery_windows']),packet_windows=r['packet_windows']),indent=2))
