#!/usr/bin/env python3
"""D-102 original populated incidents; no vacuous passes or synthetic reports."""
import json
from pathlib import Path
from evidence_io import read_json
from review_holds import joins, ns
from review_stops import matched


def incident_at(snapshot, hid):
    return next((h for h in snapshot.get('incidents', []) if h['id'] == hid), None)


def retention(attempt, samples):
    hid = attempt['incident_id']
    snapshots = [incident_at(attempt.get(k, {}), hid) for k in ('before_stop', 'after_stop', 'after_restart')]
    retained = all(h and h['ended_at'] is None for h in snapshots)
    same_first = len({h['first_seen_at'] for h in snapshots if h}) == 1
    before = [s for s in samples if s['hold_id'] == hid and ns(s['received_at']) < ns(attempt['stop']['start'])]
    after = [s for s in samples if s['hold_id'] == hid and ns(s['received_at']) >= ns(attempt['restart']['start'])]
    before.sort(key=lambda s: ns(s['received_at'])); after.sort(key=lambda s: ns(s['received_at']))
    reset = bool(before and after and before[-1].get('reported_steps') and after[0].get('reported_steps') and
                 int(before[-1]['reported_steps']) > int(after[0]['reported_steps']) and
                 int(before[-1]['reported_elapsed_ms']) > int(after[0]['reported_elapsed_ms']))
    try:
        stop = json.loads(attempt['stop']['output'])
    except (ValueError, KeyError):
        stop = {}
    receipt = stop.get('parent_wait_receipt') or {}
    stopped = matched(stop, receipt) and stop.get('exit_observed') is True
    proxy = attempt.get('proxy_after_stop') or {}
    armed = proxy.get('arm', {}).get('id') == attempt['arm']['id'] and proxy.get('hold_started_at') and not proxy.get('released_at')
    return dict(status='pass' if retained and same_first and reset and stopped and armed else 'fail',
                incident_snapshots=snapshots, last_before=before[-1] if before else None,
                first_after=after[0] if after else None, local_counters_reset=reset,
                stop_receipt=receipt, stop_identity_matched=matched(stop, receipt), proxy_still_armed=bool(armed))


