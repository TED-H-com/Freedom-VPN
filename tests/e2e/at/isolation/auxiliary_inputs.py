"""Pin actual same-invocation inputs, including compressed filenames. No AT edits."""
import gzip
import hashlib
import json
from pathlib import Path
from evidence_io import read_json

NAMES = ('AT13.json', 'clone-runtime.json', 'identity-freeze.json',
         'receiver-closure.json', 'container-lifecycle-events.json',
         'restart-lifecycle.json', 'populated-holds.json', 'receiver-capture.json')


def seal(root, invocation):
    rows = []
    for name in NAMES:
        path = root/name
        if not path.exists():
            path = root/(name+'.gz')
        row = dict(logical_name=name, status='unavailable')
        if path.exists():
            # Same lossless gzip convention as the existing evidence readers.
            if name in ('clone-runtime.json','identity-freeze.json','receiver-capture.json','populated-holds.json') and path.suffix != '.gz' and path.stat().st_size > 1 << 20:
                raw = path.read_bytes()
                compressed = Path(str(path)+'.gz')
                compressed.write_bytes(gzip.compress(raw, mtime=0))
                path.unlink()
                path = compressed
            raw = path.read_bytes()
            row.update(status='available', path=path.name, bytes=len(raw),
                       sha256=hashlib.sha256(raw).hexdigest())
        rows.append(row)
    result = dict(invocation, files=rows, phase='after lifecycle finalization',
                  provenance='assembled runner; original inputs, no historical replacement')
    (root/'auxiliary-inputs.json').write_text(json.dumps(result, indent=2)+'\n')
    return result


def inputs(root, expected_run=None):
    manifest = read_json(root/'auxiliary-inputs.json')
    if not manifest.get('run_id') or (expected_run and manifest['run_id'] != expected_run):
        raise ValueError('input manifest run mismatch')
    out = {}
    for row in manifest['files']:
        if row['status'] != 'available':
            continue
        path = root/row['path']
        if path.name != row['path']:
            raise ValueError('input path not a basename')
        raw = path.read_bytes()
        if len(raw) != row['bytes'] or hashlib.sha256(raw).hexdigest() != row['sha256']:
            raise ValueError('input digest mismatch: '+row['logical_name'])
        out[row['logical_name']] = read_json(path)
    for name in ('AT13.json', 'clone-runtime.json', 'identity-freeze.json', 'container-lifecycle-events.json', 'receiver-capture.json'):
        value = out.get(name)
        if value is not None and value.get('run_id') != manifest['run_id']:
            raise ValueError('input run mismatch: '+name)
    for name in ('AT13.json', 'clone-runtime.json', 'identity-freeze.json'):
        value = out.get(name)
        if value is not None and (value.get('build') != manifest['build'] or value.get('source_dirty') != manifest['source_dirty']):
            raise ValueError('input build mismatch: '+name)
    for row in out.get('receiver-closure.json', []):
        if row.get('run_id') != manifest['run_id'] or row.get('build') != manifest['build'] or row.get('source_dirty') != manifest['source_dirty']:
            raise ValueError('closure provenance mismatch')
    return manifest, out
