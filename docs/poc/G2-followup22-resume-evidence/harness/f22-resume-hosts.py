from pathlib import Path
import json
root=Path('.agent/f22-resume-runtime');out=Path('docs/poc/G2-followup22-resume-evidence');out.mkdir(exist_ok=True)
rows=[]
for leg in ['R-A1','R-B1','R-A2','R-B2']:
 for suite in ['agent','agent-trace']:
  name=leg+'-'+suite;status=root/name/'status.json'
  if not status.exists():continue
  state=json.loads(status.read_text())
  if not state['postflight_quiet']:continue
  files=[]
  for stage in ['preflight','ready']:
   candidates=sorted(root.glob(name+'-'+stage+'-*.json'),key=lambda p:int(p.stem.rsplit('-',1)[1]))
   accepted=[p for p in candidates if json.loads(p.read_text())['quiet']]
   assert accepted;files.append(accepted[-1])
  files.append(root/(name+'-postflight.json'))
  samples=[]
  for p in files:
   r=json.loads(p.read_text());assert r['quiet'] and len(r['intervals'])==6
   samples.append(dict(path='runtime/'+p.name,start_ns=r['start_ns'],end_ns=r['end_ns'],max_r=max(x['r'] for x in r['intervals']),min_idle=min(x['id'] for x in r['intervals']),load_before=r['load_before'],load_after=r['load_after'],D_state_before=r['d_state_before'],D_state_after=r['d_state_after'],matched_before=len(r['suite_processes_before'].splitlines()),matched_after=len(r['suite_processes_after'].splitlines())))
  admission=json.loads((root/name/'admission-after-start.json').read_text());assert not admission['processes']
  rows.append(dict(population=name,status=state,after_startup_matching_suites=0,samples=samples))
(out/'host-summary.json').write_text(json.dumps(dict(test_only=True,window=json.loads((root/'window.json').read_text()),rows=rows),indent=2)+'\n')
print(len(rows),'qualified populations')
receipts=[]
for p in sorted(root.rglob('*.json')):
 if 'source-A' in p.parts or 'source-B' in p.parts:continue
 if not any(part in p.name for part in ['-preflight-','-ready-','-postflight.']):continue
 r=json.loads(p.read_text())
 if 'intervals' not in r:continue
 receipts.append(dict(path='runtime/'+str(p.relative_to(root)),quiet=r['quiet'],max_r=max(x['r'] for x in r['intervals']),min_idle=min(x['id'] for x in r['intervals']),matched_before=len(r['suite_processes_before'].splitlines()),matched_after=len(r['suite_processes_after'].splitlines()),start_ns=r['start_ns'],end_ns=r['end_ns']))
excluded=[]
for p in sorted((root/'unqualified').glob('*/R-*/status.json')):
 excluded.append(dict(path='runtime/'+str(p.relative_to(root)),status=json.loads(p.read_text())))
(out/'qualification-summary.json').write_text(json.dumps(dict(test_only=True,receipts=receipts,excluded_attempts=excluded),indent=2)+'\n')
