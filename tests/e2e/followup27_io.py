"""Read original Follow-up 27 bytes, including pinned lossless archives."""
import gzip
import hashlib
import json

INDEX = '<!-- e2e-followup27: document-index -->'


def receipt(raw):
    return dict(bytes=str(len(raw)), sha256=hashlib.sha256(raw).hexdigest())


def read(path):
    if path.exists() and path.suffix != '.gz':
        raw = path.read_bytes()
        assert not raw.startswith(INDEX.encode()), 'document index; read the linked .gz capture'
        return raw
    archived = path if path.suffix == '.gz' else path.with_name(path.name+'.gz')
    root = next(parent for parent in archived.parents if (parent/'archive.json').exists())
    record = json.loads((root/'archive.json').read_text())
    entries = json.loads(next(o['detail'] for o in record['observations'] if o['name']=='analysis'))['files']
    entry = next(e for e in entries if e['archive']==archived.relative_to(root).as_posix())
    compressed = archived.read_bytes()
    assert receipt(compressed)==entry['compressed'], 'compressed receipt mismatch'
    raw = gzip.decompress(compressed)
    assert receipt(raw)==entry['original'], 'original receipt mismatch'
    return raw
