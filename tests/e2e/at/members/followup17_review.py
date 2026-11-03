"""Read-only F17 joins. Every send remains in the denominator; no cause inferred."""
from pathlib import Path
from collections import Counter
from datetime import datetime
import hashlib
import json
import sys

from evidence_io import load, publish

ROOT = Path('docs/e2e/at/members')


def save(name, value):
    path = ROOT / name
    path.write_text(json.dumps(value, indent=2) + '\n')
    publish(path)


def ns(value):
    whole, _, fraction = value.replace('+00:00', 'Z').removesuffix('Z').partition('.')
    # UTC RFC3339 inputs, integer seconds: never float a sequence or ns clock.
    epoch = datetime.fromisoformat(whole + '+00:00') - datetime.fromisoformat('1970-01-01T00:00:00+00:00')
    return (epoch.days * 86400 + epoch.seconds) * 10**9 + int((fraction+'000000000')[:9])


def ident(row):
    return {k: row[k] for k in ['boot_id', 'run_id', 'sequence']}


def preapply(row):
    t = row['timing']
    before = row.get('host', {}).get('before_apply', {})
    commands = [c for c in (t.get('commands') or []) if c['phase'] == 'revoke']
    out = dict(available=bool(before.get('ok')), queued_delete_proven=bool(commands), revoke_commands=commands)
    if out['available']:
        a, b = int(before['end_ns']), int(t['start_ns'])
        assert 0 < a <= b
        out.update(start_ns=str(a), end_ns=str(b), duration_ns=str(b-a))
    return out


def since_wall(row, mono, wall):
    anchor = row['anchor']
    delta = ns(anchor['wall']) - ns(wall) + int(mono)
    return dict(lower_ns=str(delta-int(anchor['mono_after_ns'])), upper_ns=str(delta-int(anchor['mono_before_ns'])))


def streams():
    result, pins = {}, {}
    for role, name in [('node', 'f17-parser-producer.jsonl'), ('node2', 'f17-node2-producer.jsonl')]:
        path = Path('.agent/tmp') / name
        raw = path.read_bytes()
        rows = [json.loads(line) for line in raw.splitlines()]
        last = {}
        for r in rows:
            key = r['boot_id'], r['run_id']
            assert int(r['sequence']) == last.get(key, 0)+1, 'producer sequence gap'
            last[key] = int(r['sequence'])
        result[role] = rows
        pins[role] = dict(bytes=str(len(raw)), sha256=hashlib.sha256(raw).hexdigest(), records=len(rows))
    return result, pins


def send_join(cases, recovery, holds):
    step = next(s for s in cases['AT02']['steps'] if s['name']=='D-135 continuous admitted sibling capture')
    value = json.loads(step['actual'])
    sender, observation = value['sender'], value['node']
    sends = sender['sends']
    assert len(sends) == sender['attempts'] and sender['boot_id'] == observation['boot_id']
    assert [int(x['sequence']) for x in sends] == list(range(sender['attempts']))
    ingress, egress = {}, {}
    for p in observation['packets']:
        (ingress if p['interface']=='fvmb0' else egress).setdefault(p['sequence'], []).append(p['mono_ns'])
    trace = load(ROOT/'AT02-enforcement-timing.json')['records']
    applies = [r for r in trace if r.get('timing', {}).get('operation')=='apply' and r['boot_id']==sender['boot_id']]
    rows = []
    for a in sends:
        lo, hi = int(a['mono_before_ns']), int(a['mono_after_ns'])
        assert lo <= hi
        seq = a['sequence']
        guards = []
        reservations = []
        for r in applies:
            d = r['timing']['deadline']
            if d['guard_released'] and lo <= int(d['release_end_ns']) and hi >= int(d['install_start_ns']):
                guards.append(ident(r))
            b = preapply(r)
            if b['available'] and lo <= int(b['end_ns']) and hi >= int(b['start_ns']):
                reservations.append(dict(ident(r), **b))
        brackets = [dict(producer_run_id=b['producer_run_id'], baseline_sequence=b['baseline_sequence'], recovery_sequence=b['next_successful_apply_sequence']) for b in recovery['brackets'] if b['sibling_coverage']!='outside continuous sibling window' and lo <= int(b['recovery_end_ns'] or observation['end_ns']) and hi >= int(b['baseline_start_ns'])]
        rows.append(dict(send=a, node_ingress_ns=ingress.get(seq, []), node_egress_ns=egress.get(seq, []), lost=seq not in egress, missing_ingress=seq not in ingress, send_overlaps_completed_guards=guards, send_overlaps_baseline_recovery=brackets, send_overlaps_pre_apply=reservations))
    losses = [r for r in rows if r['lost']]
    missing = [r for r in rows if r['missing_ingress']]
    assert len(losses)==recovery['lost_admitted_packets']
    assert len(missing)==recovery['not_observed_at_node_ingress']
    gaps=[]
    ordered=sorted((int(times[0]),seq) for seq,times in ingress.items())
    for (a,s),(b,t) in zip(ordered,ordered[1:]):
        if b-a>100_000_000:
            between=[r for r in rows if int(r['send']['mono_before_ns'])>=a and int(r['send']['mono_after_ns'])<=b]
            gaps.append(dict(previous_sequence=s,next_sequence=t,start_ns=str(a),end_ns=str(b),duration_ns=str(b-a),sends=[r['send']['sequence'] for r in between],losses=[r['send']['sequence'] for r in between if r['lost']]))
    save('F17-AT02-every-send.json', dict(run_id=cases['AT02']['run_id'],source_commit=cases['AT02']['source_commit'],clock=sender['clock'],boot_id=sender['boot_id'],attempts=len(rows),send_errors=sender['send_errors'],lost=len(losses),missing_ingress=len(missing),rows=rows,ingress_gaps_over_100ms=gaps,original_node_capture_statistics={k:observation[k] for k in ['capture_stats','stats_available','read_errors']},original_revocation_audits=holds['joined']['revocations'],mutation_snapshots=holds['snapshots'],scope='Every original write, including errors, and every missing egress retained. Send overlap locates attempts, not the cause of a drop. Node-ingress guard accounting remains independently in the original provisional D-135/D-136 result. No fixed outage ceiling; no historic verdict changed.',cause_status='unproven'))
    return rows


