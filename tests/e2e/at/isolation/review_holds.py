"""Read-only joins for wire-nodes 34; absent reports never become evidence."""
import gzip
import json
from pathlib import Path
from candidate_evidence import wall_ns
from producer_timing import identity, read_operations


def read(root, name):
    path = root/name
    raw = path.read_bytes() if path.exists() else gzip.decompress(Path(str(path)+'.gz').read_bytes())
    return json.loads(raw)


def ns(value):
    return wall_ns(value.replace('+00:00', 'Z'))


def marker(r):
    return 'policy_missing' in (r.get('forwarding_block_reasons') or []) and (
        any(c.get('name') == 'lease' and c.get('error_code') == 'STALE_GENERATION' for c in r.get('checks', [])) or
        any(e.get('stage') == 'lease' and e.get('code') == 'STALE_GENERATION' for e in r.get('errors', [])))


def independent_failure(r):
    for c in r.get('checks', []):
        if c.get('status') != 'failed':
            continue
        if c.get('name') == 'lease' and c.get('error_code') == 'STALE_GENERATION':
            continue
        if c.get('error_code') == 'VALIDATION_FAILED' and (c.get('name'), c.get('detail')) in (
                ('qos', 'Rate policy could not be applied.'), ('source_binding', 'No member binding was admitted.')):
            continue
        return True
    return any(e.get('stage') != 'lease' or e.get('code') != 'STALE_GENERATION' for e in r.get('errors', []))


