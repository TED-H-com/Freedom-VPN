"""D-144/D-163 scheduled exit mapping; original records and exits stay intact."""
import json
from pathlib import Path


HOLD_REVIEW_PHASE = 'python3 -B tests/e2e/at/isolation/review_populated_holds.py'
DRIVER_PHASE = 'python3 tests/e2e/at/isolation/driver.py run'
HOLD_ITEMS = frozenset((
    'populated_hold', 'independent_denial', 'alarm_single_escalation',
    'unrelated_failure_precedence', 'active_stop_retention', 'matching_recovery',
    'distinct_later_incident', 'pure_stale_generation', 'central_30s_fallback'))


def review_mapping(root, phase, run_id):
    """Only the named review's complete same-run result can explain exit 1."""
    if phase['name'] != HOLD_REVIEW_PHASE:
        return None
    entries = json.loads(Path(__file__).with_name('expected-not-exercised.json').read_text())
    entries = [e for e in entries if e.get('review') == 'populated-holds']
    allowed = {e['item']: e for e in entries}
    valid_entries = len(allowed) == len(entries) and all(
        all(e.get(k) for k in ('item', 'reason', 'owner', 'established_by')) for e in entries)
    try:
        record = json.loads((root/'populated-holds-review.json').read_text())
    except (OSError, ValueError):
        return None
    if not isinstance(record, dict):
        return None
    coverage = record.get('coverage', {})
    if not isinstance(coverage, dict):
        return None
    missing = {k for k, v in coverage.items() if v == 'not exercised'}
    valid = (valid_entries and record.get('run_id') == run_id and
             set(coverage) == HOLD_ITEMS and record.get('join_problems') == [] and
             not record.get('errors') and
             all(v in ('pass', 'not exercised') for v in coverage.values()) and
             isinstance(record.get('attempts'), list) and
             all(isinstance(a, dict) and a.get('status') in ('pass', 'partial') and
                 isinstance(a.get('coverage'), dict) and
                 all(v in ('pass', 'not exercised') for v in a['coverage'].values())
                 for a in record['attempts']))
    expected = (valid and phase.get('exit_code') == 1 and record.get('status') == 'incomplete' and
                bool(missing) and missing <= allowed.keys())
    unneeded = valid and phase.get('exit_code') == 0 and record.get('status') == 'pass' and not missing
    mapping = 'expected not exercised' if expected else 'no longer needed; remove from expected-not-exercised list' if unneeded else 'unexpected'
    return dict(review='populated-holds', status=record.get('status'), mapping=mapping,
                items=[dict(item=e['item'], status=coverage.get(e['item'], 'missing'),
                            reason=e['reason'], owner=e['owner'], established_by=e['established_by'])
                       for e in entries], coverage=coverage, join_problems=record.get('join_problems'))


def summarize(root, phases, run_id):
    expected=json.loads(Path(__file__).with_name('expected-blocked.json').read_text())
    allowed={(e['case'],e['row']):e for e in expected}
    rows=[];errors=[];reviews=[];phase_errors=[]
    for case in ('AT10','AT11','AT12','AT13'):
        path=root/(case+'.json')
        if not path.exists(): errors.append(case+': current record missing');continue
        record=json.loads(path.read_text())
        if record.get('run_id')!=run_id: errors.append(case+': different run');continue
        if not record.get('checks'): errors.append(case+': no executed checks')
        for check in record.get('checks',[]):
            entry=allowed.get((case,check['name']))
            status=check['status']
            mapping='pass' if status=='pass' else 'unexpected'
            if entry and status=='blocked': mapping='expected blocked'
            if entry and status=='pass': mapping='unblocked; remove from expected-blocked list'
            rows.append(dict(case=case,row=check['name'],status=status,mapping=mapping,
                             reason=check.get('actual',''),owner=check.get('owner',''),
                             expected=entry))
            if mapping=='unexpected':errors.append(case+' '+check['name']+': '+status)
    acceptance_errors = list(errors)
    for phase in phases:
        mapping = review_mapping(root, phase, run_id)
        if mapping:
            reviews.append(mapping)
        if phase.get('exit_code') != 0 and not (mapping and mapping['mapping'] == 'expected not exercised'):
            detail = phase['name']+': nonzero exit '+str(phase.get('exit_code'))
            phase_errors.append(detail)
            # Keep the driver exit visible without counting the aggregate row
            # failure twice. It still makes the job red, even if all rows pass.
            if phase['name'] != DRIVER_PHASE or not acceptance_errors:
                errors.append(detail)
        elif mapping and mapping['mapping'] == 'unexpected':
            errors.append(phase['name']+': inconsistent review result')
    result=dict(run_id=run_id,status='fail' if errors or phase_errors else 'pass',rows=rows,reviews=reviews,phases=phases,errors=errors,phase_errors=phase_errors,
                scope='Scheduled job only. Acceptance statuses unchanged. Only explicitly listed same-run review coverage gaps map to expected not exercised. Unknown blocks, any failure, join problem or unexplained subprocess error stay red.')
    (root/'scheduled-job.json').write_text(json.dumps(result,indent=2)+'\n')
    lines=['Scheduled at-isolation: **'+result['status']+'**','', '| Case / row | Acceptance | Job mapping | Reason / owner |','|---|---|---|---|']
    clean=lambda v:' '.join(str(v).split()).replace('|','\\|')
    for r in rows:
        reason=r['reason']+' / '+r['owner']
        if r['expected']:reason+='; expected: '+r['expected']['reason']+' / '+r['expected']['owner']
        lines.append('| '+' | '.join(map(clean,[r['case']+' / '+r['row'],r['status'],r['mapping'],reason]))+' |')
    if reviews:
        lines += ['', '| Review / item | Review coverage | Job mapping | Reason / owner |', '|---|---|---|---|']
        for r in reviews:
            for item in r['items']:
                lines.append('| '+' | '.join(map(clean, [r['review']+' / '+item['item'], item['status'], r['mapping'], item['reason']+' / '+item['owner']]))+' |')
    if phase_errors:lines += ['', 'Nonzero phases (aggregate driver failures share the acceptance error above):'] + ['- '+clean(e) for e in phase_errors]
    if errors:lines+=['','Errors:']+['- '+clean(e) for e in errors]
    (root/'scheduled-job.md').write_text('\n'.join(lines)+'\n')
    return result
