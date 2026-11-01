from pathlib import Path
import json
root=Path('docs/poc/G2-followup23-evidence')
summary=[]
for name in ['lab','lab-guard','lab-final']:
 p=root/name
 mutations=list(map(json.loads,(p/'mutations.jsonl').read_text().splitlines()))
 attempts=list(map(json.loads,(p/'attempts.jsonl').read_text().splitlines()))
 for group in json.loads((p/'summary.json').read_text()):
  rows=[r for r in attempts if r['condition']==group['condition'] and r['command']==group['command']]
  writes=[r for r in mutations if r['condition']==group['condition'] and r['ordinal']>0 and r['exit']==0]
  overlaps=contained=0
  for r in rows:
   a,b=int(r['start_ns']),int(r['end_ns'])
   overlaps+=any(int(w['start_ns'])<b and int(w['end_ns'])>a for w in writes)
   contained+=any(int(w['start_ns'])>=a and int(w['end_ns'])<=b for w in writes)
  assert len(rows)==group['attempts']
  assert sum(bool(r.get('duplicates')) for r in rows)==group['duplicate_listings']
  assert sum(bool(r.get('mixed_generations')) for r in rows)==group['mixed_generation_listings']
  summary.append(dict(run=name,**group,writer_bracket_overlaps=overlaps,writer_bracket_contained=contained))
(root/'reproduction-summary.json').write_text(json.dumps(summary,indent=2)+'\n')
print('listings',sum(r['attempts'] for r in summary),'duplicates',sum(r['duplicate_listings'] for r in summary),'overlap',sum(r['writer_bracket_overlaps'] for r in summary),'contained',sum(r['writer_bracket_contained'] for r in summary))
for condition in dict.fromkeys(r['condition'] for r in summary):
 rows=[r for r in summary if r['condition']==condition]
 print(condition,'table',sum(r['attempts'] for r in rows if 'table' in r['command']),'ruleset',sum(r['attempts'] for r in rows if 'ruleset' in r['command']),'writer attempts',sum(r['writer'].get('attempts',0) for r in rows if 'table' in r['command']))
