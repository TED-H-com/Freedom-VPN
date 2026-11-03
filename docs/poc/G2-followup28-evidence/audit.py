#!/usr/bin/env python3
"""TEST ONLY offline publication audit; never edits evidence."""
import gzip
import hashlib
import json
from pathlib import Path
import re
import subprocess
from urllib.parse import unquote

DEST=Path(__file__).resolve().parent
ROOT=DEST.parents[2]
base=json.loads((DEST/'d098-initial.json').read_bytes())['base']
manifest=DEST/'retention-pins.json'
entries=json.loads(manifest.read_bytes())['files']
assert {str(p.relative_to(DEST)) for p in DEST.rglob('*') if p.is_file()} == {r['path'] for r in entries}|{'retention-pins.json'}
for row in entries:
    data=(DEST/row['path']).read_bytes()
    assert len(data)==row['bytes'] and hashlib.sha256(data).hexdigest()==row['sha256'],row['path']
    if row['path'].endswith('.gz'):
        data=gzip.decompress(data)
        assert len(data)==row['uncompressed_bytes'] and hashlib.sha256(data).hexdigest()==row['uncompressed_sha256']
    for bad in (b'/ho'+b'me/',b'BEGIN '+b'RSA PRIVATE KEY',b'BEGIN '+b'EC PRIVATE KEY',b'BEGIN '+b'OPENSSH PRIVATE KEY',b'sk_'+b'live_',b'gh'+b'p_',b'gh'+b'o_'):
        assert bad not in data,row['path']
    if row['path'].endswith('.md'):
        for link in re.findall(r'\]\(([^)]+)\)',data.decode()):
            if '://' in link or link.startswith('#'):continue
            target=unquote(link.split('#')[0])
            assert (DEST/row['path']).parent.joinpath(target).exists(),(row['path'],link)
for row in json.loads((DEST/'unchanged.json').read_bytes())['files']:
    data=(ROOT/row['path']).read_bytes()
    assert hashlib.sha256(data).hexdigest()==row['sha256']
    assert data==subprocess.check_output(['git','show',base+':'+row['path']],cwd=ROOT)
for tree in ('node/enforce','node/agent','node/updater','packaging/installer','tests/e2e','tests/netlab/agent','go.mod','go.sum'):
    assert not subprocess.check_output(['git','diff',base,'--',tree],cwd=ROOT),tree
print('Complete manifest:',len(entries),'payload files; SHA-256:',hashlib.sha256(manifest.read_bytes()).hexdigest())
print('Gzip originals, public patterns, local Markdown links, original UDP/D-120/production/consumer sources: PASS')
