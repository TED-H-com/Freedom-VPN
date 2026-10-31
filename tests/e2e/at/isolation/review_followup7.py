#!/usr/bin/env python3
"""Read-only derivation from one completed run; never changes AT outcomes."""
import json
import sys
from datetime import datetime
from pathlib import Path

root = Path(sys.argv[1] if len(sys.argv) > 1 else 'docs/e2e/at/isolation')
def read(name):
    return json.loads((root / name).read_text())
def time(s):
    return datetime.fromisoformat(s.replace('Z', '+00:00'))
def ms(a, b):
    return round((time(a) - time(b)).total_seconds() * 1000, 3)
def write(name, value):
    (root / name).write_text(json.dumps(value, indent=2) + '\n')

receiver = read('receiver-capture.json')
transactions = receiver.get('transactions', [])
observations = [r for r in transactions if r['path'].endswith('/observations')]
leases = {r['response']['signed']['lease_id']: r['response']['signed'] for r in transactions
          if r['path'].endswith('/lease-renewals') and 'signed' in r.get('response', {})}
desired = {r['response']['signed']['config_generation']: r['response']['signed'] for r in transactions
           if r['path'].endswith('/desired-state') and 'signed' in r.get('response', {})}
accepted = {r['body']['report_id'] for r in observations if 200 <= r['status'] < 300}
history = read('lease-history.json')
central_ids = set(history.get('accepted_report_ids') or [])
candidates = []
for r in observations:
    b = r['body']
    lease = leases.get((b.get('lease') or {}).get('lease_id'), {})
    generation = b.get('applied_config_generation')
    d = desired.get(generation, {})
    checks = {c['name']: c.get('error_code') for c in b.get('checks', [])}
    errors = b.get('errors') or []
    coherent = (200 <= r['status'] < 300 and lease and not lease.get('members')
                and d.get('device_bindings') and generation == lease.get('config_generation')
                and time(lease['issued_at']) <= time(b['observed_at']) < time(lease['expires_at']))
    shape = (b.get('provisioning_state') == 'ready' and b.get('safety_status') == 'passed'
             and b.get('member_forwarding_enabled') is False
             and b.get('forwarding_block_reasons') == ['awaiting_member_grants']
             and all(checks.get(k) == 'AWAITING_MEMBER_GRANTS' for k in ['lease', 'qos', 'source_binding'])
             and len(errors) == 1 and errors[0].get('stage') == 'lease'
             and errors[0].get('code') == 'AWAITING_MEMBER_GRANTS')
    if coherent and shape:
        populated = sorted((l for l in leases.values() if l.get('members')
                            and l.get('config_generation') == generation
                            and time(l['issued_at']) > time(b['observed_at'])), key=lambda l: l['issued_at'])
        candidates.append({'report_id': b['report_id'], 'observed_at': b['observed_at'],
                           'received_at': r['received_at'], 'lease_id': lease['lease_id'],
                           'generation': generation, 'desired_bindings': len(d['device_bindings']),
                           'populated_lease_id': populated[0]['lease_id'] if populated else None,
                           'populated_at': populated[0]['issued_at'] if populated else None,
                           'ready_to_populated_ms': ms(populated[0]['issued_at'], b['observed_at']) if populated else None,
                           'report': b})
write('provisional-receiver-analysis.json', {
    'run_id': receiver.get('run_id'), 'status': 'pass' if any(c['populated_at'] for c in candidates) else 'blocked',
    'scope': 'Actual emitted/accepted provisional reports, correlated with signed desired bindings and valid empty lease, then same-generation populated issuance. Counted admission is separately required by AT10.',
    'drain_ms': receiver.get('drain_ms'), 'observation_posts': len(observations),
    'distinct_accepted_report_ids': len(accepted), 'central_accepted_report_ids': len(central_ids),
    'central_ids_missing_from_receiver': sorted(central_ids - accepted),
    'desired_generations': sorted(desired, key=int), 'qualifying_reports': candidates,
    'comparison_ms': {'follow_up_6_assisted': 4767, 'usage_follow_up_3': [5021, 5025]}})

if (root / 'whole-node-probes.json').exists():
    data = read('whole-node-probes.json'); deadline = data['signed_deadline']
    probes = data['probes']; successes = [p for p in probes if p['result'] == 'connected']
    failures = [p for p in probes if p['result'] != 'connected']
    for p in probes:
        p['start_minus_deadline_ms'] = ms(p['start'], deadline)
        p['end_minus_deadline_ms'] = ms(p['end'], deadline)
    first_fail = failures[0] if failures else None
    last_success = successes[-1] if successes else None
    readbacks = read('whole-node-continuous-reads.json')['reads']
    baseline = [r for r in readbacks if 'fvpn_baseline=1' in r['readback']]
    expired = [r for r in readbacks if 'reason lease_expired' in r['readback'] or 'reason member_grants_expired' in r['readback']]
    actual = next(c for c in read('AT13.json')['checks'] if c['name'] == 'short whole-node lease expiry')
    write('expiry-timing-analysis.json', {'run_id': receiver.get('run_id'), 'status': actual['status'],
          'case_definition': 'Unchanged: near-deadline counted forward, no early baseline, baseline within one configured tick, post-deadline ingress denied with flat counters.',
          'signed_deadline': deadline, 'last_successful_connect': last_success, 'first_failed_connect': first_fail,
          'all_probes': probes, 'first_baseline_bracket': baseline[0] if baseline else None,
          'first_expiry_heartbeat_bracket': expired[0] if expired else None,
          'successes_after_deadline': [p for p in successes if time(p['end']) >= time(deadline)],
          'interpretation_limit': 'A connect timeout brackets a missing response; captures distinguish ingress from egress. Conservative monotonic expiry is permitted by §16.3; the strict case result is retained without redefining its predicate.'})

frozen = [r for r in observations if r.get('response', {}).get('identity_frozen') is True]
if frozen:
    first = frozen[0]
    following = [r for r in observations if time(r['received_at']) >= time(first['response_at'])]
    renewals = [r for r in transactions if r['path'].endswith('/lease-renewals') and time(r['received_at']) >= time(first['response_at'])]
    write('freeze-receiver-analysis.json', {'run_id': receiver.get('run_id'), 'first_frozen_acceptance': first,
          'first_later_observation': following[0] if following else None,
          'first_later_renewal': renewals[0] if renewals else None,
          'renewal_delay_ms': ms(renewals[0]['received_at'], first['response_at']) if renewals else None,
          'later_forwarding_reports': [r['body']['report_id'] for r in following if r['body'].get('member_forwarding_enabled')],
          'later_generation_acks': [r['body']['report_id'] for r in following if r['body'].get('applied_config_generation') is not None]})
print(json.dumps({'run_id': receiver.get('run_id'), 'observation_posts': len(observations), 'qualifying_provisional_reports': len(candidates), 'frozen_acceptances': len(frozen)}))
