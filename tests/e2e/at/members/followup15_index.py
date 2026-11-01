"""Index original F15 evidence without changing any live verdict or retrying it."""
from pathlib import Path
from collections import Counter
from datetime import datetime
import json

from evidence_io import load, publish

ROOT = Path('docs/e2e/at/members')
IDS = ['AT05', 'AT01', 'AT02', 'AT03', 'AT04', 'AT08', 'AT20', 'AT21', 'AT22', 'AT09']


def stamp(value):
    return datetime.fromisoformat(value.replace('Z', '+00:00'))


def save(name, value):
    path = ROOT / name
    path.write_text(json.dumps(value, indent=2) + '\n')
    publish(path)


def main():
    cases = {case: load(ROOT / (case + '.json')) for case in IDS}
    run = cases['AT02']['run_id']
    source = cases['AT02']['source_commit']
    assert all(case['run_id'] == run and case['complete'] for case in cases.values())
    provenance = dict(run_id=run, source_commit=source)
    totals = Counter()
    rows = []
    attempts = []
    for name, case in cases.items():
        counts = Counter(step['status'] for step in case['steps'])
        totals.update(counts)
        rows.append(dict(case=name, status=case['status'], subchecks=dict(counts),
                         failures=[{k: s[k] for k in ['name', 'status', 'at', 'expected', 'command']} for s in case['steps'] if s['status'] == 'fail']))
        for step in case['steps']:
            if step['name'] == 'F14-01 original re-grant attempt retained':
                attempts.append(dict(case=name, original_step=step, evidence=json.loads(step['actual'])))
    save('F15-results.json', dict(provenance, cases=rows, subchecks=dict(totals),
                                 complete_cases=dict(Counter(c['status'] for c in cases.values()))))
    save('F15-D077-regrant-attempts.json', dict(provenance, attempts=attempts, cause_status='unproven',
         historical_F14_01='Failed original remains failed; no common cause inferred with F11-01.',
         historical_F11_01='Devstack 25 is a campaign, not a repair; first-selection success cannot close either finding.'))
    save('F15-F11-01-first-selection.json', dict(provenance, original_forwarding=cases['AT01'].get('forwarding', []), original_readiness=[s for s in cases['AT01']['steps'] if 'readiness' in s['name']], cause_status='unproven', historical_status='F11-01 and F14-01 remain unresolved; no cause or fix inferred.'))
    recovery = load(ROOT / 'F15-AT02-recovery.json')
    trace = load(ROOT / 'AT02-enforcement-timing.json')['records']
    policy = []
    for b in recovery['brackets']:
        if b['baseline_cause'] != 'policy_missing':
            continue
        a, z = int(b['baseline_sequence']), int(b['next_successful_apply_sequence'])
        window = [x for x in trace if x['run_id'] == b['producer_run_id'] and a-1 <= int(x['sequence']) <= z+2]
        policy.append(dict(original_bracket=b, producer_window=window))
    save('F15-AT02-policy-missing.json', dict(provenance, windows=policy, cause_status='unproven',
         correction='Ordinary stale lease: baseline, ACK, renew, Apply, verify, heartbeat. Verified deny-all only with a retained Apply error. No finite outage ceiling. D-139 option (b) is a design track only.',
         all_sibling_losses=recovery['lost_admitted_packets'], guard_losses=recovery['losses_in_completed_guards'],
         outside_guard_losses=recovery['losses_outside_completed_guards'],
         accounting='Every loss counted; overlapping baseline brackets are never summed as unique losses.'))
    # Central joins use exact durable IDs. Temporal adjacency is explicitly
    # supplementary; it cannot substitute for an incident/report/lease join.
    h = load(ROOT / 'F15-AT02-generation-holds.json')
    joined = h['joined']
    incidents = []
    for incident in joined['incidents']:
        samples = [s for s in joined['samples'] if s['hold_id'] == incident['id']]
        audits = [a for a in joined['escalations'] if a['evidence']['incident_id'] == incident['id']]
        checks = [c for c in h['checks'] if c['hold_id'] == incident['id']]
        lease_ids = {s['lease_id'] for s in samples}
        baselines = [dict(producer_run_id=b['producer_run_id'], sequence=b['baseline_sequence'],
                          lease_id=b['original_baseline_record'].get('lease', {}).get('lease_id'),
                          baseline_start_ns=b['baseline_start_ns'], recovery_end_ns=b['recovery_end_ns'],
                          sibling_coverage=b['sibling_coverage'], admitted_losses=b['admitted_losses'])
                     for b in recovery['brackets']
                     if b['original_baseline_record'].get('lease', {}).get('lease_id') in lease_ids]
        revocations_inside = [a for a in joined['revocations']
                              if stamp(a['occurred_at']) >= stamp(incident['first_seen_at'])
                              and (incident['ended_at'] is None or stamp(a['occurred_at']) <= stamp(incident['ended_at']))]
        latest = next((s for s in samples if s['report_id'] == incident['latest_report_id']), None)
        assert latest is not None
        assert len(audits) == (1 if incident['escalated_at'] else 0)
        assert all(c['valid'] for c in checks)
        incidents.append(dict(incident=incident, overlaps_AT02=(stamp(incident['first_seen_at']) <= stamp(h['snapshots'][-1]['at']) and (incident['ended_at'] is None or stamp(incident['ended_at']) >= stamp(cases['AT02']['at']))), report_ids=[s['report_id'] for s in samples],
                              lease_ids=sorted(lease_ids), escalation_audit_ids=[a['id'] for a in audits],
                              node_reported_alarm=any(s['reported_alarm'] is True for s in samples),
                              central_escalated=bool(audits), original_samples=samples, checks=checks,
                              same_lease_baseline_brackets=baselines,
                              revocations_during_central_incident=revocations_inside,
                              matching_lease_closure_is_not_service_recovery=True))
    # No scenario may be inferred from mere grant/device withdrawal. The
    # members table, not entitlements or a counter disappearing, defines the
    # owner criterion for members_revoked. Preserve all snapshots for review.
    terminal = [i for i in incidents if i['incident']['end_reason'] == 'members_revoked']
    sole = [i for i in terminal if len(i['original_samples'][-1]['leased_member_ids']) == 1]
    safety = [c for c in h['checks'] if c['retained_safety_failure']]
    save('F15-AT02-central-hold-join.json', dict(provenance, incidents=incidents,
         revocations=joined['revocations'], revocation_member_ID_joins=load(ROOT / 'F15-AT02-revocation-members.json'), mutation_snapshots=h['snapshots'],
         coverage=dict(sole_member_revocation='exercised' if sole else 'not exercised',
                       sole_member_incident_ids=[i['incident']['id'] for i in sole],
                       sibling_member_revocation='not exercised: classify from retained member-state snapshots; device/grant withdrawal is not member suspension or closure',
                       retained_safety_precedence='exercised' if safety else 'not exercised',
                       retained_safety_checks=safety),
         interpretation='Node-wide; no per-member hold field. Eligible siblings remain affected until matching-lease evidence. A members_revoked closure never claims recovered service.',
         rows_agree_with_node_wide_reading=all(not s['observation']['member_forwarding_enabled'] for s in joined['samples']),
         limits='Cumulative durable evidence through AT02; overlaps_AT02 separates earlier incidents. Same-lease baseline joins identify related producer records, not one-to-one delivery or causation. Every central ID and sample is retained; revocations outside the central receipt window remain in the full audit list.'))
    steps = cases['AT09']['steps']
    save('F15-AT09-restart.json', dict(provenance, original_steps=[s for s in steps if any(x in s['name'] for x in ['restart', 'offline', 'namespace', 'provisional'])]))
    save('F15-D137-native-paths.json', dict(provenance, original_rows=[s for s in steps if any(x in s['name'] for x in ['MTU', 'D-137', 'relay'])],
         relay_ordering='682d4434 routed-after-relay ordering executed in this run; row result retained independently.',
         historical_relay='Follow-up 14 FAIL remains recorded and confounded by its earlier routed hop.',
         receive_only='Limitation rows retain their own result; never upgraded to PASS.',
         historical_F8_02='Historical failures remain failed and unattributed.'))
    print(json.dumps(dict(run_id=run, cases=Counter(c['status'] for c in cases.values()), subchecks=totals,
                          regrant_attempts=len(attempts), AT02_hold_incidents=len(incidents))))


if __name__ == '__main__':
    main()
