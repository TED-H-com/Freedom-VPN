#!/usr/bin/env python3
"""Compact per-Apply command inventory; original command records stay lossless."""
from collections import Counter
import json
from pathlib import Path
from producer_timing import command_lifecycle, read_operations


def review(root):
    run_id=json.loads((root/'AT10.json').read_text())['run_id']
    counts,phases,skips=Counter(),Counter(),Counter()
    failed=[];applies=0
    for row in read_operations(root):
        if row.get('timing',{}).get('operation')!='apply': continue
        applies+=1
        lifecycle=command_lifecycle(row)
        counts['applies_with_'+lifecycle['status']]+=1
        for item in lifecycle['records']:
            c=item['record'];counts['records']+=1;phases[c['phase']]+=1
            for key in ['started','ok','exit_available','signal_available','cpu_available','switches_available']:
                counts[key]+=int(c[key])
            for key in ['child_start_exact','cancel_observed']:
                counts[key]+=int(c[key]['available'])
            for side in ['before','after']:
                counts[side+'_host_ancestors_available']+=int(c[side]['host_ancestors_available'])
                if c[side].get('skip_reason'):skips[side+':'+c[side]['skip_reason']]+=1
            if not c['ok']:failed.append(item)
    out=dict(run_id=run_id,applies=applies,counts=counts,phases=phases,resource_sampling_omissions=skips,failed_commands=failed,
        scope='Every Apply and original command index, including unsuccessful commands. Skipped resource samples stay unavailable; command exit and CPU fields remain independent. Exact child start and hidden host ancestors are not inferred.')
    (root/'command-lifecycle-summary.json').write_text(json.dumps(out,indent=2)+'\n')
    events=json.loads((root/'container-lifecycle-events.json').read_text())['events']
    stops=[]
    for i,event in enumerate(events):
        if event['action']!='kill' or event['attributes'].get('signal')!='15':continue
        container=event['attributes']['name']
        die=next((e for e in events[i+1:] if e['attributes']['name']==container and e['action']=='die'),None)
        if die:
            kills=[e for e in events[i+1:] if e['attributes']['name']==container and e['action']=='kill' and e['attributes'].get('signal')=='9' and int(e['time_ns'])<int(die['time_ns'])]
            stops.append(dict(container=container,term=event,die=die,term_to_die_ns=str(int(die['time_ns'])-int(event['time_ns'])),sigkill_events=kills,
                scope='Container PID1 lifecycle; cannot qualify the agent cooperative shutdown bound. Join restart/clone/terminal timestamps separately.'))
    (root/'container-stop-summary.json').write_text(json.dumps(dict(run_id=run_id,stops=stops),indent=2)+'\n')
    return out


if __name__=='__main__':
    import sys
    result=review(Path(sys.argv[1]));print(json.dumps({k:result[k] for k in ['run_id','applies','counts','resource_sampling_omissions']},indent=2))