def reservation_review(cases, recovery, holds, data, pins):
    inventory=[]
    for role, records in data.items():
        for row in records:
            t=row.get('timing',{})
            if t.get('operation')!='apply':continue
            b=preapply(row)
            inventory.append(dict(node_role=role, **ident(row), original_record=row, pre_apply=b))
    windows=[]
    for b in recovery['brackets']:
        r=b.get('original_recovery_record')
        if not r:continue
        p=preapply(r)
        windows.append(dict(producer_run_id=b['producer_run_id'],baseline_sequence=b['baseline_sequence'],baseline_start_ns=b['baseline_start_ns'],recovery_sequence=b['next_successful_apply_sequence'],recovery_end_ns=b['recovery_end_ns'],baseline_to_recovery_ns=b['baseline_to_recovery_ns'],pre_apply=p,entire_reservation_inside_baseline_outage=(p['available'] and int(b['baseline_start_ns'])<=int(p['start_ns'])<=int(p['end_ns'])<=int(b['recovery_end_ns'])),losses=b['admitted_losses']))
    before=next(s for s in holds['snapshots'] if s['label']=='before AT02 device revoke with sibling')
    after=next(s for s in holds['snapshots'] if s['label']=='after AT02 device revoke with sibling')
    revokes=[]
    for i in inventory:
        row=i['original_record'];t=row['timing']
        if i['node_role']=='node' and i['pre_apply']['queued_delete_proven'] and ns(before['at'])<=ns(row['anchor']['wall'])<=ns(after['at'])+120*10**9:
            revokes.append(dict(identity=ident(row),pre_apply=i['pre_apply'],before_command_to_install_end=since_wall(row,t['deadline']['installed_by_ns'],before['at']) if int(t['deadline']['installed_by_ns']) else None,after_command_to_install_end=since_wall(row,t['deadline']['installed_by_ns'],after['at']) if int(t['deadline']['installed_by_ns']) else None))
    suspensions=[]
    for step in cases['AT04']['steps']:
        if not step['name'].startswith('AT04 monitor producer stream after member-suspend'):continue
        value=json.loads(step['actual']);role=value['after']['node'];at=value['event_at']
        selected=[i for i in inventory if i['node_role']==role and ns(at)<=ns(i['original_record']['anchor']['wall'])<=ns(value['last_sample'])]
        suspensions.append(dict(original_monitor=step,node_role=role,event_at=at,applies=[dict(identity=ident(i['original_record']),pre_apply=i['pre_apply'],input_to_apply_end=since_wall(i['original_record'],i['original_record']['timing']['end_ns'],at),input_to_restricting_install_end=since_wall(i['original_record'],i['original_record']['timing']['deadline']['installed_by_ns'],at) if int(i['original_record']['timing']['deadline']['installed_by_ns']) else None) for i in selected]))
    save('F17-pre-apply-reservations.json',dict(run_id=cases['AT02']['run_id'],source_commit=cases['AT02']['source_commit'],bracket_read='host.before_apply.end_ns -> timing.start_ns',producer_exports=pins,applies=inventory,sibling_baseline_recovery_windows=windows,AT02_revocation_command_bounds=dict(before=before['at'],after=after['at']),AT02_revocation_batches=revokes,AT04_suspension=suspensions,scope='The pre-Apply bracket includes the durable reservation and common setup; not isolated database transaction latency. Nonempty batches are proved by original phase=revoke commands. Absence of a command cannot exclude a queued batch on an early failed Apply. Request-to-install and baseline-to-recovery intervals include this bracket without adding it twice. A baseline may already close revoked forwarding before the restricting Apply. Applied ACK refs authorize no full-h or partial-service classification.'))


