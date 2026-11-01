#!/usr/bin/env python3
"""Join original stops to persistent parent wait receipts, separately from PID1."""
import json
from pathlib import Path
from candidate_evidence import wall_ns
from evidence_io import read_json


def matched(identity, receipt):
    return bool(identity.get('launch_id') and identity.get('start_ticks') and
        receipt.get('id')==identity['launch_id'] and receipt.get('pid')==identity.get('pid') and
        receipt.get('start_ticks')==identity['start_ticks'])


def review(root):
    read=lambda name:read_json(root/name)
    events=read('container-lifecycle-events.json')['events']; rows=[]
    for label,name,start,end in [('whole_container','restart-lifecycle.json','stop_started','stop_completed'),('clone_source','identity-freeze.json','started_at','completed_at')]:
        source=read(name);stop=source['agent_stop'];identity=stop['identity_before_stop'];receipt=stop.get('parent_wait_receipt') or {}
        lo,hi=wall_ns(source[start]),wall_ns(source[end])
        container=[e for e in events if e['attributes']['name']=='fvpn-w4-isolation-node' and lo<=int(e['time_ns'])<=hi]
        row=dict(label=label,source=name,identity=identity,parent_wait_receipt=receipt,parent_identity_matched=matched(identity,receipt),
            attempted_escalation=stop.get('attempted_escalation'),container_events=container,container_state_at_read=stop.get('container_state_at_read'),
            agent_budget_seconds=15,container_grace_seconds=25,command_start=source[start],command_end=source[end])
        terms=[e for e in container if e['action']=='kill' and e['attributes'].get('signal')=='15']
        if terms and receipt.get('wait_observed_at'):
            row['container_term_to_agent_parent_wait_ns']=str(wall_ns(receipt['wait_observed_at'])-int(terms[0]['time_ns']))
        rows.append(row)
    for source in read('receiver-closure.json'):
        try:s=json.loads(source['output'])
        except ValueError:s={}
        receipt=s.get('parent_wait_receipt') or {}
        rows.append(dict(label='direct_agent',container=source['container'],source='receiver-closure.json',identity={k:s.get(k) for k in ('launch_id','pid','start_ticks')},
            parent_wait_receipt=receipt,parent_identity_matched=matched(s,receipt),attempted_escalation=s.get('attempted_escalation'),
            signal_start=s.get('signal_start'),signal_return=s.get('signal_return'),exit_sample=s.get('exit_sample'),signal_to_exit_upper_ns=s.get('signal_to_exit_upper_ns'),within_15s=s.get('within_15s'),
            agent_budget_seconds=15,stored_lease_survived=source['stored_lease_survived_stop'],step_cancel=s.get('step_cancel')))
    for row in rows:
        receipt=row['parent_wait_receipt'];row['agent_exit_code']=receipt.get('exit_code');row['agent_signal']=receipt.get('signal')
        row['cooperative_exit']=row['parent_identity_matched'] and receipt.get('exit_code')==0 and receipt.get('signal') is None
    out=dict(run_id=read('AT13.json')['run_id'],all_parent_receipts_matched=all(r['parent_identity_matched'] for r in rows),stops=rows,
        scope='Each original stop only. Parent wait receipts supply actual agent exit code/signal. Container exit means PID1 handler completion. Escalation sidecars are intent only. Receipt and pidfd times are observation bounds, not exact kernel exit instants; Step cancellation is unavailable. Historical 10.087s/20.153s exit-137 rows are unchanged.')
    (root/'stop-receipt-review.json').write_text(json.dumps(out,indent=2)+'\n')
    return out


if __name__=='__main__':
    import sys
    r=review(Path(sys.argv[1]));print(json.dumps(dict(run_id=r['run_id'],all_parent_receipts_matched=r['all_parent_receipts_matched'],stops=[{k:s.get(k) for k in ('label','container','parent_identity_matched','agent_exit_code','agent_signal','signal_to_exit_upper_ns','container_term_to_agent_parent_wait_ns')} for s in r['stops']]),indent=2))
