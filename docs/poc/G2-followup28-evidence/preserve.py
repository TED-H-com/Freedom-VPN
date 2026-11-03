#!/usr/bin/env python3
"""TEST ONLY: verify D-098 and immutable diagnostic inputs, without writing them."""
import datetime
import gzip
import hashlib
import json
from pathlib import Path
import subprocess
import sys

DEST = Path(__file__).resolve().parent
ROOT = DEST.parents[2]
INITIAL = DEST / 'd098-initial.json'
BASE = json.loads(INITIAL.read_bytes())['base'] if INITIAL.exists() else subprocess.check_output(['git', 'rev-parse', 'main'], cwd=ROOT, text=True).strip()

def digest(data):
    return hashlib.sha256(data).hexdigest()

rows = []
for number in (25, 26, 27):
    tree = DEST.parent / f'G2-followup{number}-evidence'
    manifest = tree / 'retention-pins.json'
    assert manifest.read_bytes() == subprocess.check_output(['git', 'show', BASE + ':' + str(manifest.relative_to(ROOT))], cwd=ROOT)
    pins = json.loads(manifest.read_bytes())['files']
    assert {str(p.relative_to(tree)) for p in tree.rglob('*') if p.is_file()} == {r['path'] for r in pins} | {'retention-pins.json'}
    for pin in pins:
        data = (tree / pin['path']).read_bytes()
        assert len(data) == pin['bytes'] and digest(data) == pin['sha256'], pin['path']
        if pin['path'].endswith('.gz'):
            raw = gzip.decompress(data)
            assert len(raw) == pin['uncompressed_bytes'] and digest(raw) == pin['uncompressed_sha256']
    rows.append(dict(followup=number, files=len(pins), manifest_sha256=digest(manifest.read_bytes()), complete=True))
source = []
for path in ('tests/netlab/scripts/reload_analysis.py', 'tests/netlab/scripts/reload_analysis_test.py', 'tests/netlab/scripts/stream.py', 'node/enforce/apply.go'):
    data = (ROOT / path).read_bytes()
    assert data == subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT), path
    source.append(dict(path=path, sha256=digest(data)))
record = dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(), base=BASE, retained=rows, unchanged=source, gocache_sha256=digest(subprocess.check_output(['go', 'env', 'GOCACHE'], text=True).strip().encode()))
target = DEST / sys.argv[1]
assert not target.exists()
target.write_text(json.dumps(record, indent=2) + '\n')
print(json.dumps(record, indent=2))