def hold_review(cases):
    rows=[]
    central_joins=load(ROOT/'F17-AT02-generation-holds.json')['joined']
    supplemental=[] # member.patch is retained by the live query
    central_joins['revocations']=list({a['id']:a for a in central_joins['revocations']+supplemental}.values())
    for name in ['alarm-recovery','sibling-removal','sole-member','retained-safety']:
        v=load(ROOT/('F17-hold-'+name+'.json'))
        central=v.get('central') or {}
        incident=next((h for h in central.get('incidents',[]) if h['id']==v.get('incident_id')),None)
        samples=[s for s in central.get('samples',[]) if incident and s['hold_id']==incident['id']]
        report_ids={s['report_id'] for s in samples}
        reports=[o for o in central.get('reports',[]) if o['report_id'] in report_ids]
        audits=[a for a in central.get('audits',[]) if incident and a['evidence']['incident_id']==incident['id']]
        failed=[]
        if incident:
            lo=ns(incident['first_seen_at']);hi=ns(incident['ended_at']) if incident['ended_at'] else ns(v['finished_at'])
            failed=[r for r in v['producer_records'] if r.get('timing',{}).get('operation')=='apply' and not r['timing']['succeeded'] and lo<=ns(r['anchor']['wall'])<=hi]
        safety=[]
        for o in reports:
            retained=[c for c in o.get('checks') or [] if c['name'] in ('qos','source_binding') and c['status']=='failed' and c.get('error_code') not in (None,'STALE_GENERATION','VALIDATION_FAILED')]
            if retained:safety.append(dict(report_id=o['report_id'],checks=retained))
        alarm_samples=[s for s in samples if s['summary_valid'] and s['reported_alarm']]
        trigger='not exercised'
        if audits and incident:
            age=ns(audits[0]['occurred_at'])-ns(incident['first_seen_at'])
            trigger='reported alarm before 30-second central fallback' if age<30*10**9 and alarm_samples else 'central age or reported alarm; inspect original audit'
        leases=[x for x in central.get('leases',[]) if x['lease_id'] in {s['lease_id'] for s in samples}]
        row=dict(row=name,node_id=v['node_id'],incarnation_id=v['incarnation_id'],arm_id=v.get('arm',{}).get('id'),incident=incident,report_ids=sorted(report_ids),reports=reports,original_samples=samples,leases=leases,audit_ids=[a['id'] for a in audits],audits=audits,first_seen_at=incident['first_seen_at'] if incident else None,last_seen_at=incident['last_seen_at'] if incident else None,alarm_trigger=trigger,alarm_coverage='exercised' if alarm_samples and len(audits)==1 else 'not exercised',failed_apply_coverage='exercised' if failed else 'not exercised',failed_apply_records=failed,retained_safety_coverage='exercised' if safety else 'not exercised',retained_safety_checks=safety,member_revocation=v.get('member_revocation'),survivor_hold_observed=v.get('survivor_hold_observed'),members_revoked_closure=v.get('members_revoked_closure'),proxy=v.get('proxy'),forwarded_body_matches_local=v.get('forwarded_body_matches_local'),service_recovery_claimed=False)
        if incident and name=='sole-member' and v.get('members_revoked_closure'):
            assert all(len(x['members'])==1 for x in leases)
            assert {x['members'][0]['member_id'] for x in leases}=={v['member_revocation']['member_id']}
        mutation=v.get('member_revocation')
        row['member_revocation_audits']=[a for a in central_joins['revocations'] if mutation and a['resource']=='member:'+mutation['member_id'] and ns(mutation['started_at'])<=ns(a['occurred_at'])<=ns(mutation['completed_at'])]
        row['publication_classification_replays']=[c for c in load(ROOT/'F17-AT02-generation-holds.json')['checks'] if incident and c['hold_id']==incident['id']]
        rows.append(row)
    save('F17-populated-hold-joins.json',dict(run_id=cases['AT02']['run_id'],source_commit=cases['AT02']['source_commit'],rows=rows,scope='Join accepted samples by node/incarnation/report and incident_id; join audits by exact incident_id, never node alone. Each scenario has a fresh transport arm and its original lease population. No incident ending is a service-recovery assertion. Missing failed-Apply or safety combinations remain not exercised.'))


def main():
    cases={case:load(ROOT/(case+'.json')) for case in ['AT02','AT04']}
    recovery=load(ROOT/'F17-AT02-recovery.json')
    holds=load(ROOT/'F17-AT02-generation-holds.json')
    data,pins=streams()
    send_join(cases,recovery,holds)
    reservation_review(cases,recovery,holds,data,pins)
    hold_review(cases)
    print('Every-send, reservation and populated-hold ID joins written')

if __name__=='__main__':main()
