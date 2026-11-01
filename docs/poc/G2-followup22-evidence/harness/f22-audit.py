from pathlib import Path
import json,hashlib,collections
root=Path('.agent/f22-runtime')
records=[]
for p in sorted(root.glob('B*-*/live/**/at10-apply.jsonl'))+sorted(root.glob('B*-*/live/**/apply.jsonl')):
 records.extend(json.loads(x) for x in p.read_text().splitlines())
for p in sorted(root.glob('B*-agent-trace/live/**/agent.txt')):
 records.extend(json.loads(x) for x in p.read_text().splitlines() if x.startswith('{') and '"event":"apply"' in x)
commands=[];hashes=0
for row in records:
 t=row['timing'];start,end=int(t['start_ns']),int(t['end_ns'])
 for key,pin in [('test_only_input_model','test_only_input_sha256'),('test_only_render','test_only_render_sha256')]:
  assert hashlib.sha256(json.dumps(row[key],separators=(',',':'),ensure_ascii=False).encode()).hexdigest()==row[pin]
  hashes+=1
 if row['test_only_render']['rendered']:
  assert int(row['test_only_render']['timeout_reserve_ns'])==100000000
 for batch in row['batches']:
  assert hashlib.sha256(batch['test_only_script'].encode()).hexdigest()==batch['test_only_sha256'];hashes+=1
 previous=start
 for c in t.get('commands',[]):
  before,after=c['before']['bracket'],c['after']['bracket']
  assert before['before']['available'] and before['after']['available'] and after['before']['available'] and after['after']['available']
  assert previous<=int(before['before']['ns'])<=int(before['after']['ns'])<=int(c['start_attempt']['ns'])<=int(c['start_return']['ns'])<=int(after['before']['ns'])<=int(after['after']['ns'])<=end
  assert not c['child_start_exact']['available']
  if c['started']:
   assert c['exit_available'] and c['cpu_available'] and c['wait_return']['available']
   assert int(c['start_return']['ns'])<=int(c['wait_return']['ns'])<=int(after['before']['ns'])
  else:assert not c['exit_available'] and not c['cpu_available'] and not c['wait_return']['available']
  if c['cancel_observed']['available']:assert int(c['start_attempt']['ns'])<=int(c['cancel_observed']['ns'])<=int(c['wait_return']['ns'])
  assert not c['before']['host_ancestors_available'] and not c['after']['host_ancestors_available']
  previous=int(after['after']['ns']);commands.append(c)
summary=dict(test_only=True,trace_applies=len(records),verified_byte_hashes=hashes,commands=len(commands),phases=dict(collections.Counter(c['phase'] for c in commands)),cgroup_before_available=sum(c['before']['cgroup_v2_available'] for c in commands),child_cpu_available=sum(c['cpu_available'] for c in commands),child_switches_available=sum(c['switches_available'] for c in commands),exact_child_start_available=sum(c['child_start_exact']['available'] for c in commands))
summary['outcomes']=[dict(started=k[0],ok=k[1],context_before=k[2],context_after=k[3],signal=k[4],count=v) for k,v in sorted(collections.Counter((c['started'],c['ok'],c['context_before'],c['context_after'],c['signal']) for c in commands).items())]
# The same final trace is retained by the agent producer, with run/boot/sequence intact.
joins=[]
for leg in ['B1','B2']:
 base=root/(leg+'-agent-trace')
 trace=base/'live/TestD116_AT10_InstallerMarkerEnrollment/agent.txt'
 prod=base/'agent/suite/operations.jsonl'
 if not trace.exists():continue
 native={r['timing']['start_ns']:r for r in map(json.loads,prod.read_text().splitlines()) if r.get('timing',{}).get('operation')=='apply'}
 rows=[json.loads(x) for x in trace.read_text().splitlines() if x.startswith('{') and '"event":"apply"' in x]
 for r in rows:
  n=native[r['timing']['start_ns']];assert n['timing']==r['timing']
  joins.append(dict(leg=leg,run_id=n['run_id'],boot_id=n['boot_id'],sequence=n['sequence'],start_ns=r['timing']['start_ns']))
summary['producer_joins']=joins
fixture=root/'B1-diagnostic/live/TestF22_D133_AT02_CommandLifecycle'
if fixture.exists():
 rows=[json.loads(x) for x in (fixture/'apply.jsonl').read_text().splitlines()]
 assert len(rows)==7 and sum(r['test_read_fault_injected'] if 'test_read_fault_injected' in r else False for r in rows)==3
 for p in fixture.glob('baseline-*.json'):assert 'fvpn_generation=0' in p.read_text()
 packets=[json.loads(x) for x in (fixture/'node.headers.jsonl').read_text().splitlines()]
 for i,r in enumerate(rows):
  if not r.get('test_read_fault_injected'):continue
  recovered=rows[i+1]
  release=next(b for b in recovered['batches'] if b['kind']=='guard_release')
  boundary=int(release['end']['wall_ns'])
  assert any(p.get('probe_sequence',1<<33)<1<<32 and p['direction']=='out' and int(p['wall_ns'])>boundary for p in packets)
 checks=json.loads((fixture/'capture-checks.json').read_text());assert all(c['complete'] and c['kernel_drops']==0 for c in checks.values())
 summary['two_source_capture_checks']=checks
summary['passed']=True
Path('docs/poc/G2-followup22-evidence/command-audit.json').write_text(json.dumps(summary,indent=2)+'\n')
print(json.dumps({k:v for k,v in summary.items() if k not in ['producer_joins']},indent=2))
