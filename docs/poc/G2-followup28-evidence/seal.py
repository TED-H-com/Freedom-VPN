#!/usr/bin/env python3
"""TEST ONLY: seal this evidence tree, including both forms of gzip artifacts."""
import gzip
import hashlib
import json
from pathlib import Path

ROOT=Path(__file__).resolve().parent
rows=[]
for path in sorted(ROOT.rglob('*')):
    if not path.is_file() or path.name=='retention-pins.json':continue
    assert '__pycache__' not in path.parts
    data=path.read_bytes()
    row=dict(path=str(path.relative_to(ROOT)),bytes=len(data),sha256=hashlib.sha256(data).hexdigest())
    name=path.name
    if name.endswith('.gz'):
        data=gzip.decompress(data)
        row.update(uncompressed_bytes=len(data),uncompressed_sha256=hashlib.sha256(data).hexdigest())
        name=name[:-3]
    else:
        assert not (path.suffix in ('.json','.jsonl') and len(data)>1_000_000),path
    if name.endswith('.json'):json.loads(data)
    if name.endswith('.jsonl'):
        for line in data.splitlines():
            if line.strip():json.loads(line)
    rows.append(row)
raw=(json.dumps(dict(test_only=True,files=rows),indent=2)+'\n').encode()
(ROOT/'retention-pins.json').write_bytes(raw)
print('files:',len(rows),'manifest_sha256:',hashlib.sha256(raw).hexdigest(),
      'total_bytes:',sum(row['bytes'] for row in rows)+len(raw),
      'gzip_files:',sum(row['path'].endswith('.gz') for row in rows))