def review(root):
    raw = read_json(root/'populated-holds.json')
    central = read_json(root/'generation-holds-central.json')
    if raw['run_id'] != central['run_id']:
        raise ValueError('different runs')
    joined = joins(central)
    final = raw.get('final') or {}
    rows = []
    previous = None
    for attempt in raw['attempts']:
        hid = attempt.get('incident_id')
        h = incident_at(final, hid)
        row = dict(attempt=attempt['attempt'], arm_id=attempt.get('arm', {}).get('id'), incident_id=hid,
                   status='not exercised', reason=attempt.get('reason'), coverage={})
        rows.append(row)
        if not h:
            row['coverage'] = {k:'not exercised' for k in ('populated_hold','alarm_single_escalation','unrelated_failure_precedence','active_stop_retention','matching_recovery','distinct_later_incident')}
            continue
        holds = [v for v in joined['holds'] if v['incident']['id'] == hid]
        samples = [s for s in final.get('samples', []) if s['hold_id'] == hid]
        audits = [a for a in final.get('audits', []) if a['evidence']['incident_id'] == hid]
        report_ids = {s['report_id'] for s in samples}
        alarms = [s for s in samples if s['summary_valid'] and s['reported_alarm'] and
                  (int(s['reported_steps']) >= 3 or int(s['reported_elapsed_ms']) >= 30000)]
        escalation = next((v for v in joined['escalations'] if v['incident_id'] == hid), None)
        ends = next((v for v in joined['matching_ends'] if v['incident_id'] == hid), {})
        matching_ids = ends.get('matching_report_ids', [])
        lease_id = attempt.get('local_lease', {}).get('lease_id')
        forwarded = [e for e in attempt.get('proxy', {}).get('events', []) if e['kind']=='matching_lease_forwarded' and
                     e.get('lease_id')==lease_id and e.get('body_sha256')==attempt.get('local_lease', {}).get('body_sha256')]
        reports = {r['report']['report_id']: r['report'] for r in central['reports']}
        accepted_local = [rid for rid in matching_ids if reports[rid].get('lease', {}).get('lease_id') == lease_id]
        unrelated = [v for v in holds if v['report_id'] == attempt.get('unrelated_report_id') and v['unrelated_failure']]
        denial = attempt.get('denial') or {}
        coverage = dict(
            populated_hold='pass' if holds and attempt.get('before', {}).get('member_count',0)>=2 and all(not v['problems'] for v in holds) else 'fail',
            independent_denial='pass' if denial.get('ingress') and denial.get('result')=='timeout' and denial.get('delivered') is False else 'not exercised',
            alarm_single_escalation='pass' if alarms and len(audits)==1 and escalation and not any(p.get('incident_id')==hid for p in joined['problems']) else 'fail',
            unrelated_failure_precedence='pass' if unrelated and all(v['classification'] in ('check_failed','isolation_unconfirmed') and not v['problems'] for v in unrelated) else 'not exercised',
            active_stop_retention='not exercised',
            matching_recovery='pass' if h['end_reason']=='generation_matched' and accepted_local and forwarded and attempt.get('recovered_counted', {}).get('counted') else 'fail',
            distinct_later_incident='not exercised')
        if attempt.get('stop'):
            row['retention'] = retention(attempt, final.get('samples', []))
            coverage['active_stop_retention'] = row['retention']['status']
        if previous:
            coverage['distinct_later_incident'] = 'pass' if h['id'] != previous['incident_id'] and row['arm_id'] != previous['arm_id'] and previous['end_reason']=='generation_matched' and ns(h['first_seen_at']) > ns(previous['ended_at']) else 'fail'
        row.update(status='fail' if 'fail' in coverage.values() else 'partial' if 'not exercised' in coverage.values() else 'pass',
                   node_id=h['node_id'], incarnation_id=h['incarnation_id'], first_seen_at=h['first_seen_at'], last_seen_at=h['last_seen_at'],
                   ended_at=h['ended_at'], end_reason=h['end_reason'], report_ids=sorted(report_ids), audit_ids=[a['id'] for a in audits],
                   alarm_report_ids=[s['report_id'] for s in alarms], escalation=escalation, matching_report_ids=matching_ids,
                   accepted_local_matching_report_ids=accepted_local, forwarded_local_lease_events=forwarded,
                   unrelated_report_ids=[v['report_id'] for v in unrelated], unrelated_classifications=[v['classification'] for v in unrelated], coverage=coverage)
        previous = row
    required = ('populated_hold','independent_denial','alarm_single_escalation','unrelated_failure_precedence','active_stop_retention','matching_recovery','distinct_later_incident')
    coverage = {k:'fail' if any(r['coverage'].get(k)=='fail' for r in rows) else 'pass' if any(r['coverage'].get(k)=='pass' for r in rows) else 'not exercised' for k in required}
    result = dict(run_id=raw['run_id'], status='pass' if all(v=='pass' for v in coverage.values()) and not joined['problems'] else 'incomplete',
                  attempts=rows, coverage=coverage, join_problems=joined['problems'],
                  scope='Only original accepted reports, own signed leases, exact central incidents and incident-keyed audits. Missing rows remain not exercised. Parent wait exit/signal is observed, never inferred cooperative.')
    (root/'populated-holds-review.json').write_text(json.dumps(result,indent=2)+'\n')
    return result


if __name__ == '__main__':
    import sys
    result = review(Path(sys.argv[1]))
    print(json.dumps({k:result[k] for k in ('run_id','status','coverage','join_problems')},indent=2))
    raise SystemExit(result['status'] != 'pass')
