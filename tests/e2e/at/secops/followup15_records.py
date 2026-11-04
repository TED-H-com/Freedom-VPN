#!/usr/bin/env python3
"""Render this transport run only; preserve every other acceptance campaign."""
import hashlib
import json

from followup11_records import protocol_cutoff
from followup15_archive import ARCHIVE
from run import ROOT, PUBLIC, write_json


def load(path):
    return json.loads(path.read_text())


def verdict(cases):
    statuses = [cases.get(name, {}).get('status', 'blocked') for name in
                ('grant_expiry', 'lease_expiry', 'device_revoke', 'manual_grant_withdrawal', 'guest_clock_rollback')]
    return 'failed' if 'failed' in statuses else 'blocked' if 'blocked' in statuses else 'passed'


def main():
    run = load(PUBLIC / 'followup15-transports-run.json')
    if not run.get('finished_at'):
        raise SystemExit('run is not finished')
    live_path = PUBLIC / 'followup15-transports-live.json'
    if live_path.exists():
        # Shared helper evidence may name a private capture below the worktree.
        # Keep the relative capture path, never a developer home directory.
        live_path.write_text(live_path.read_text().replace(str(ROOT)+'/', ''))
        live = load(live_path)
    else:
        live = {}
    cases = live.get('transport_cases', {})
    actual = {name: cases.get(name, dict(status='blocked', reason='run did not reach this sub-case')) for name in
              ('grant_expiry', 'lease_expiry', 'device_revoke', 'manual_grant_withdrawal', 'guest_clock_rollback')}
    actual['guest_clock_rollback'] = dict(status='blocked', owner='w3-devstack-e2e / operator',
        missing_export='Disposable guest with independently adjustable wall clock; containers share the development host clock.')
    for name in ('grant_expiry', 'lease_expiry', 'device_revoke'):
        if actual[name].get('cutoff_samples'):
            actual[name]['protocol_verdicts'] = {p: protocol_cutoff(actual[name], p) for p in ('TCP', 'UDP')}
    common = dict(schema_version=1, follow_up=15, run_id=live.get('run_id'), driver_run_id=run['driver_run_id'],
                  application_base=run['application_base'], source_revision=run['source_revision'])
    blockers = ['Guest clock rollback needs an independently clocked disposable guest; owner w3-devstack-e2e / operator.']
    for name, row in actual.items():
        if name != 'guest_clock_rollback' and row.get('status') != 'passed':
            blockers.append(name+': '+row.get('status', 'unmeasured')+'; see this run’s exact row.')
    record = dict(common, id='AT29', status=verdict(actual), requirements=['ENT02', 'ISO02', 'SEC01'],
                  spec=['§9.4', '§16.3', '§16.5', '§19 AT29'], actual=actual, blockers=blockers,
                  evidence=['followup15-transports-live.json', 'followup15-transports-run.json'],
                  previous_records='history/before-follow-up-15/', record_kind='one fresh transport run; no failed-sub-case retry')
    write_json(PUBLIC / 'AT29.json', record)
    table = '\n'.join('| '+name+' | '+row.get('status', 'unmeasured')+' |' for name, row in actual.items())
    (PUBLIC / 'AT29.md').write_text(f'''# AT29 — Follow-up 15

Overall **{record['status']}**, previously **blocked**. Run `{common['run_id']}`;
application base `{common['application_base']}`. Shared development host only.

| Sub-case | Result |
|---|---|
{table}

Manual withdrawal uses contract revision 14 `revokeMemberGrant` (#297).
The public 202 receipt, same-key replay, completed operation, live original
TCP/9446 and authenticated UDP/9445 senders, per-transport stop observations,
kernel timeout sets, monotonic elapsed time/deadline and recovery ACK are in
[the exact record](AT29.json). An accepted operation alone is not a pass.

The online bound is the §16.5 recommended 60-second starting target, measured
conservatively from request start. The earlier actual observed lease/member
deadline wins (§9.4, §16.3, §19 AT29). The 90-second setup, 120-second recovery,
one-second sampling, four-second positive window and ten-second quiet window
come from the existing transport harness; they are observation limits, not
production guarantees. Recovery requires a valid finite grant with the original
end, explicit administrator device authorization under D-077, operation/device
read-back, a new applied policy ACK and both fresh recovery transports.

The independent kernel/watchdog evidence is retained with the Agent killed.
Native route withdrawal and node policy enforcement remain jointly observed;
this run does not isolate their separate causal contribution.

Guest clock rollback remains blocked with **w3-devstack-e2e / operator**.
Containers share the host clock. Earlier runs, including `cf81f6f8`, remain
[byte-identical in the pre-run archive](history/before-follow-up-15/archive-index.json).
''')
    # AT31 is changed only when this same test actually emits its relay row.
    if 'forced_relay' in cases:
        old = load(ARCHIVE / 'AT31.json')
        old.update(common)
        old['actual'] = dict(forced_relay=cases['forced_relay'])
        old['record_kind'] = 'bounded relay row produced again by the AT29 transport test; no capacity campaign'
        old['evidence'] = ['followup15-transports-live.json', 'followup15-transports-run.json', 'history/before-follow-up-15/AT31.json']
        old.pop('current_restore_run_id', None)
        write_json(PUBLIC / 'AT31.json', old)
        (PUBLIC / 'AT31.md').write_text(f'''# AT31 — bounded relay row in Follow-up 15

Overall **blocked**. Run `{common['run_id']}` produced the one-member relay row
again: **{cases['forced_relay']['status']}**. See [the exact record](AT31.json).
Candidate hardware and the 20-member/60-device mixed relay campaign remain
operator/acceptance work. No capacity or candidate-hardware claim is added.
The previous row remains in [the archive](history/before-follow-up-15/AT31.json).
''')
    # Historical secops pins that matched the old root bytes now target the
    # byte-identical archive. Never rewrite an unmatched historical digest.
    scope = load(PUBLIC / 'scope-review.json')
    def redirect(value):
        if isinstance(value, dict):
            if 'path' in value and 'sha256' in value:
                from pathlib import Path
                path = Path(value['path'])
                archived = ARCHIVE / path.name
                if path.parent == PUBLIC.relative_to(ROOT) and archived.exists():
                    if hashlib.sha256(archived.read_bytes()).hexdigest() == value['sha256']:
                        value['path'] = str(archived.relative_to(ROOT))
            for child in value.values():
                redirect(child)
        elif isinstance(value, list):
            for child in value:
                redirect(child)
    redirect(scope)
    scope['follow_up_15'] = dict(scope='AT29 transports and shared node-write adoption; earlier campaigns retained',
        records=[dict(path=str(p.relative_to(ROOT)), sha256=hashlib.sha256(p.read_bytes()).hexdigest()) for p in
                 [PUBLIC/'AT29.json', PUBLIC/'AT29.md', PUBLIC/'AT31.json', PUBLIC/'AT31.md',
                  PUBLIC/'followup15-transports-run.json'] + ([live_path] if live_path.exists() else [])])
    write_json(PUBLIC / 'scope-review.json', scope)
    print('AT29:', record['status'], '; sub-cases:', {k: v.get('status') for k, v in actual.items()})


if __name__ == '__main__':
    main()
