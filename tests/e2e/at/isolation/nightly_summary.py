"""D-144 scheduled exit mapping is separate from unchanged acceptance records."""
import json
from pathlib import Path


def summarize(root, phases, run_id):
    expected=json.loads(Path(__file__).with_name('expected-blocked.json').read_text())
    allowed={(e['case'],e['row']):e for e in expected}
    rows=[];errors=[]
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
    for phase in phases:
        if phase['exit_code']!=0:errors.append(phase['name']+': nonzero exit '+str(phase['exit_code']))
    result=dict(run_id=run_id,status='fail' if errors else 'pass',rows=rows,phases=phases,errors=errors,
                scope='Scheduled job only. Acceptance statuses unchanged. Unknown blocks, any failure or subprocess error stay red.')
    (root/'scheduled-job.json').write_text(json.dumps(result,indent=2)+'\n')
    lines=['Scheduled at-isolation: **'+result['status']+'**','', '| Case / row | Acceptance | Job mapping | Reason / owner |','|---|---|---|---|']
    clean=lambda v:' '.join(str(v).split()).replace('|','\\|')
    for r in rows:
        reason=r['reason']+' / '+r['owner']
        if r['expected']:reason+='; expected: '+r['expected']['reason']+' / '+r['expected']['owner']
        lines.append('| '+' | '.join(map(clean,[r['case']+' / '+r['row'],r['status'],r['mapping'],reason]))+' |')
    if errors:lines+=['','Errors:']+['- '+clean(e) for e in errors]
    (root/'scheduled-job.md').write_text('\n'.join(lines)+'\n')
    return result
