#!/usr/bin/env python3
"""TEST ONLY preservation and source-identity audit; never edits old evidence."""
import datetime
import gzip
import hashlib
import json
from pathlib import Path
import re
import subprocess

DEST=Path(__file__).resolve().parent
ROOT=DEST.parents[2]
initial=json.loads((DEST/'d098-initial.json').read_bytes())
base=initial['base']

def digest(data):return hashlib.sha256(data).hexdigest()
def git(*args):return subprocess.check_output(['git',*args],cwd=ROOT)
def function(data,name):
    text=data.decode()
    start=re.search(r'^func '+re.escape(name)+r'\(',text,re.M).start()
    return text[start:text.index('\n}',start)+2].encode()

retained=[]
for pinned in initial['retained']:
    root=DEST.parent/('G2-followup'+str(pinned['followup'])+'-evidence')
    path=root/'retention-pins.json'
    assert digest(path.read_bytes())==pinned['manifest_sha256']
    rows=json.loads(path.read_bytes())['files']
    assert len(rows)==pinned['files']
    names={row['path'] for row in rows}
    assert {str(p.relative_to(root)) for p in root.rglob('*') if p.is_file()}==names|{'retention-pins.json'}
    for row in rows:
        data=(root/row['path']).read_bytes()
        assert len(data)==row['bytes'] and digest(data)==row['sha256'],row['path']
        if row['path'].endswith('.gz'):
            raw=gzip.decompress(data)
            assert len(raw)==row['uncompressed_bytes'] and digest(raw)==row['uncompressed_sha256']
    retained.append(pinned)

prior=[]
for row in git('ls-tree','-r',base,'--','docs/poc/').decode().splitlines():
    meta,path=row.split('\t',1)
    if path.startswith('docs/poc/G2-') and not path.startswith('docs/poc/G2-followup27-evidence/'):
        prior.append((path,meta.split()[2]))
actual=subprocess.check_output(['git','hash-object','--stdin-paths'],cwd=ROOT,
    input=('\n'.join(path for path,_ in prior)+'\n').encode()).decode().splitlines()
assert len(actual)==len(prior)
assert all(got==expected for got,(_,expected) in zip(actual,prior))

source=json.loads((DEST/'unchanged-contracts.json').read_bytes())
for row in source['checks']:
    current=(ROOT/row['path']).read_bytes()
    original=git('show',base+':'+row['path'])
    if 'function' in row:
        current=function(current,row['function']);original=function(original,row['function'])
    assert current==original and digest(current)==row['sha256'],row

cache=subprocess.check_output(['go','env','GOCACHE'],cwd=ROOT,text=True).strip()
record=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),base=base,retained=retained,
    earlier_G2_git_blobs_identical=len(prior),unchanged_source_checks=len(source['checks']),
    gocache_sha256=digest(cache.encode()),gocache_unchanged=digest(cache.encode())==initial['gocache_sha256'],
    go_version=subprocess.check_output(['go','version'],text=True).strip(),
    kernel=subprocess.check_output(['uname','-r'],text=True).strip(),
    machine=subprocess.check_output(['uname','-m'],text=True).strip())
assert record['gocache_unchanged']
(DEST/'d098-final.json').write_text(json.dumps(record,indent=2)+'\n')
print(json.dumps(record,indent=2))