def joins(central):
    final = central['final']
    incidents = {h['id']: h for h in final['incidents']}
    samples = {s['report_id']: s for s in final['samples']}
    reports = {row['report']['report_id']: row for row in central['reports']}
    problems, holds = [], []
    for rid, row in reports.items():
        r = row['report']
        if not marker(r):
            continue
        s = samples.get(rid)
        h = incidents.get(s['hold_id']) if s else None
        lease = (row.get('lease') or {}).get('signed', {})
        errors = []
        if not s or not h:
            errors.append('accepted hold has no retained sample/incident')
        else:
            for field, expected in [('node_id', r['node_id']), ('incarnation_id', r['node_incarnation_id']),
                                    ('boot_id', r['boot_id']), ('lease_id', r['lease']['lease_id']),
                                    ('applied_generation', r.get('applied_config_generation'))]:
                if s[field] != expected:
                    errors.append('sample mismatch: '+field)
            for field, expected in [('lease_id', s['lease_id']), ('node_id', s['node_id']),
                                    ('node_incarnation_id', s['incarnation_id']), ('config_generation', s['lease_generation'])]:
                if lease.get(field) != expected:
                    errors.append('own signed lease mismatch: '+field)
            if row.get('stored_lease_generation') != s['lease_generation']:
                errors.append('stored lease generation mismatch')
            if not row['lease'].get('signatures'):
                errors.append('signed lease envelope absent')
        cause = row.get('publication_data_path_failure')
        unrelated = independent_failure(r)
        if not row.get('classification_available') or not row.get('reverse_array_replay_available'):
            errors.append('classification unavailable')
        if cause != row.get('reverse_array_replay_cause'):
            errors.append('array order changed classification')
        if unrelated and cause not in ('check_failed', 'isolation_unconfirmed'):
            errors.append('unrelated failure lost precedence')
        if not unrelated and cause != 'stale_generation':
            # isolation status/dependency evidence must still take precedence.
            if cause != 'isolation_unconfirmed':
                errors.append('unexpected hold classification')
        if r.get('member_forwarding_enabled') is not False:
            errors.append('hold reported member forwarding')
        # A pre-existing succeeded publication is historical. Only a completion
        # referencing this exact report could falsely credit the hold's ACK.
        credited = []
        for op in row.get('operations_at_read', []):
            for name, step in (op.get('steps') or {}).items():
                if name == 'data_path_probe' and step.get('status') == 'succeeded' and rid in json.dumps(step):
                    credited.append(op['id'])
        if credited:
            errors.append('publication data path credited the hold report')
        holds.append(dict(report_id=rid, observation_id=row['observation_id'], received_at=row['received_at'],
            observed_at=r['observed_at'], lease_id=r['lease']['lease_id'], signed_lease_generation=lease.get('config_generation'),
            applied_generation=r.get('applied_config_generation'), desired_generation_at_read=row['desired_generation_at_read'],
            applied_equals_desired_at_read=r.get('applied_config_generation') == row['desired_generation_at_read'],
            sample=s, incident=h, classification=cause, reverse_array_replay_cause=row.get('reverse_array_replay_cause'),
            unrelated_failure=unrelated, reported_forwarding=r.get('member_forwarding_enabled'),
            publication_operation_snapshots=row.get('operations_at_read', []), false_publication_credit=credited,
            audit_ids=[a['id'] for a in final['audits'] if h and a['evidence']['incident_id'] == h['id']], problems=errors))
        problems.extend(dict(report_id=rid, problem=e) for e in errors)
    for rid in samples:
        if rid not in reports:
            problems.append(dict(report_id=rid, problem='durable sample missing its accepted original report'))
    ends, escalations = [], []
    for hid, h in incidents.items():
        audit = [a for a in final['audits'] if a['evidence']['incident_id'] == hid]
        if len(audit) != (1 if h['escalated_at'] else 0):
            problems.append(dict(incident_id=hid, problem='escalation count differs from durable state'))
        for a in audit:
            ev = a['evidence']; trigger = ev.get('alarm_report_id')
            alarm = samples.get(trigger) if trigger else None
            if trigger and (not alarm or not alarm['summary_valid'] or alarm['reported_alarm'] is not True):
                problems.append(dict(incident_id=hid, problem='alarm audit not joined to valid reported true'))
            age = ns(a['occurred_at']) - ns(h['first_seen_at'])
            if not trigger and age < 30_000_000_000:
                problems.append(dict(incident_id=hid, problem='central-age audit before 30 seconds'))
            escalations.append(dict(incident_id=hid, audit=a, trigger='reported_alarm' if trigger else 'central_age_30s', age_ns=str(age)))
        if h['end_reason'] == 'generation_matched':
            matches = []
            for rid, row in reports.items():
                r = row['report']; lease = (row.get('lease') or {}).get('signed', {})
                if (ns(row['received_at']) == ns(h['ended_at']) and not marker(r) and lease.get('config_generation')
                        and lease['config_generation'] == r.get('applied_config_generation')):
                    matches.append(rid)
            if not matches:
                problems.append(dict(incident_id=hid, problem='generation_matched has no exact matching-lease receipt'))
            later = [i['id'] for i in incidents.values() if ns(i['first_seen_at']) > ns(h['ended_at'])]
            ends.append(dict(incident_id=hid, ended_at=h['ended_at'], matching_report_ids=matches, later_distinct_incident_ids=later))
    return dict(holds=holds, incidents=list(incidents.values()), escalations=escalations, matching_ends=ends, problems=problems)


