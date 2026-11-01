from pathlib import Path
import json
root=Path('.agent/f22-runtime')
evidence=Path('docs/poc/G2-followup22-evidence')
s=json.loads((evidence/'comparison-summary.json').read_text())
lines=['| Population | Applies / guards / failed | Outer p50 / p90 / p99 / max (ms) | Install p50 / p90 / p99 / max (ms) | Render→installed p50 / p90 / p99 / max (ms) |','|---|---:|---|---|---|']
for p in s['populations']:
 def vals(k):
  v=p['percentiles_ms'][k]
  return ' / '.join(f'{v[x]:.6f}' for x in ['p50','p90','p99','max']) if v else 'unavailable'
 lines.append(f"| {p['population']} | {p['applies']} / {p['completed_guards']} / {p['failed_applies']} | {vals('outer_ns')} | {vals('install_ns')} | {vals('render_to_installed_ns')} |")
(evidence/'distribution-table.md').write_text('\n'.join(lines)+'\n')
rows=[]
for p in sorted(root.glob('*-postflight.json')):
 name=p.stem.removesuffix('-postflight')
 q=json.loads(p.read_text())
 matching=[]
 for stage in ['preflight','ready']:
  files=sorted(root.glob(name+'-'+stage+'-*.json'),key=lambda x:int(x.stem.rsplit('-',1)[1]))
  accepted=[x for x in files if json.loads(x.read_text())['quiet']]
  if accepted:matching.append(accepted[-1])
 matching.append(p)
 samples=[]
 for f in matching:
  r=json.loads(f.read_text())
  samples.append(dict(path=f.name,start_ns=r['start_ns'],end_ns=r['end_ns'],max_r=max(x['r'] for x in r['intervals']),min_idle=min(x['id'] for x in r['intervals']),load_before=r['load_before'],load_after=r['load_after'],d_state_before=r['d_state_before'],d_state_after=r['d_state_after'],quiet=r['quiet'],matching_processes_before=len(r['suite_processes_before'].splitlines()),matching_processes_after=len(r['suite_processes_after'].splitlines())))
 status=root/name/'status.json'
 rows.append(dict(population=name,status=json.loads(status.read_text()) if status.exists() else None,host_samples=samples))
(evidence/'host-summary.json').write_text(json.dumps(dict(test_only=True,rows=rows),indent=2)+'\n')
print('\n'.join(lines))
failures=[]
for p in sorted((evidence/'analysis').glob('*.json')):
 d=json.loads(p.read_text())
 for f in d['failures']:
  r=f['record'];body=r.get('result',r)
  failures.append(dict(population=d['population'],analysis_path=str(p.relative_to(evidence)),row_index=f['index'],run_id=body.get('run_id'),boot_id=body.get('boot_id'),sequence=body.get('sequence'),failure_stage=f['failure_stage'],cause_status=f['cause_status'],apply_bracket=f['apply_bracket'],deadline=body['timing']['deadline']))
(evidence/'failed-apply-index.json').write_text(json.dumps(dict(test_only=True,cause_status_scope='install overrun only; not_applicable does not explain other failures',rows=failures),indent=2)+'\n')
