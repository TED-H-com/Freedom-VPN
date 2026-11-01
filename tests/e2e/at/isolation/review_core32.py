"""Read-only wire/stop review of agent-core 32 in one assembled run."""
import json
from evidence_io import read_json
from pathlib import Path
import re
from candidate_evidence import wall_ns


def hold_reports(transactions):
    leases={(w.get('response') or {}).get('signed',{}).get('lease_id'):(w.get('response') or {}).get('signed',{}) for w in transactions if 'lease' in w.get('path','')}
    reports={}
    for i,w in enumerate(transactions):
        b=w.get('body') or {}
        if not w.get('path','').endswith('/observations') or not b.get('report_id'): continue
        item=reports.setdefault(b['report_id'],dict(body=b,attempt_indices=[],statuses=[]))
        if item['body']!=b: raise ValueError('retried report body changed')
        item['attempt_indices'].append(i);item['statuses'].append(w['status'])
    holds=[]; other=[]
    pattern=r'Serving lease generation hold: applied=([0-9]+) lease=([0-9]+) steps=([0-9]+) elapsed_ms=([0-9]+) alarm=(true|false)\.'
    for rid,item in reports.items():
        b=item['body']; lease=leases.get((b.get('lease') or {}).get('lease_id'),{})
        applied=int(b.get('applied_config_generation') or 0); generation=int(lease.get('config_generation') or 0)
        errors=[e for e in b.get('errors') or [] if e.get('stage')=='lease' and e.get('code')=='STALE_GENERATION']
        mismatch=bool(generation and applied>generation and b.get('member_forwarding_enabled') is False)
        missing='policy_missing' in (b.get('forwarding_block_reasons') or [])
        if not errors and not mismatch:
            if missing: other.append(dict(report_id=rid,observation_at=b['observed_at'],reason='policy_missing alone does not establish an acknowledged stale-lease hold',attempt_indices=item['attempt_indices']))
            continue
        summaries=[]
        for e in errors:
            m=re.fullmatch(pattern,e['message'])
            if m:
                a,g,steps,elapsed=map(int,m.groups()[:4]);alarm=m[5]=='true'
                summaries.append(dict(applied=str(a),lease=str(g),steps=str(steps),elapsed_ms=str(elapsed),alarm=alarm,
                    threshold_reached=steps>=3 or elapsed>=30000,
                    summary_consistent=(a==applied and g==generation and a>g and alarm==(steps>=3 or elapsed>=30000))))
        holds.append(dict(report_id=rid,observed_at=b['observed_at'],attempt_indices=item['attempt_indices'],statuses=item['statuses'],
            accepted=202 in item['statuses'],applied_generation=str(applied),signed_lease_generation=str(generation),
            forwarding=b['member_forwarding_enabled'],reasons=b.get('forwarding_block_reasons'),lease_errors=errors,
            qualifies=missing and bool(errors) and b['member_forwarding_enabled'] is False,
            streak_summaries=summaries))
    alarms=[dict(report_id=h['report_id'],observed_at=h['observed_at'],accepted=h['accepted'],summary=s) for h in holds for s in h['streak_summaries'] if s['alarm']]
    return dict(unique_hold_reports=len(holds),all_hold_reports_qualified=bool(holds) and all(h['qualifies'] for h in holds),
        accepted_hold_reports=sum(h['accepted'] for h in holds),holds=holds,alarms=alarms,
        alarm_observed=bool(alarms),all_summary_thresholds_consistent=all(s['summary_consistent'] for h in holds for s in h['streak_summaries']),
        other_policy_missing_reports=other,
        interpretation='Every observed hold must carry policy_missing and STALE_GENERATION at stage=lease with forwarding false. Signed lease generations resolve the ACK comparison; baseline alone is not a hold classification. Alarm is observed only when the original summary says true; 3 Steps or 30 seconds is diagnostic, never a forwarding grace. Missing Step events prevent independently reconstructing every unreported increment.')


def review(root):
    read=lambda name:read_json(root/name)
    transactions=read('receiver-capture.json')['transactions']
    holds=hold_reports(transactions)
    stops=[]
    for row in read('receiver-closure.json'):
        try: sample=json.loads(row['output'])
        except (ValueError,KeyError): sample=dict(available=False)
        during=[]
        if sample.get('signal_ok') and sample.get('exit_observed'):
            lo=wall_ns(sample['signal_return']['wall']);hi=wall_ns(sample['exit_sample']['wall'])
            during=[dict(transaction_index=i,report_id=(w.get('body') or {}).get('report_id'),response_at=w['response_at'],status=w['status']) for i,w in enumerate(transactions) if w.get('path','').endswith('/observations') and lo<=wall_ns(w['response_at'])<=hi]
        stops.append(dict(container=row['container'],sample=sample,stored_lease_survived=row['stored_lease_survived_stop'],
            lease_id_before=row['before']['lease_id'],lease_id_after=row['after']['lease_id'],
            receiver_completions_during_signal_to_exit=during,no_control_io_after_cancel='not observable here: Step cancellation timestamp and agent-side POST completion are not exported; proxy completion alone cannot prove it'))
    result=dict(run_id=read('AT10.json')['run_id'],hold=holds,terminal_process_stops=stops,
        cooperative_bound_ns='15000000000',stored_lease_scope='Survival does not establish adoption or current serving eligibility.',
        container_lifecycle='container-lifecycle-events.json; distinct from directly signalled agent process exits',
        baseline_brackets='policy-missing-analysis.json: baseline to next successful Apply, every loss counted, later recovery required')
    (root/'agent-core32-regression.json').write_text(json.dumps(result,indent=2)+'\n')
    return result


if __name__=='__main__':
    import sys
    review(Path(sys.argv[1]))
