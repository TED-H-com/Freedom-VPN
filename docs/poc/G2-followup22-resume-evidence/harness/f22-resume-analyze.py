from pathlib import Path
import hashlib,json,math
root=Path('.agent/f22-resume-runtime')
out=Path('docs/poc/G2-followup22-resume-evidence')

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

import re
out.mkdir(exist_ok=True)
results=[];tests={};joins=[]
for leg in ['R-A1','R-B1','R-A2','R-B2']:
 for suite in ['agent','agent-trace']:
  name=leg+'-'+suite;base=root/name
  status=base/'status.json'
  if not status.exists():continue
  state=json.loads(status.read_text())
  if not state['postflight_quiet']:continue
  producer=base/'agent/suite/operations.jsonl'
  rows=[json.loads(x) for x in producer.read_text().splitlines()] if producer.exists() else []
  if suite=='agent-trace':
   trace=base/'live/TestD116_AT10_InstallerMarkerEnrollment/agent.txt'
   traced=[json.loads(x) for x in trace.read_text().splitlines() if x.startswith('{') and '"event":"apply"' in x]
   native={r['timing']['start_ns']:r for r in rows if r.get('timing',{}).get('operation')=='apply'}
   for r in traced:
    n=native[r['timing']['start_ns']];assert n['timing']==r['timing']
    joins.append(dict(population=name,start_ns=r['timing']['start_ns'],run_id=n['run_id'],boot_id=n['boot_id'],sequence=n['sequence']))
    r['sequence']=n['sequence'];r['run_id']=n['run_id'];r['boot_id']=n['boot_id']
   rows=traced
  item=summary(name,rows);item['suite_status']=state
  log=(root/(name+'-suite.log')).read_text()
  tests[name]={test:outcome for outcome,test in re.findall(r'^--- (PASS|FAIL|SKIP): (\S+)',log,re.M)}
  item['tests']=tests[name];results.append(item)
for attempt in sorted((root/'unqualified').glob('*')):
 rows=[]
 for p in sorted(attempt.rglob('suite/operations.jsonl')):rows.extend(json.loads(x) for x in p.read_text().splitlines())
 if rows:summary('excluded-'+attempt.name,rows)
required=[leg+'-'+suite for leg in ['R-A1','R-B1','R-A2','R-B2'] for suite in ['agent','agent-trace']]
missing=[name for name in required if name not in tests]
comparisons=[];b_only=[];baseline_failures=[]
for suite in ['agent','agent-trace']:
 a_names=['R-A1-'+suite,'R-A2-'+suite]
 for a in a_names:
  for test,status in tests.get(a,{}).items():
   if status=='FAIL':baseline_failures.append(dict(population=a,test=test))
 for b in ['R-B1-'+suite,'R-B2-'+suite]:
  for test,status in tests.get(b,{}).items():
   if status=='FAIL' and all(tests.get(a,{}).get(test)=='PASS' for a in a_names):b_only.append(dict(population=b,test=test,A_controls=a_names))
  if not all(n in tests for n in a_names+[b]):continue
  arows=[json.loads((out/'analysis'/(n+'.json')).read_text()) for n in a_names]
  brow=json.loads((out/'analysis'/(b+'.json')).read_text())
  for metric in ['outer_ns','install_ns','render_to_installed_ns']:
   for percentile,num,den in [('p50',1,2),('p90',9,10),('p99',99,100),('max',1,1)]:
    def quant(row):
     values=sorted(int(g[metric]) for g in row['guards'])
     return values[(num*len(values)+den-1)//den-1]
    av=[quant(r) for r in arows];bv=quant(brow)
    comparisons.append(dict(population=b,metric=metric,percentile=percentile,A_values_ns=list(map(str,av)),A_min_ns=str(min(av)),A_max_ns=str(max(av)),B_ns=str(bv),within=min(av)<=bv<=max(av),direction='below' if bv<min(av) else 'above' if bv>max(av) else 'inside'))
blocked=json.loads((root/'blocked.json').read_text()) if (root/'blocked.json').exists() else None
write(out/'comparison.json',dict(test_only=True,required=required,missing=missing,blocked=blocked,populations=results,producer_joins=joins,baseline_test_failures=baseline_failures,B_only_test_failures=b_only,distribution_comparisons=comparisons))
lines=['| Population | Applies / guards / failed Applies | Outer p50 / p90 / p99 / max (ms) | Install p50 / p90 / p99 / max (ms) | Render→installed p50 / p90 / p99 / max (ms) |','|---|---:|---|---|---|']
for r in results:
 def numbers(k):return ' / '.join(f"{r['percentiles_ms'][k][q]:.6f}" for q in ['p50','p90','p99','max'])
 lines.append(f"| {r['population']} | {r['applies']} / {r['completed_guards']} / {r['failed_applies']} | {numbers('outer_ns')} | {numbers('install_ns')} | {numbers('render_to_installed_ns')} |")
(out/'distribution-table.md').write_text('\n'.join(lines)+'\n')
print('Populations',len(results),'missing',missing,'A test failures',baseline_failures,'B-only test failures',b_only)
print('Spread components outside:',sum(not r['within'] for r in comparisons),'of',len(comparisons))
failed_index=[]
for p in sorted((out/'analysis').glob('*.json')):
 d=json.loads(p.read_text())
 for f in d['failures']:
  r=f['record'];body=r.get('result',r)
  failed_index.append(dict(population=d['population'],analysis_path=str(p.relative_to(out)),row_index=f['index'],run_id=body.get('run_id'),boot_id=body.get('boot_id'),sequence=body.get('sequence'),failure_stage=f['failure_stage'],cause_status=f['cause_status'],apply_bracket=f['apply_bracket'],deadline=body['timing']['deadline']))
write(out/'failed-apply-index.json',dict(test_only=True,cause_status_scope='install-overrun causality only',rows=failed_index))
