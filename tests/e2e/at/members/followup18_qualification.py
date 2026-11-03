"""Read-only qualification of the single assembled run; no predicate changes."""
from pathlib import Path
import sys,json,ipaddress
sys.path.insert(0,'tests/e2e/at/members')
from evidence_io import load,publish
root=Path('docs/e2e/at/members')
cases={name:load(root/(name+'.json')) for name in ['AT05','AT01','AT02','AT03','AT04','AT08','AT20','AT21','AT22','AT09']}
run=cases['AT02']['run_id'];assert run!='81b638e8-be59-4996-9594-14dd402ebb86'
assert {c['run_id'] for c in cases.values()}=={run}
steps=cases['AT02']['steps'];restores=[]
for i,s in enumerate(steps):
 if not s['name'].startswith('conditional hold restoration '):continue
 value=json.loads(s['actual']);following=steps[i+1] if i+1<len(steps) else None
 patch=following if following and following['name']=='conditional suspension update' else None
 restores.append(dict(original=s,member_state=value,immediately_following_patch=patch,consistent=bool(patch)==value['mutation_needed']))
holds=load(root/'F18-AT02-generation-holds.json');audits=[a for a in holds['joined']['revocations'] if a['action']=='member.patch']
checks=cases['AT09']['steps'];http=cases['AT09'].get('member_http',[])
reauth=[s for s in checks if s['name']=='AT09 member OIDC reauthentication']
transitions=[dict(original_step=s,record=json.loads(s['actual'])) for s in checks if s['name'].startswith('agent transition selected namespace')]
sets=[]
for row in transitions:
 for key in ['stored_lease_before','stored_lease_after']:
  value=row['record'].get(key,{})
  if 'retained_restart_sets' in value:
   sets.append(dict(transition=row['original_step']['name'],side=key,read_at=value['read_at'],sets=value['retained_restart_sets']))
def canonical(value):
 return {key:None if value.get(key) is None else sorted({str(ipaddress.ip_network(v,strict=False)) for v in value[key]}) for key in ['denied_v4','extra_denied_v4','host_v4']}
equal=len(sets)==4 and all(canonical(v['sets'])==canonical(sets[0]['sets']) for v in sets)
failed=[s for s in checks if s['status']=='fail']
non2xx=[dict(case=name,response=v) for name,case in cases.items() for v in case.get('member_http',[]) if not 200<=v['status']<300]
rows=[]
for name in ['alarm-recovery','sibling-removal','sole-member','retained-safety']:
 path=root/('F18-hold-'+name+'.json')
 if not path.exists() and not Path(str(path)+'.gz').exists():
  rows.append(dict(row=name,coverage='not exercised',incident=None,survivor_generation_join=False,original_blockers=cases['AT02'].get('blockers',[])))
  continue
 v=load(path);central=v.get('central') or {};incident=next((h for h in central.get('incidents',[]) if h['id']==v.get('incident_id')),None)
 proxy=v.get('proxy') or {};post=v.get('accepted_post_command_survivor');recovery=v.get('recovery_serving_report')
 events=proxy.get('events') or []
 rows.append(dict(row=name,arm=v.get('arm'),incident=incident,entry_hold_report_id=v.get('entry_hold_report_id'),alarm_hold_report_id=v.get('alarm_hold_report_id'),escalation_audit_id=v.get('escalation_audit_id'),member_revocation=v.get('member_revocation'),post_command_report=post,initial_target_generation=v.get('initial_target_generation'),proxy=proxy,local_recovery=v.get('final_local_lease'),recovery_serving_report=recovery,survivor_generation_join=bool(post and post['applied_generation']==proxy.get('continued_generation')),forwarded_body_matches_local=v.get('forwarded_body_matches_local'),original_retained_lease=v.get('retained_local_lease'),original_before=v.get('before')))
value=dict(run_id=run,source_commit=cases['AT02']['source_commit'],conditional_restoration=dict(records=restores,consistent=all(v['consistent'] for v in restores)),member_patch_query=dict(original_live_audits=audits,no_supplement_used=True),AT09_reauthentication=dict(records=reauth,member_responses=http,non2xx=[v for v in http if not 200<=v['status']<300],unauthorized_count=sum(v['status']==401 for v in http)),all_member_endpoint_non2xx=non2xx,AT09_original_failed_steps=failed,AT09_restart_sets=dict(records=sets,all_four_record_sides_present=len(sets)==4,distinct_read_times=len({x['read_at'] for x in sets}),sets_unchanged=equal,scope='Retained installed sets immediately before stop, after stop and after restart; the stopped read is reused on both adjacent transition records. Compare masked prefix sets; nil DeniedV4 retains spec-default semantics. This readback does not instrument the exact Boot decision.'),hold_rows=rows,source_address_fault_hold=dict(coverage='not exercised',needed='Owner-provided independent source-address/binding fault hook with bounded cleanup; TC fault cannot substitute.'),historical_scope='Historical F16/F17 failures unchanged. D-135/D-136 provisional; F11-01/F14-01 causes unproven; Only the D-137 receive-only TCP downgrade changes under complete D-146 evidence.')
p=root/'F18-qualification.json';p.write_text(json.dumps(value,indent=2)+'\n');publish(p)
print('conditional restores',len(restores),'consistent',all(v['consistent'] for v in restores),'member.patch audits',len(audits),'AT09 401s',sum(v['status']==401 for v in http),'restart sets unchanged',equal)
print('AT09 failures',[s['name'] for s in failed])
print('hold rows',[(r['row'],bool(r['incident']),r['survivor_generation_join']) for r in rows])
