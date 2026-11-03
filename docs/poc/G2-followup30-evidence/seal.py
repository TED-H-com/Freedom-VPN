#!/usr/bin/env python3
"""TEST ONLY seal this additive tree; never write earlier evidence."""
import gzip
import hashlib
import json
from pathlib import Path
import subprocess
import sys

D=Path(__file__).resolve().parent
subprocess.run([sys.executable,'-B',str(D/'publish.py')],check=True)
files=[]
for p in sorted(D.rglob('*')):
    if not p.is_file() or p.name=='retention-pins.json':continue
    data=p.read_bytes();row=dict(path=str(p.relative_to(D)),bytes=len(data),sha256=hashlib.sha256(data).hexdigest())
    if p.suffix=='.gz':
        raw=gzip.decompress(data);row.update(uncompressed_bytes=len(raw),uncompressed_sha256=hashlib.sha256(raw).hexdigest())
    files.append(row)
manifest=D/'retention-pins.json';manifest.write_text(json.dumps(dict(test_only=True,files=files),indent=2)+'\n')
print(json.dumps(dict(payload_files=len(files),manifest_sha256=hashlib.sha256(manifest.read_bytes()).hexdigest()),indent=2))
