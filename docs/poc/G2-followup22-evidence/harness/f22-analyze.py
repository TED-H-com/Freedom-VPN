from pathlib import Path
import hashlib,json,math
root=Path('.agent/f22-runtime')
out=Path('docs/poc/G2-followup22-evidence')

def write(p,x):p.parent.mkdir(parents=True,exist_ok=True);p.write_text(json.dumps(x,indent=2)+'\n')

def percentiles(values):
 values=sorted(values)
 return {key:round(values[min(len(values)-1,math.ceil(p*len(values))-1)]/1e6,6) for key,p in [('p50',.5),('p90',.9),('p99',.99),('max',1)]} if values else None

def summary(name,rows):
 guards=[];failed=[];seen=set()
 for index,original in enumerate(rows):
  row=original.get('result',original)
  timing=row.get('timing')
  if not timing or timing.get('operation')!='apply': continue
  key=timing['start_ns']
  if key in seen:continue
  seen.add(key)
  d=timing['deadline']
  if d['guard_released']:
   guard=dict(index=index,sequence=row.get('sequence'),generation=timing['candidate_generation'],deadline=d,commands_observed=len(timing.get('commands',[])),outer_ns=str(int(d['release_end_ns'])-int(d['install_start_ns'])),install_ns=str(int(d['installed_by_ns'])-int(d['install_start_ns'])),render_to_installed_ns=str(int(d['installed_by_ns'])-int(d['render_start_ns'])))
   guards.append(guard)
  if not timing['succeeded']:
   if not int(d['install_start_ns']):stage='pre_install'
   elif not d['install_ok']:stage='install_command'
   elif int(d['installed_by_ns'])-int(d['render_start_ns'])>int(d['install_bound_ns']):stage='install_budget'
   elif not d['guard_released']:stage='release'
   elif not timing['verified']:stage='verification_or_quarantine'
   else:stage='verified_post_apply'
   failed.append(dict(index=index,failure_stage=stage,cause_status='unproven' if stage in ['install_command','install_budget'] else 'not_applicable',apply_bracket=dict(start_ns=timing['start_ns'],end_ns=timing['end_ns']),record=original))
 value=dict(test_only=True,population=name,cause_status_scope='install-overrun causality only; not_applicable does not assert a cause for another failure',applies=len(seen),completed_guards=len(guards),failed_applies=len(failed),measured_guards=sum(g['commands_observed']>0 for g in guards),percentiles_ms={field:percentiles([int(g[field]) for g in guards]) for field in ['outer_ns','install_ns','render_to_installed_ns']},guards=guards,failures=failed)
 write(out/'analysis'/(name+'.json'),value)
 return {k:v for k,v in value.items() if k not in ['guards','failures']}

result=[]
for p in (out/'analysis').glob('*.json'):p.unlink()
for leg in ['A1','B1','A2','B2']:
 for suite in ['live','agent','agent-trace']:
  path=root/(leg+'-'+suite)
  if not path.exists():continue
  if suite=='live':rows=[json.loads(p.read_text()) for p in sorted((path/'commands').glob('*.json'))]
  elif suite=='agent':
   file=path/'agent/suite/operations.jsonl'
   rows=[json.loads(line) for line in file.read_text().splitlines()] if file.exists() else []
  else:
   file=path/'live/TestD116_AT10_InstallerMarkerEnrollment/agent.txt'
   rows=[json.loads(line) for line in file.read_text().splitlines() if line.startswith('{') and '"event":"apply"' in line] if file.exists() else []
   producer=path/'agent/suite/operations.jsonl'
   native={}
   if producer.exists():
    for line in producer.read_text().splitlines():
     r=json.loads(line)
     if r.get('timing',{}).get('operation')=='apply':native[r['timing']['start_ns']]=r
   for r in rows:
    joined=native.get(r['timing']['start_ns'])
    if joined:
     assert joined['timing']==r['timing']
     r['producer']=joined
     r['sequence']=joined['sequence']
  if rows:
   item=summary(leg+'-'+suite,rows)
   status=path/'status.json'
   item['suite_status']=json.loads(status.read_text()) if status.exists() else {'status':'incomplete'}
   result.append(item)
path=root/'B1-diagnostic/live/TestF22_D133_AT02_CommandLifecycle/apply.jsonl'
if path.exists():result.append(summary('B1-diagnostic',[json.loads(line) for line in path.read_text().splitlines()]))
# Retain full failed-Apply brackets and guard vectors for excluded attempts too.
for attempt in sorted((root/'unqualified').glob('*')):
 if not attempt.is_dir():continue
 rows=[]
 for p in sorted(attempt.rglob('commands/*.json')):rows.append(json.loads(p.read_text()))
 for p in sorted(attempt.rglob('suite/operations.jsonl')):rows.extend(json.loads(x) for x in p.read_text().splitlines())
 if rows:summary('unqualified-'+attempt.name,rows)
required=[leg+'-'+suite for leg in ['A1','B1','A2','B2'] for suite in ['live','agent']]
missing=[name for name in required if not (root/name/'status.json').exists() or not json.loads((root/name/'status.json').read_text())['postflight_quiet']]
blocked=json.loads((root/'blocked.json').read_text()) if (root/'blocked.json').exists() else None
write(out/'comparison-summary.json',dict(test_only=True,required_populations=required,missing_qualified_populations=missing,blocked=blocked,populations=result))
for p in result:print(p['population'],p['applies'],p['completed_guards'],p['failed_applies'],p['percentiles_ms'])
