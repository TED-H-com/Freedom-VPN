#!/usr/bin/env python3
"""Retain the new run and historical failure without changing any verdict."""
import datetime
import json

from followup11_records import protocol_cutoff
from followup16_records import verdict
from run import ROOT, PUBLIC, write_json

HISTORY = 'history/follow-up-17/'
NAMES = ('grant_expiry', 'lease_expiry', 'control_lost_agent_alive',
         'device_revoke', 'manual_grant_withdrawal', 'guest_clock_rollback')


def qualifying_outage(row):
    lease = row.get('signed_lease', {})
    try:
        issued = datetime.datetime.fromisoformat(lease['issued_at'].replace('Z','+00:00'))
        expires = datetime.datetime.fromisoformat(lease['expires_at'].replace('Z','+00:00'))
        cap_at = datetime.datetime.fromisoformat(row['lease_setup']['cap_changed_at'].replace('Z','+00:00'))
    except (KeyError, ValueError, TypeError):
        return False
    return bool(row.get('lease_setup_qualified') is True and row.get('control_stop_completed_at')
                and lease.get('lease_id') and lease.get('max_offline_seconds') == 60
                and issued > cap_at and 0 < (expires-issued).total_seconds() <= 60)


def main():
    def load(name):
        return json.loads((PUBLIC/name).read_text())
    run = load('followup17-transports-run.json')
    if not run.get('finished_at'):
        raise SystemExit('run has not finished cleanup')
    path = PUBLIC/'followup17-transports-live.json'
    if path.exists():
        path.write_text(path.read_text().replace(str(ROOT)+'/', ''))
        live = json.loads(path.read_text())
    else:
        live = {}
    cases = live.get('transport_cases',{})
    actual = {name: cases.get(name, dict(status='blocked',reason='run did not reach this sub-case')) for name in NAMES}
    for name in ('grant_expiry','lease_expiry','device_revoke'):
        if actual[name].get('cutoff_samples'):
            actual[name]['protocol_verdicts'] = {p:protocol_cutoff(actual[name],p) for p in ('TCP','UDP')}
    overlap = load('followup17-overlap.json')
    status = verdict(actual)
    if overlap['confounded'] and status == 'passed':
        status = 'blocked'
    row = actual['control_lost_agent_alive']
    answered = qualifying_outage(row)
    finding = dict(id='SECOPS-F16-01',status='answered' if answered else 'open',
                   historical_status='failed',historical_run_id='1bfd5cff-e593-481b-a4d2-94ca08cf7c3d',
                   historical_evidence='history/before-follow-up-17/followup16-finding.json',
                   new_run_id=live.get('run_id'), qualifying_outage_reached=answered,
                   design='cap before member admission',
                   product_finding=answered and row.get('status')=='failed',
                   evidence=HISTORY+'followup17-transports-live.json')
    write_json(PUBLIC/'followup17-finding.json',finding)
    record=dict(schema_version=1,id='AT29',follow_up=17,status=status,previous_status='failed',
                run_id=live.get('run_id'),driver_run_id=run['driver_run_id'],source_revision=run['source_revision'],application_base=run['application_base'],
                requirements=['ENT02','ISO02','SEC01'],spec=['§9.4','§16.3','§17.1','§16.5','§19 AT29'],
                actual=actual,secops_f16_01=finding,confounded=overlap['confounded'],
                previous_records='history/before-follow-up-17/',
                historical_failures=[dict(run_id='1bfd5cff-e593-481b-a4d2-94ca08cf7c3d',status='failed',evidence='history/before-follow-up-17/AT29.json')],
                record_kind='one fresh transport attempt; no failed-sub-case retry',
                evidence=[HISTORY+n for n in ('followup17-transports-live.json','followup17-transports-run.json','followup17-overlap.json')],
                blockers=['guest_clock_rollback needs an independently clocked disposable guest; w3-devstack-e2e / operator.']+
                         [name+': '+r.get('status','unmeasured') for name,r in actual.items() if name!='guest_clock_rollback' and r.get('status')!='passed'])
    write_json(PUBLIC/'AT29.json',record)
    table='\n'.join('| '+name+' | '+r.get('status','unmeasured')+' |' for name,r in actual.items())
    (PUBLIC/'AT29.md').write_text(f'''# AT29 — Follow-up 17

Overall **{status}**, previously **failed**. Run `{record['run_id']}` on
application base `{record['application_base']}`. Shared development host only.
SECOPS-F16-01 is **{finding['status']}**; qualifying outage reached: **{answered}**.
D-133 confounded window: **{overlap['confounded']}**. No failed sub-case rerun.

| Sub-case | Result |
|---|---|
{table}

The setup sets and reads back cap 60 before the existing finite grant and native
device admission. It accepts only the locally stored lease issued after that
write, with lifetime within 60s and the matching serving heartbeat. Timeout
names the observed lease. No database edits or product changes.

Every Follow-up 16 continuity predicate is retained: the original live agent,
one identity, original TCP/9446 and authenticated UDP/9445 sockets, at least
three Step intervals through D−101ms−1s, agent timeout log evidence, and quiet
by D+3s through the following ten seconds. The killed-agent case remains the
independent kernel/watchdog case; its three-second diagnostic is unchanged.
No D-140/D-142 packet-level isolation qualification or physical arm64 claim.

See [this run's complete record](AT29.json) and
[dated diagnosis and hand-off](2026-11-04-follow-up-17.md).
Run `1bfd5cff` remains **failed** in the
[byte-identical pre-run archive](history/before-follow-up-17/AT29.json).
Guest clock rollback remains blocked with **w3-devstack-e2e / operator**.
''')
    print('AT29:',status,'; SECOPS-F16-01:',finding['status'],'; cases:',{n:r.get('status') for n,r in actual.items()})


if __name__=='__main__':
    main()
