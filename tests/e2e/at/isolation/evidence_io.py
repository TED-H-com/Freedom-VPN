"""Read a published JSON document or its lossless gzip form."""
import gzip
import json
from pathlib import Path
import re

_HOME = '/' + 'home/' + r'[^\s"\']+'
_SK = 'sk_' + 'live_' + r'[A-Za-z0-9_]+'
_GHP = 'gh' + 'p_' + r'[A-Za-z0-9_]+'
_GHO = 'gh' + 'o_' + r'[A-Za-z0-9_]+'
_PK = '-----BEGIN ' + r'[A-Z0-9 ]*PRIVATE' + ' KEY-----.*?-----END ' + r'[A-Z0-9 ]*PRIVATE' + ' KEY-----'

_PATTERNS = (
    (re.compile(r'(?i)"(token|setup_key|password|secret|authorization|seed_b64|api_key)"\s*:\s*"[^"]*"'), r'"\1": "[redacted]"'),
    (re.compile(r'(?i)\b(token|setup_key|password|secret|authorization|seed_b64|api_key)=[^\s"\']+'), r'\1=[redacted]'),
    (re.compile(_HOME), '[redacted]'),
    (re.compile(r'(?i)postgres://[^\s"\']+'), '[redacted]'),
    (re.compile(r'(?i)bearer\s+[^\s"\']+'), '[redacted]'),
    (re.compile(_SK), '[redacted]'),
    (re.compile(_GHP), '[redacted]'),
    (re.compile(_GHO), '[redacted]'),
    (re.compile(_PK, re.DOTALL), '[redacted]'),
)


def redact_text(value):
    out = str(value)
    for pattern, repl in _PATTERNS:
        out = pattern.sub(repl, out)
    return out


def redact_error(error):
    if isinstance(error, OSError):
        name = Path(str(error.filename)).name if getattr(error, 'filename', None) else ''
        detail = error.strerror or 'os error'
        msg = f'{detail}: {name}' if name else detail
        return f'{type(error).__name__}: {redact_text(msg)}'
    return f'{type(error).__name__}: {redact_text(error)}'


def read_json(path):
    path=Path(path)
    if path.suffix == ".gz":
        return json.loads(gzip.decompress(path.read_bytes()))
    if path.exists():
        return json.loads(path.read_bytes())
    return json.loads(gzip.decompress(Path(str(path)+'.gz').read_bytes()))
