#!/usr/bin/env python3
"""Focused transport, admission, restart and node-write receipts from one run."""
import json

from followup18_records import boundaries
from run import PUBLIC, write_json


def collect_writes(value, path='$'):
    rows=[]
    if isinstance(value,dict):
        for key,child in value.items():
            if key=='node_write_receipt':
                rows.append(dict(source=path+'.'+key, **child))
            else: rows.extend(collect_writes(child,path+'.'+key))
    elif isinstance(value,list):
        for i,child in enumerate(value): rows.extend(collect_writes(child,f'{path}[{i}]'))
    return rows


def main():
    live=json.loads((PUBLIC/'followup18-transports-live.json').read_text())
    cases=live['transport_cases']
    old=cases.get('lease_expiry',{})
    control=cases.get('control_lost_agent_alive',{})
    samples=control.get('samples',[])
    bounds=control.get('bounds',{})
    control_edges={}
    if samples:
        control_edges=boundaries(dict(samples=[s['transport'] for s in samples],deadline_elapsed_seconds=bounds['deadline_monotonic_seconds']))
    snapshots=live.get('restart_snapshots',[])
    watchdog=[]
    seen=set()
    for snap in snapshots:
        for line in snap.get('watchdog',{}).get('numbered_lines',[]):
            if line not in seen:
                seen.add(line);watchdog.append(line)
    restarts=[]
    for name in ('grant_expiry','lease_expiry','control_lost_agent_alive','device_revoke','manual_grant_withdrawal'):
        pair=[s for s in snapshots if s['case_boundary'].startswith(name+' ')]
        if len(pair)==2:
            restarts.append(dict(case=name,restarted=pair[0]['agent_pid_start_ticks_state'].split()[:2]!=pair[1]['agent_pid_start_ticks_state'].split()[:2],
                                 before=pair[0],after=pair[1]))
    decision_lines=[line for line in watchdog if 'baseline decision' in line]
    agent_lines=[line for snap in snapshots for line in snap.get('agent',{}).get('numbered_lines',[])]
    writes=collect_writes(live)
    unique={tuple(a.get('idempotency_key') for a in r.get('attempts',[])):r for r in writes}
    conflicts=[dict(source=r['source'],attempt=a) for r in unique.values() for a in r.get('attempts',[]) if a.get('status') in (409,412)]
    write_json(PUBLIC/'followup18-node-writes.json',dict(run_id=live['run_id'],receipts=writes,unique_command_receipts=len(unique),conflicts=conflicts))
    out=dict(run_id=live['run_id'],old_case=dict(status=old.get('status'),
                lease_before_fault=old.get('lease_before_fault'),lease_at_fault=old.get('lease_at_fault'),
                heartbeat_at_fault=old.get('heartbeat_at_fault'),fault_lease_read=old.get('fault_lease_read'),
                deadline_monotonic_ns=old.get('deadline_monotonic_ns'),
                control_stop_started_at=old.get('control_stop_started_at'),control_stop_completed_at=old.get('control_stop_completed_at'),
                agent_kill_started_at=old.get('agent_kill_started_at'),agent_kill_completed_at=old.get('agent_kill_completed_at'),
                transport_boundaries=boundaries(old),timed_section=old.get('timed_section')),
             control_loss=dict(status=control.get('status'),lease=control.get('signed_lease'),cap_setup=control.get('lease_setup'),
                bounds=bounds,verdict=control.get('verdict'),transport_boundaries=control_edges,
                control_loss_started_at=control.get('control_loss_started_at'),control_stop_completed_at=control.get('control_stop_completed_at'),
                sample_count=len(samples),identities=sorted(set(s['agent_pid_start_ticks'] for s in samples)),
                all_samples_alive=all(s['agent_alive'] for s in samples),timeout_lines=control.get('agent_timeout_log'),
                after_heartbeat=control.get('after_heartbeat'),agent_after_recovery=control.get('agent_after_recovery'),
                final_continuity_sample=next((s for s in reversed(samples) if s['read_end_monotonic_seconds']<=bounds.get('continuity_end_monotonic_seconds',0)),None),
                restoration={k:v for k,v in control.items() if 'restor' in k or 'restart_ok' in k or 'unpause_ok' in k}),
             restarts=restarts,watchdog_lines=watchdog,watchdog_decisions=decision_lines,
             baseline_unverified_observed=any('baseline_unverified' in line for line in watchdog),
             generation_mismatch_observed=any('GENERATION_MISMATCH' in line for line in agent_lines))
    write_json(PUBLIC/'followup18-diagnostic.json',out)
    print('diagnostics: restarts',sum(r['restarted'] for r in restarts),'node_write conflicts',len(conflicts))


if __name__=='__main__': main()
