#!/usr/bin/env python3
"""TEST ONLY redact residual home-relative locations from admission metadata."""
import hashlib
import json
from pathlib import Path
import re
D=Path(__file__).resolve().parent
manifest=D/'location-publication.json'
changes=json.loads(manifest.read_text())['files'] if manifest.exists() else []
pattern=re.compile(r'<home>/[^\s\"\'<>]+')

def scrub(value):
    if isinstance(value,str):
        return pattern.sub(lambda m:'go' if m.group().endswith('/bin/go') else '<private-location>',value)
    if isinstance(value,list):return [scrub(x) for x in value]
    if isinstance(value,dict):return {k:scrub(v) for k,v in value.items()}
    return value

for p in sorted(D.rglob('*')):
    if not p.is_file() or p==manifest or p.suffix not in ('.json','.jsonl','.txt'):continue
    before=p.read_bytes()
    if b'<home>/' not in before:continue
    if p.suffix=='.json':after=(json.dumps(scrub(json.loads(before)),indent=2)+'\n').encode()
    elif p.suffix=='.jsonl':after=''.join(json.dumps(scrub(json.loads(line)))+'\n' for line in before.splitlines()).encode()
    else:after=scrub(before.decode()).encode()
    if after==before:continue
    p.write_bytes(after)
    changes.append(dict(path=str(p.relative_to(D)),before_sha256=hashlib.sha256(before).hexdigest(),after_sha256=hashlib.sha256(after).hexdigest(),reason='Residual home-relative tool or other-lane location redacted; CPU, timing, packet and verdict values unchanged. Private text is not republished.'))
manifest.write_text(json.dumps(dict(test_only=True,files=changes),indent=2)+'\n')
print('location redactions:',len(changes))
