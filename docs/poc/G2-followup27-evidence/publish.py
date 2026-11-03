#!/usr/bin/env python3
"""TEST ONLY lossless publication of this follow-up's evidence tree only."""
import gzip
import hashlib
import json
from pathlib import Path
import shutil

ROOT=Path(__file__).resolve().parent
mapping=ROOT/'publication.json'
rows=json.loads(mapping.read_text())['files'] if mapping.exists() else []

def digest(data):return hashlib.sha256(data).hexdigest()
def relative(p):return str(p.relative_to(ROOT))

cache=ROOT/'__pycache__'
if cache.exists():shutil.rmtree(cache)
for p in sorted(ROOT.rglob('*')):
    if not p.is_file() or p.name=='retention-pins.json':continue
    data=p.read_bytes()
    if p.suffix=='.json':
        try:json.loads(data)
        except (ValueError,UnicodeError):
            # This pre-existing producer returns Go struct text under JSON.
            assert p.name=='counters.json' and data.startswith(b'{Counters:map['),str(p)
            target=p.with_suffix('.txt');assert not target.exists()
            p.rename(target)
            rows.append(dict(original=relative(p),published=relative(target),sha256=digest(data),bytes=len(data),reason='producer Go struct dump; exact bytes, correct suffix'))
            p=target
    if p.suffix in ('.json','.jsonl') and len(data)>1_000_000:
        target=Path(str(p)+'.gz');assert not target.exists()
        zipped=gzip.compress(data,mtime=0)
        target.write_bytes(zipped)
        assert gzip.decompress(zipped)==data
        rows.append(dict(original=relative(p),published=relative(target),sha256=digest(zipped),bytes=len(zipped),uncompressed_sha256=digest(data),uncompressed_bytes=len(data),reason='lossless large JSON/JSONL'))
        p.unlink()
    elif p.suffix in ('.txt','.nft'):
        text=data.decode()
        normalized='\n'.join(line.rstrip() for line in text.splitlines())+'\n'
        if data and normalized.encode()!=data:
            target=Path(str(p)+'.raw.gz');assert not target.exists()
            zipped=gzip.compress(data,mtime=0);target.write_bytes(zipped)
            p.write_text(normalized)
            rows.append(dict(original=relative(p),published=relative(target),sha256=digest(zipped),bytes=len(zipped),uncompressed_sha256=digest(data),uncompressed_bytes=len(data),reason='raw bytes preserved; readable text trailing whitespace normalized'))
mapping.write_text(json.dumps(dict(test_only=True,files=rows),indent=2)+'\n')
print('Publication mappings:',len(rows))
