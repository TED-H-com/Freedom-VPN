#!/usr/bin/env python3
"""Join original stops to persistent parent wait receipts, separately from PID1."""
import json
from pathlib import Path
from candidate_evidence import wall_ns
from evidence_io import read_json, redact_error, redact_text
from review_unexecuted import populated_holds, present


def matched(identity, receipt):
    return bool(identity.get('launch_id') and identity.get('start_ticks') and
        receipt.get('id')==identity['launch_id'] and receipt.get('pid')==identity.get('pid') and
        receipt.get('start_ticks')==identity['start_ticks'])


def _review(root):
    read=lambda name:read_json(root/name)
    events=read('container-lifecycle-events.json')['events']; rows=[]
    unavailable_stages=[]
    populated, unavailable = populated_holds(root)
    if unavailable:
        unavailable_stages.append(unavailable)
    for label,name,start,end in [('whole_container','restart-lifecycle.json','stop_started','stop_completed'),('clone_source','identity-freeze.json','started_at','completed_at')]:
        if not present(root, name):
            unavailable_stages.append(dict(status='unavailable', stage=label, reason='stop stage not reached'))
            continue
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
        if source.get('status') in ('not created', 'unavailable') or not source.get('output'):
            unavailable_stages.append(source)
            continue
        try:s=json.loads(source['output'])
        except (ValueError, TypeError):s={}
        receipt=s.get('parent_wait_receipt') or {}
        rows.append(dict(label='direct_agent',container=source['container'],source='receiver-closure.json',identity={k:s.get(k) for k in ('launch_id','pid','start_ticks')},
            parent_wait_receipt=receipt,parent_identity_matched=matched(s,receipt),attempted_escalation=s.get('attempted_escalation'),
            signal_start=s.get('signal_start'),signal_return=s.get('signal_return'),exit_sample=s.get('exit_sample'),signal_to_exit_upper_ns=s.get('signal_to_exit_upper_ns'),within_15s=s.get('within_15s'),
            agent_budget_seconds=15,stored_lease_survived=source.get('stored_lease_survived_stop'),closure_start=source.get('start'),closure_end=source.get('end'),command_ok=source.get('ok'),parse_ok=bool(s),exit_observed=s.get('exit_observed'),signal_ok=s.get('signal_ok'),step_cancel=s.get('step_cancel')))
    if populated is not None:
        if populated['run_id']!=read('AT13.json')['run_id']:raise ValueError('populated stops from another run')
        for attempt in populated['attempts']:
            if not attempt.get('stop'):continue
            try:s=json.loads(attempt['stop']['output'])
            except (ValueError, TypeError):s={}
            receipt=s.get('parent_wait_receipt') or {}
            rows.append(dict(label='populated_hold_agent',source='populated-holds.json',incident_id=attempt['incident_id'],
                identity={k:s.get(k) for k in ('launch_id','pid','start_ticks')},parent_wait_receipt=receipt,
                parent_identity_matched=matched(s,receipt),attempted_escalation=s.get('attempted_escalation'),
                signal_start=s.get('signal_start'),signal_return=s.get('signal_return'),exit_sample=s.get('exit_sample'),
                signal_to_exit_upper_ns=s.get('signal_to_exit_upper_ns'),within_15s=s.get('within_15s'),agent_budget_seconds=15,step_cancel=s.get('step_cancel')))
    for row in rows:
        receipt=row['parent_wait_receipt'];row['agent_exit_code']=receipt.get('exit_code');row['agent_signal']=receipt.get('signal')
        row['cooperative_exit']=row['parent_identity_matched'] and receipt.get('exit_code')==0 and receipt.get('signal') is None
    out=dict(run_id=read('AT13.json')['run_id'],all_parent_receipts_matched=not unavailable_stages and bool(rows) and all(r['parent_identity_matched'] for r in rows),stops=rows,
        status='unavailable' if unavailable_stages else 'reviewed', unavailable_stages=unavailable_stages,
        scope='Each original stop only. Parent wait receipts supply actual agent exit code/signal. Container exit means PID1 handler completion. Escalation sidecars are intent only. Receipt and pidfd times are observation bounds, not exact kernel exit instants; Step cancellation is unavailable. Historical 10.087s/20.153s exit-137 rows are unchanged.')
    (root/'stop-receipt-review.json').write_text(redact_text(json.dumps(out,indent=2)+'\n'))
    return out


def review(root, expected_run=None):
    """Historical callers retain their scope; new invocations require sealed inputs."""
    manifest = None
    try:
        if expected_run is not None or (root/'auxiliary-inputs.json').exists():
            from auxiliary_inputs import inputs
            manifest, sealed = inputs(root, expected_run)
        result = _review(root)
        if manifest is not None:
            errors = list(sealed.get('container-lifecycle-events.json', {}).get('errors', []))
            if 'container-lifecycle-events.json' not in sealed:
                errors.append('lifecycle evidence unavailable')
            if result['run_id'] != manifest['run_id']:
                errors.append('stop review run mismatch')
            direct = [r for r in result['stops'] if r['label'] == 'direct_agent']
            if sorted(r['container'] for r in direct) != ['fvpn-w4-isolation-node', 'fvpn-w4-isolation-node-clone']:
                errors.append('source and clone direct stop rows required')
            for row in result['stops']:
                receipt = row['parent_wait_receipt']
                lo = row.get('closure_start', row.get('command_start'))
                hi = row.get('closure_end', row.get('command_end'))
                # Populated hold receipt times are independently checked by its
                # existing reviewer; here they must still lie in this invocation.
                try:
                    same_window = (wall_ns(manifest['started_at']) <= wall_ns(receipt['started_at']) <=
                                   wall_ns(receipt['wait_observed_at']) <= wall_ns(manifest['completed_at']))
                    if lo and hi:
                        same_window = same_window and wall_ns(lo) <= wall_ns(receipt['wait_observed_at']) <= wall_ns(hi)
                except (KeyError, TypeError, ValueError):
                    same_window = False
                row['same_run_parent_window'] = same_window
                row['final_stop_matched'] = (row['parent_identity_matched'] and same_window and
                    (row['label'] != 'direct_agent' or (row.get('command_ok') is True and
                     row.get('parse_ok') is True and row.get('signal_ok') is True and row.get('exit_observed') is True)))
                if not row['final_stop_matched']:
                    errors.append('unavailable or mismatched parent receipt: '+row.get('container', row['label']))
            result['errors'] = errors
            result['build'], result['source_dirty'] = manifest['build'], manifest['source_dirty']
            result['input_manifest'] = 'auxiliary-inputs.json'
            result['all_parent_receipts_matched'] = result['all_parent_receipts_matched'] and not errors
            if errors:
                result['status'] = 'unavailable'
    except (OSError, ValueError, TypeError, KeyError) as error:
        result = dict(run_id=expected_run or (manifest or {}).get('run_id'), status='unavailable',
                      all_parent_receipts_matched=False, stops=[], unavailable_stages=[],
                      errors=[redact_error(error)])
    (root/'stop-receipt-review.json').write_text(redact_text(json.dumps(result, indent=2)+'\n'))
    return result


if __name__=='__main__':
    import sys
    r=review(Path(sys.argv[1]), sys.argv[2] if len(sys.argv)>2 else None)
    print(json.dumps(r, indent=2))
    raise SystemExit(0 if r['all_parent_receipts_matched'] else 1)
