"""Read a published JSON document or its lossless gzip form."""
import gzip
import json
from pathlib import Path


def read_json(path):
    path=Path(path)
    if path.exists():
        return json.loads(path.read_bytes())
    return json.loads(gzip.decompress(Path(str(path)+'.gz').read_bytes()))
