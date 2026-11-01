from evidence_io import load
from pathlib import Path
import json,hashlib,datetime
root=Path('docs/e2e/at/members')
run=load(root/'AT02.json')['run_id']
streams={};receipts={}
for node,name in [('node','f14-parser-producer.jsonl'),('node2','f14-node2-producer.jsonl')]:
 path=Path('.agent/tmp')/name
 if path.exists():
  raw=path.read_bytes();streams[node]=[json.loads(line) for line in raw.splitlines()]
  receipts[node]={'bytes':str(len(raw)),'sha256':hashlib.sha256(raw).hexdigest(),'records':len(streams[node]),'scope':'private complete producer export; only exact joined windows published'}
rows=[]
parse=lambda s:datetime.datetime.fromisoformat(s.replace('Z','+00:00'))
for case in ['AT02','AT08','AT09']:
 v=load(root/(case+'.json'))
 if v['run_id']!=run:continue
 for s in v['steps']:
  if not s['name'].startswith('agent transition '):continue
  a=json.loads(s['actual']);node='node2' if 'namespace' in s['name'] else 'node'
  data=streams.get(node,[]);started=parse(a['started_at']);newrun=next((r['run_id'] for r in data if int(r['sequence'])==1 and parse(r['anchor']['wall'])>=started),None)
  initial=[r for r in data if r['run_id']==newrun][:8] if newrun else []
  # Stop-only rows use their exact closed-storage readback; do not join an
  # unrelated later restart just because it happens in the same stream.
  if s['name'].endswith('stop'):initial=[]
  interpretation=a['stored_lease_status']
  prior=[]
  if initial and initial[0].get('lease'):
   lease=initial[0]['lease']['lease_id']
   old=[r for r in data if r['run_id']!=newrun and r.get('lease',{}).get('lease_id')==lease and parse(r['anchor']['wall'])<=parse(initial[0]['anchor']['wall'])]
   if old:
    prior=old[-2:]
    interpretation='new producer retains a prior-run lease before its first renewal; original command snapshots remain unchanged'
  rows.append({'case':case,'original_check':s,'duration_seconds':int(a['duration_ns'])/1e9,'additional_interpretation':interpretation,'prior_producer_records':prior,'first_new_producer_records':initial,'limit':'command intervals and sampled stored state; namespace restart interval includes the owner 90-second observation window; no outage ceiling or repeated-restart reliability claim'})
(root/'F14-agent-transitions.json').write_text(json.dumps({'run_id':run,'source_commit':load(root/'AT02.json')['source_commit'],'producer_exports':receipts,'transitions':rows},indent=2)+'\n')
print('agent transition rows',len(rows))
