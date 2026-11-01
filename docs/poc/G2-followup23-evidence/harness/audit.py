from pathlib import Path
import json,hashlib,runpy
root=Path('docs/poc/G2-followup23-evidence')
counts={'listing_attempts':0,'retained_listing_outputs':0,'successful_mutations':0,'failed_setup_mutations':0}
for name in ['lab-attempt1-setup-failed','lab','lab-guard','lab-final']:
 p=root/name; harness=runpy.run_path(str(p/'harness.py')); table=harness['table']; rules=harness['rules']
 for row in map(json.loads,(p/'attempts.jsonl').read_text().splitlines()):
  counts['listing_attempts']+=1
  if 'retained_output' in row:
   raw=(p/row['retained_output']).read_bytes(); assert len(raw)==row['bytes']; assert hashlib.sha256(raw).hexdigest()==row['sha256']; counts['retained_listing_outputs']+=1
   facts=harness['inspect'](raw)
   assert all(row[k]==v for k,v in facts.items())
 for row in map(json.loads,(p/'mutations.jsonl').read_text().splitlines()):
  mode=row['condition'].split('-')[0]; gen=row['ordinal']
  if mode=='cleanup': script='delete table inet fvpn\n'
  else:
   _,chains,_,rule_count,_=row['condition'].split('-'); chains,rule_count=int(chains),int(rule_count)
   if gen==0: script='destroy table inet fvpn\n'+table(1,chains,rule_count)
   elif mode=='replace': script='delete table inet fvpn\n'+table(gen,chains,rule_count)
   else:
    script='flush table inet fvpn\nflush set inet fvpn fvmarker\n'
    script+=f'add element inet fvpn fvmarker {{ {gen} }}\n'+rules(gen,rule_count)
    if mode=='guard':
     script+=('add chain inet fvpn fvguard { type filter hook forward priority -100; policy drop; }\n' if gen%2==0 else 'delete chain inet fvpn fvguard\n')
  assert hashlib.sha256(script.encode()).hexdigest()==row['script_sha256']
  if row['exit']:
   assert name=='lab-attempt1-setup-failed' and gen==0; counts['failed_setup_mutations']+=1
   raw=(p/(row['condition']+'-mutation-0.txt')).read_bytes()
   assert len(raw)==row['output_bytes'] and hashlib.sha256(raw).hexdigest()==row['output_sha256']
  else:
   counts['successful_mutations']+=1
   assert row['output_bytes']==0 and row['output_sha256']==hashlib.sha256(b'').hexdigest()
(root/'attempt-audit.json').write_text(json.dumps({'result':'pass',**counts},indent=2)+'\n')
print(counts)
