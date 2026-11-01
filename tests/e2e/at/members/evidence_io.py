"""Lossless new evidence; history is immutable. Both forms have exact byte pins."""
from pathlib import Path
import gzip
import hashlib
import json

THRESHOLD = 20_000_000

def read(path):
    path = Path(path)
    if not path.exists() and path.suffix != '.gz':
        path = Path(str(path) + '.gz')
    raw = path.read_bytes()
    if path.suffix != '.gz':
        return raw
    pin = json.loads(Path(str(path) + '.sha256.json').read_text())
    assert hashlib.sha256(raw).hexdigest() == pin['compressed_sha256']
    assert str(len(raw)) == pin['compressed_bytes']
    plain = gzip.decompress(raw)
    assert hashlib.sha256(plain).hexdigest() == pin['uncompressed_sha256']
    assert str(len(plain)) == pin['uncompressed_bytes']
    return plain

def load(path):
    return json.loads(read(path))

def publish(path):
    path = Path(path)
    raw = path.read_bytes()
    if len(raw) <= THRESHOLD:
        return
    compressed = gzip.compress(raw, mtime=0)
    dest = Path(str(path) + '.gz')
    dest.write_bytes(compressed)
    pin = dict(compressed_sha256=hashlib.sha256(compressed).hexdigest(),
               uncompressed_sha256=hashlib.sha256(raw).hexdigest(),
               compressed_bytes=str(len(compressed)), uncompressed_bytes=str(len(raw)))
    Path(str(dest) + '.sha256.json').write_text(json.dumps(pin, indent=2) + '\n')
    path.unlink()
