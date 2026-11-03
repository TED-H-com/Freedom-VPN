#!/usr/bin/env python3
"""TEST ONLY complete-tree pin, public hygiene and local-link audit."""
import gzip
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

D=Path(__file__).resolve().parent;root=D.parents[2]
m=json.loads((D/'retention-pins.json').read_text())
actual={str(p.relative_to(D)) for p in D.rglob('*') if p.is_file()}
assert actual=={r['path'] for r in m['files']}|{'retention-pins.json'}
for row in m['files']:
    p=D/row['path'];data=p.read_bytes()
    assert len(data)==row['bytes'] and hashlib.sha256(data).hexdigest()==row['sha256'],row['path']
    if p.suffix=='.gz':
        data=gzip.decompress(data)
        assert len(data)==row['uncompressed_bytes'] and hashlib.sha256(data).hexdigest()==row['uncompressed_sha256']
    text=data.decode()
    for bad in ('/ho'+'me/',os.environ['FVPN_HOLD_SH'],'BEGIN '+'RSA PRIVATE KEY','BEGIN '+'EC PRIVATE KEY','BEGIN '+'OPENSSH PRIVATE KEY','sk_'+'live_','gh'+'p_','gh'+'o_'):
        assert bad not in text,row['path']
    name=p.name.removesuffix('.gz')
    if name.endswith('.json'):json.loads(text)
    if name.endswith('.jsonl'):
        for line in text.splitlines():json.loads(line)
    if p.suffix=='.md':
        for link in re.findall(r'\]\(([^)]+)\)',text):
            link=link.split('#',1)[0]
            if not link or '://' in link:continue
            assert (p.parent/link).exists(),(row['path'],link)
base=json.loads((D/'d098-initial.json').read_text())['base']
assert not subprocess.check_output(['git','diff',base,'--','node/enforce','node/agent','node/updater','packaging/installer','tests/e2e'],cwd=root)
for path in ('tests/netlab/scripts/reload_analysis.py','tests/netlab/scripts/reload_analysis_test.py','tests/netlab/scripts/stream.py','tests/netlab/scenario_at10_test.go'):
    assert (root/path).read_bytes()==subprocess.check_output(['git','show',base+':'+path],cwd=root)
print('PASS: complete pins, gzip originals, public hygiene, Markdown links, production/consumers and D-120 unchanged')
