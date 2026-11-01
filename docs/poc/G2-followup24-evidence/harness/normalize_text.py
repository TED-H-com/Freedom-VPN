#!/usr/bin/env python3
"""TEST ONLY: retain exact raw text bytes; trim presentation whitespace."""
import gzip
import hashlib
import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parent.parent


def main():
    receipt = ROOT / "text-normalization.json"
    if receipt.exists():
        raise RuntimeError("normalization already recorded; do not overwrite it")
    files = []
    for path in sorted(ROOT.rglob("*.txt")):
        raw = path.read_bytes()
        normalized = b"\n".join(line.rstrip(b" \t\r") for line in raw.splitlines()).rstrip(b"\n")
        if normalized:
            normalized += b"\n"
        if normalized == raw:
            continue
        archive = path.with_name(path.name + ".raw.gz")
        if archive.exists():
            raise RuntimeError("refusing to overwrite raw archive")
        archive.write_bytes(gzip.compress(raw, mtime=0))
        assert gzip.decompress(archive.read_bytes()) == raw
        path.write_bytes(normalized)
        files.append(dict(path=str(path.relative_to(ROOT)),
                          raw_archive=str(archive.relative_to(ROOT)),
                          raw_bytes=len(raw), raw_sha256=hashlib.sha256(raw).hexdigest(),
                          readable_bytes=len(normalized),
                          readable_sha256=hashlib.sha256(normalized).hexdigest()))
    receipt.write_text(json.dumps(dict(
        test_only=True,
        note="Only trailing line whitespace and terminal blank lines were trimmed from readable text. Exact original public bytes are retained in raw.gz; earlier output digests refer to those decompressed bytes. No timing, verdict or observation was changed.",
        files=files), indent=2) + "\n")
    print("Raw text archives and readable copies:", len(files))


if __name__ == "__main__":
    main()
