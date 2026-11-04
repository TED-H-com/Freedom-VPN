"""Index original follow-up 24 results; never evaluate or replace a predicate."""
from pathlib import Path
from collections import Counter
import json, hashlib, sys
sys.path.insert(0,'tests/e2e/at/members')
from evidence_io import load, publish
ROOT=Path('docs/e2e/at/members')
IDS=['AT05','AT01','AT02','AT03','AT04','AT08','AT20','AT21','AT22','AT09']
def save(name,data):
 p=ROOT/name; p.write_text(json.dumps(data,indent=2)+'\n'); publish(p)
def parsed(value):
 try: return json.loads(value)
 except (ValueError,TypeError): return None
def main():
 cases={i:load(ROOT/(i+'.json')) for i in IDS}; run=cases['AT05']['run_id']; source=cases['AT05']['source_commit']
 assert run == sys.argv[1], 'not the measured follow-up 24 run'
 assert source == sys.argv[2], 'not the measured follow-up 24 source'
 assert all(x['run_id']==run and x['complete'] for x in cases.values())
 provenance=dict(run_id=run,source_commit=source)
 counts=Counter(); rows=[]; allchecks={}; writes={}
 def scan(obj,location):
  if isinstance(obj,str):
   v=parsed(obj)
   if isinstance(v,(dict,list)): scan(v,location)
  elif isinstance(obj,list):
   for i,v in enumerate(obj): scan(v,location+'/'+str(i))
  elif isinstance(obj,dict):
   if 'attempts' in obj and 'max_attempts' in obj and 'outcome' in obj:
    h=hashlib.sha256(json.dumps(obj,sort_keys=True).encode()).hexdigest()
    r=writes.setdefault(h,dict(receipt=obj,locations=[])); r['locations'].append(location)
    return
   for k,v in obj.items(): scan(v,location+'/'+k)
 for name,c in cases.items():
  tally=Counter(s['status'] for s in c['steps']); counts.update(tally)
  rows.append(dict(case=name,status=c['status'],requirements=c.get('spec'),counts=dict(tally),blockers=c.get('blockers',[])))
  allchecks[name]=[dict(index=i,**{k:s[k] for k in ['name','status','expected','at','command']}) for i,s in enumerate(c['steps'])]
  scan(c,name)
 save('F24-results.json',dict(provenance,cases=rows,case_counts=dict(Counter(c['status'] for c in cases.values())),subcheck_counts=dict(counts),subchecks=allchecks))
 write_rows=[]
 for sha,v in writes.items():
  r=v['receipt']; conflicts=[i for i,a in enumerate(r['attempts']) if a.get('status')==412]
  write_rows.append(dict(sha256=sha,**v,conflict_attempt_indexes=conflicts))
 save('F24-node-writes.json',dict(provenance,distinct_receipts=len(writes),conflicting_receipts=sum(bool(x['conflict_attempt_indexes']) for x in write_rows),conflict_attempts=sum(len(x['conflict_attempt_indexes']) for x in write_rows),receipts=write_rows))
 restart=[dict(index=i,**s) for i,s in enumerate(cases['AT09']['steps']) if any(x in s['name'] for x in ['restart','offline','namespace','provisional','TC_FOREIGN_FILTER'])]
 save('F24-AT09-restart.json',dict(provenance,original_steps=restart))
 recovery=load(ROOT/'F18-AT02-recovery.json'); assert recovery['run_id']==run
 compact={k:v for k,v in recovery.items() if k not in ['sender','observation','apply_records','brackets','baseline_calls','closed_recovery_steps','interface_down_intervals','lost_sends']}
 compact['brackets']=[{k:v for k,v in b.items() if not isinstance(v,(dict,list)) and k not in ['original_baseline_record','original_recovery_record','later_live_verification','later_heartbeat']} for b in recovery['brackets']]
 compact['closed_recovery_steps']=recovery.get('closed_recovery_steps',[])
 compact['original_artifact']='F18-AT02-recovery.json'+('.gz' if not (ROOT/'F18-AT02-recovery.json').exists() else '')
 compact['accounting_checks']=[dict(index=i,**s) for i,s in enumerate(cases['AT02']['steps']) if s['name'] in ['D-135 D-136 node-wide baseline transitions (provisional)','F11-03 D-135 admitted sibling recovery']]
 # The checks retain their original complete actual strings in the case files.
 for s in compact['accounting_checks']: s.pop('actual',None)
 save('F24-AT02-recovery.json',compact)
 mtu=[]; d146=[]
 for i,s in enumerate(cases['AT09']['steps']):
  if any(x in s['name'] for x in ['MTU','D-137','D-146','relay']): mtu.append(dict(index=i,**s))
  if s['name'].endswith(' D-146 clamp evidence'):
   v=parsed(s['actual']); label=s['name'].removesuffix(' D-146 clamp evidence'); transfer=next(x for x in cases['AT09']['steps'] if x['name']==label+' transfer'); upload=next(x for x in cases['AT09']['steps'] if x['name']==label+' upload'); tr=parsed(transfer['actual'])
   d146.append(dict(index=i,row=label,selected_family=tr['selected_exit_before']['chosen_family'],transfer_status=transfer['status'],upload_status=upload['status'],evidence_status=s['status'],transfer=tr['transfer'],qualified=v['qualified'],checks=v.get('checks'),serving_rules={k:x for k,x in v.get("rules",{}).items() if k!="pairs"},reasons=v.get('reasons'),endpoint_mss={k:sorted({p['mss'] for p in ps},key=lambda x:-1 if x is None else x) for k,ps in v.get('endpoint_mss',{}).items()},reverse={k:v for k,v in v.get('reverse',{}).items() if k not in ['sender_packets','receiver_packets']},no_ptb=v.get('no_ptb')))
 save('F24-native-MTU.json',dict(provenance,original_steps=mtu,D146=d146,limits='D-137/D-146 receive-only UDP and QUIC without PTB remain limited; no physical platform or arm64 qualification.'))
 lines=['# Follow-up 24 observed subchecks','',f'Run `{run}`, source `{source}`. All statuses copied from original case records; no replay or predicate rewrite.','']
 for name in IDS:
  lines+=['## '+name,'',f"Case status: **{cases[name]['status']}**. Original case retains every actual value, timestamp and expected value.",'','| Index | Check | Status |','|---:|---|---|']
  lines += [f"| {s['index']} | {s['name'].replace('|','/')} | {s['status']} |" for s in allchecks[name]]
  lines+=['']
 (ROOT/'F24-subchecks.md').write_text('\n'.join(lines))
 print(json.dumps(dict(provenance,cases=rows,subchecks=counts,node_write_conflicts=sum(len(x['conflict_attempt_indexes']) for x in write_rows),recovery_brackets=len(recovery['brackets']))))
if __name__=='__main__': main()