def review(root):
    central = read(root, 'generation-holds-central.json')
    result = joins(central)
    wire = read(root, 'receiver-capture.json')['transactions']
    for hold in result['holds']:
        report=next(row['report'] for row in central['reports'] if row['report']['report_id']==hold['report_id'])
        delivered=[]
        for i, transaction in enumerate(wire):
            signed=(transaction.get('response') or {}).get('signed',{})
            if ('desired-state' in transaction.get('path','') and transaction.get('status')==200 and
                signed.get('node_id')==report['node_id'] and signed.get('node_incarnation_id')==report['node_incarnation_id'] and
                ns(transaction['response_at'])<=ns(report['observed_at'])):
                delivered.append((i,transaction))
        latest=max(delivered,key=lambda item:ns(item[1]['response_at'])) if delivered else None
        hold['desired_wire']=dict(transaction_index=latest[0],transaction=latest[1]) if latest else None
        hold['desired_generation']=latest[1]['response']['signed']['config_generation'] if latest else None
        hold['applied_equals_delivered_desired']=bool(latest and hold['applied_generation']==hold['desired_generation'])
        hold['desired_scope']='Latest original signed desired response completed before this report; central desired_generation_at_read is a separate later polling value.'
    accepted = {w['body']['report_id'] for w in wire if w.get('status') == 202 and (w.get('body') or {}).get('report_id')}
    operations = read(root, 'publication-operation-history.json')
    if operations['run_id'] != central['run_id']:
        raise ValueError('operation snapshots belong to another run')
    completions = {}
    hold_ids = {h['report_id'] for h in result['holds']}
    for snapshot in operations['snapshots']:
        for op in snapshot['operations']:
            if op['kind'] != 'node.publish' or not op['completed_at']:
                continue
            overlaps = [h['id'] for h in result['incidents'] if ns(h['first_seen_at']) <= ns(op['completed_at']) and
                        (h['ended_at'] is None or ns(op['completed_at']) < ns(h['ended_at']))]
            refs = op.get('observed_refs', {})
            credits = [rid for rid in hold_ids if rid in refs.get('data_path_probe', '')]
            completions[op['id']] = dict(operation=op, open_incidents_at_completion=overlaps, hold_report_credits=credits)
            if op['status'] == 'succeeded' and (overlaps or credits):
                problem=dict(operation_id=op['id'], problem='publication completed during a retained hold or credited its report')
                if problem not in result['problems']: result['problems'].append(problem)
    result['publication_completions'] = list(completions.values())
    result['publication_snapshot_scope'] = dict(first_read=operations['snapshots'][0]['read_started'],
        final_read=operations['last_successful_read'], end=operations.get('end'),
        limitation='The sidecar began during this assembled run; initial completed operations are retained state, not continuously observed earlier transitions.')
    result['accepted_hold_reports_without_wire_receipt'] = [h['report_id'] for h in result['holds'] if h['report_id'] not in accepted]
    windows = read(root, 'policy-missing-analysis.json')['windows']
    producer = {(r['run_id'], r['sequence']): r for r in read_operations(root)}
    for hold in result['holds']:
        matching=[r for r in producer.values() if r.get('lease',{}).get('lease_id')==hold['lease_id']]
        hold['producer_baselines']=[dict(**identity(r),baseline=r['baseline'],anchor=r['anchor']) for r in matching if r.get('baseline',{}).get('cause')=='policy_missing']
        hold['producer_heartbeats']=[dict(**identity(r),heartbeat=r['heartbeat'],anchor=r['anchor']) for r in matching if r.get('heartbeat',{}).get('reason')=='policy_missing']
        hold['independent_packet_coverage']='not exercised for this hold unless separately retained in generation-hold-control-cut.json; report forwarding=false is not independent packet evidence'
    joined = []
    for w in windows:
        if not w['stale_populated_lease_candidate']:
            continue
        b = producer[w['run_id'], w['sequence']]
        anchor = b['anchor']; offset = ns(anchor['wall']) - int(anchor['mono_before_ns'])
        lo, hi = int(w['baseline_start_ns'])+offset, int(w['apply_end_ns'])+offset if w['apply_end_ns'] else None
        ids = [h['report_id'] for h in result['holds'] if h['lease_id'] == w['old_lease']['lease_id'] and
               lo <= ns(h['observed_at']) and (hi is None or ns(h['observed_at']) <= hi)]
        joined.append(dict(**identity(b), lease_id=w['old_lease']['lease_id'], lease_generation=w['old_lease']['generation'],
            baseline_start_ns=w['baseline_start_ns'], apply_end_ns=w['apply_end_ns'], accepted_hold_report_ids=ids,
            same_lease_hold_report_ids=[h['report_id'] for h in result['holds'] if h['lease_id'] == w['old_lease']['lease_id']],
            central_join='by report ID and own lease' if ids else 'not observed: no accepted hold report in this producer bracket',
            baseline_rows=[dict(**identity(a), **a['baseline']) for a in w['baseline_rows']],
            packet_coverage=w['packets'], later_recovery=w['recovery'], wall_join='producer wall/monotonic anchor, no allowance'))
    result['populated_generation_windows'] = joined
    cut = read(root, 'generation-hold-control-cut.json')
    retention = dict(status='not exercised', kernel_reboot='not exercised: AT13 operator matrix', cut_status=cut['status'])
    if cut.get('direct_stop'):
        hid = cut['incident_id']
        snapshots = {}
        for name in ('before_direct_stop', 'after_direct_stop', 'after_restoration'):
            snap = cut[name]
            snapshots[name] = dict(incident=next((h for h in snap['incidents'] if h['id'] == hid), None),
                audit_ids=[a['id'] for a in snap['audits'] if a['evidence']['incident_id'] == hid])
        retained = all(s['incident'] for s in snapshots.values())
        first_seen = {s['incident']['first_seen_at'] for s in snapshots.values() if s['incident']}
        before = [h['sample'] for h in result['holds'] if h['sample'] and h['sample']['hold_id'] == hid and ns(h['observed_at']) < ns(cut['direct_stop']['start'])]
        after = [h['sample'] for h in result['holds'] if h['sample'] and h['sample']['hold_id'] == hid and ns(h['observed_at']) >= ns(cut['process_restart']['start'])]
        counters_reset = bool(before and after and any(s['reported_steps'] is not None and after[0]['reported_steps'] is not None and
            int(s['reported_steps']) > int(after[0]['reported_steps']) for s in before))
        retention.update(status='observed' if retained and len(first_seen) == 1 else 'fail', snapshots=snapshots,
            counters_before=before, counters_after=after, local_counters_reset='observed' if counters_reset else 'not exercised in accepted samples',
            direct_stop=cut['direct_stop'], process_restart=cut['process_restart'])
        if retention['status'] == 'fail':
            result['problems'].append(dict(incident_id=hid, problem='incident/first_seen lost across direct stop'))
    result['applied_equals_delivered_desired']=sum(h['applied_equals_delivered_desired'] for h in result['holds'])
    result.update(run_id=central['run_id'], status='fail' if result['problems'] else 'pass', retention=retention,
        accepted_hold_count=len(result['holds']), applied_equals_desired_at_read=sum(h['applied_equals_desired_at_read'] for h in result['holds']),
        scope='Central row joins and classification, including reverse-array read-only replay. Poll snapshots are not ingestion transaction snapshots. Baseline writes, packet capture and publication completion are separate evidence; no inference from a counter read failure.',
        counter_read_failure_decision=dict(source='docs/poc/G2-followup23-evidence/decision.json',kind='duplicate_chain',error_type='*enforce.DuplicateChainError',duplicates=0,listings=3300,origin='unproven',historical_failures='unchanged; no retry waiver or reclassification'))
    result['coverage'] = dict(
        exact_report_sample_incident_join=('fail' if result['problems'] else 'pass' if result['holds'] else 'not exercised: no accepted hold report'),
        stale_generation_before_generic_causes='pass' if any(h['classification']=='stale_generation' for h in result['holds']) else 'not exercised',
        applied_equals_desired='observed against the node latest delivered signed desired state' if result['applied_equals_delivered_desired'] else 'did not occur',
        unrelated_failure_precedence='pass' if any(h['unrelated_failure'] for h in result['holds']) and not result['problems'] else 'not exercised',
        active_hold_direct_stop_and_local_counter_reset='pass' if retention.get('local_counters_reset')=='observed' and retention['status']=='observed' else 'not exercised',
        matching_lease_end='pass' if result['matching_ends'] and not result['problems'] else 'not exercised',
        later_new_incident='pass' if any(e['later_distinct_incident_ids'] for e in result['matching_ends']) else 'not exercised',
        one_escalation='pass' if result['escalations'] and not result['problems'] else 'not exercised',
        kernel_reboot='not exercised: AT13 operator matrix')
    if not result['problems'] and any(v.startswith('not exercised') for v in result['coverage'].values()):
        result['status']='partial'
    (root/'generation-holds-review.json').write_text(json.dumps(result,indent=2)+'\n')
    return result


if __name__ == '__main__':
    import sys
    result = review(Path(sys.argv[1]))
    print(json.dumps({k:result[k] for k in ('run_id','status','accepted_hold_count','applied_equals_delivered_desired','applied_equals_desired_at_read','problems')},indent=2))
    raise SystemExit(bool(result['problems']))
