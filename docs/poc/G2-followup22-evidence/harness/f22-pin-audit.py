from pathlib import Path
import json,hashlib,subprocess
manifests=[Path('docs/poc/G2-followup21-evidence/archive/archive.json'),Path('docs/poc/G2-followup22-evidence/inputs/archive.json')]
manifests+=sorted(Path('docs/poc/G2-followup22-evidence/sources').glob('*/archive.json'))
results=[]
for p in manifests:
 d=json.loads(p.read_text());count=0
 for f in d['files']:
  raw=(p.parent/f['path']).read_bytes()
  assert len(raw)==f['bytes'] and hashlib.sha256(raw).hexdigest()==f['sha256']
  original=subprocess.check_output(['git','show',d['source_commit']+':'+f['source_path']])
  if f.get('extraction'):
   original_lines=original.splitlines(keepends=True)
   extracted=raw.splitlines(keepends=True)
   indices=[original_lines.index(x) for x in extracted]
   assert indices==sorted(indices) and len(indices)==3
  else:assert raw==original
  count+=1
 results.append(dict(manifest=str(p),source_commit=d['source_commit'],files=count,byte_and_source_match=True))
fixture=json.loads(Path('.agent/f22-runtime/fixture-correction.json').read_text())
old=subprocess.check_output(['git','show','c6994eff40d9a98fffcf36b53dd3356f0360a31d:tests/netlab/scripts/f22_rollback.py'])
new=Path('tests/netlab/scripts/f22_rollback.py').read_bytes()
assert hashlib.sha256(old).hexdigest()==fixture['old_sha256']
assert hashlib.sha256(new).hexdigest()==fixture['new_sha256']
assert new==old.replace(b's.bind((addr, 40022))',b's.bind((addr, 40021))')
results.append(dict(fixture='f22_rollback.py',only_change='synthetic probe port 40022 to existing decoder format 40021',passed=True))
Path('docs/poc/G2-followup22-evidence/source-byte-audit.json').write_text(json.dumps(dict(test_only=True,results=results),indent=2)+'\n')
print(json.dumps(results,indent=2))
