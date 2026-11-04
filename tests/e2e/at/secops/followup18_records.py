#!/usr/bin/env python3
"""Offline D-160 and fault-time lease audit; retain every failed run."""
import datetime
import json

from followup11_records import protocol_cutoff
from followup16_records import verdict
from followup17_records import NAMES, qualifying_outage
from run import ROOT, PUBLIC, write_json
from timing_admission import validate

HISTORY = 'history/follow-up-18/'


def fault_qualification(row):
    lease = row.get('lease_at_fault', {})
    heartbeat = row.get('heartbeat_at_fault', {})
    read = row.get('fault_lease_read', {})
    try:
        issued = datetime.datetime.fromisoformat(lease['issued_at'].replace('Z','+00:00'))
        expires = datetime.datetime.fromisoformat(lease['expires_at'].replace('Z','+00:00'))
        duration = (expires-issued).total_seconds()
        d, now, sent = (int(lease['deadline_monotonic_ns']), int(read['mono_ns']), int(lease['request_monotonic_ns']))
        valid = bool(row.get('fault_lease_qualified') and read['locally_accepted'] and lease['lease_id']
                     and lease['max_offline_seconds']==60 and 0 < duration <= 60
                     and abs((d-sent)/1e9-duration) < 0.000001
                     and 0 < sent <= now < d and heartbeat.get('serving')=='1'
                     and heartbeat.get('policy_expires_mono_ns')==str(d))
    except (KeyError, TypeError, ValueError):
        valid = False
    return valid


def audit(live, events):
    timing = validate(events)
    errors = list(timing['errors'])
    sections = {s['section']: s for s in timing['sections']}
    for name, row in live.get('transport_cases', {}).items():
        if 'timed_section' not in row:
            if row.get('status')=='passed':
                errors.append('passed case missing timed section: '+name)
            continue
        section = sections.get(name)
        declared = row['timed_section']
        if not section or declared.get('start') != section['start'] or declared.get('end') != section['end']:
            errors.append('case/journal section mismatch: '+name)
    row = live.get('transport_cases',{}).get('lease_expiry',{})
    qualified = fault_qualification(row)
    if row.get('cutoff_samples') and not qualified:
        errors.append('lease_expiry cutoff without qualified fault lease')
    return dict(passed=not errors, errors=errors, timing=timing, fault_lease_qualified=qualified,
                cutoff_reached=qualified and len(row.get('cutoff_samples',[]))==2,
                answered=not errors and qualified and len(row.get('cutoff_samples',[]))==2)


def boundaries(row):
    samples = row.get('samples', []) + row.get('cutoff_samples', [])
    if not samples:
        return {}
    d = row['deadline_elapsed_seconds']
    result = {}
    for protocol, key in [('TCP','tcp_echoes'),('UDP','udp_replies')]:
        previous, last, first = None, None, None
        for sample in samples:
            n = sample['status'][key]
            elapsed = sample['monotonic_elapsed_seconds']
            if previous is not None:
                if n > previous and elapsed < d:
                    last = dict(utc=sample['utc'], elapsed_seconds=elapsed, relative_to_D_seconds=elapsed-d, replies=n)
                if n == previous and elapsed > d and first is None:
                    first = dict(utc=sample['utc'], elapsed_seconds=elapsed, relative_to_D_seconds=elapsed-d, replies=n)
            previous = n
        result[protocol] = dict(last_reply_before_D=last, first_quiet_after_D=first,
                                limitation='Sample observations, not packet timestamps')
    return result


def main():
    def load(name): return json.loads((PUBLIC/name).read_text())
    run = load('followup18-transports-run.json')
    if not run.get('finished_at'): raise SystemExit('run has not finished cleanup')
    path = PUBLIC/'followup18-transports-live.json'
    live = json.loads(path.read_text().replace(str(ROOT)+'/', '')) if path.exists() else {}
    if live: write_json(path,live)
    admission = load('followup18-admission.json')
    checked = audit(live,admission['events'])
    write_json(PUBLIC/'followup18-audit.json',checked)
    actual = {name:live.get('transport_cases',{}).get(name,dict(status='blocked',reason='not reached')) for name in NAMES}
    for name in ('grant_expiry','lease_expiry','device_revoke'):
        if actual[name].get('cutoff_samples'):
            actual[name]['protocol_verdicts']={p:protocol_cutoff(actual[name],p) for p in ('TCP','UDP')}
    overlap = load('followup18-overlap.json')
    status = verdict(actual)
    if not checked['passed']: status='failed'
    if overlap['confounded'] and status=='passed': status='blocked'
    finding=dict(id='SECOPS-F17-01',status='answered' if checked['answered'] else 'open',
                 run_id=live.get('run_id'),cutoff_reached=checked['cutoff_reached'],
                 evidence=HISTORY+'followup18-audit.json',
                 historical_run_id='b2de46c8-7611-4b7c-8863-2345ff2b0240',historical_status='failed',
                 product_finding=checked['answered'] and actual['lease_expiry'].get('status')=='failed')
    write_json(PUBLIC/'followup18-finding.json',finding)
    record=dict(schema_version=1,id='AT29',follow_up=18,status=status,previous_status='failed',
                run_id=live.get('run_id'),driver_run_id=run['driver_run_id'],source_revision=run['source_revision'],application_base=run['application_base'],
                requirements=['ENT02','ISO02','SEC01'],spec=['§9.4','§16.3','§17.1','§16.5','§19 AT29'],
                actual=actual,secops_f17_01=finding,secops_f16_01_qualifying_outage=qualifying_outage(actual['control_lost_agent_alive']),
                confounded=overlap['confounded'],previous_records='history/before-follow-up-18/',
                historical_failures=[dict(run_id=r,status='failed',evidence=p) for r,p in [
                    ('b2de46c8-7611-4b7c-8863-2345ff2b0240','history/before-follow-up-18/AT29.json'),
                    ('1bfd5cff-e593-481b-a4d2-94ca08cf7c3d','history/before-follow-up-17/AT29.json')]],
                evidence=[HISTORY+n for n in ('followup18-transports-live.json','followup18-transports-run.json','followup18-admission.json','followup18-audit.json','followup18-overlap.json')],
                blockers=['guest_clock_rollback needs an independently clocked disposable guest; w3-devstack-e2e / operator.'])
    write_json(PUBLIC/'AT29.json',record)
    table='\n'.join('| '+name+' | '+row.get('status','unmeasured')+' |' for name,row in actual.items())
    (PUBLIC/'AT29.md').write_text(f'''# AT29 — Follow-up 18

Overall **{status}**, previously **failed**. Run `{record['run_id']}` on
application base `{record['application_base']}`. D-133 confounded: **{overlap['confounded']}**.
SECOPS-F17-01 is **{finding['status']}**. No failed sub-case retry.

| Sub-case | Result |
|---|---|
{table}

The old lease-expiry case reads the accepted local lease and heartbeat after
control stops, requires cap60s and a matching future deadline, then retains
the original three-second diagnostic, D+3s sampling and ten-second quiet test.
Every timed section is preceded by admission, with committed harness receipts.
The offline evidence audit rejects any admission check inside a section.
Follow-up17's control-loss qualification and all transport predicates remain.

[Complete record](AT29.json), [dated diagnosis and results](2026-11-04-follow-up-18.md).
Run `b2de46c8` remains **failed** in the [pre-run archive](history/before-follow-up-18/AT29.json).
Guest clock rollback remains blocked; no physical arm64 or full AT31 claim.
''')
    print('AT29:',status,'; SECOPS-F17-01:',finding['status'],'; audit:',checked['passed'])


if __name__=='__main__': main()
