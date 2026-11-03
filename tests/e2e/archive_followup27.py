#!/usr/bin/env python3
"""Losslessly archive completed Follow-up 27 native captures, once per run."""
import argparse
import gzip
import json
import re
from pathlib import Path
from analyze_kernel_capture import publish
from followup27_io import INDEX, receipt


def archive(run):
    assert run.parent.name=='followup27', 'only new Follow-up 27 run paths'
    assert not (run/'archive.json').exists(), 'immutable archive already exists'
    source = json.loads((run/'measurement.json').read_text())
    files = []
    for path in sorted(run.rglob('*')):
        if not path.is_file() or path.suffix not in ('.json','.jsonl','.md') or path.stat().st_size<1024*1024:
            continue
        raw = path.read_bytes()
        compressed = gzip.compress(raw,compresslevel=9,mtime=0)
        assert gzip.decompress(compressed)==raw
        target = path.with_name(path.name+'.gz')
        with target.open('xb') as out:out.write(compressed)
        entry=dict(path=path.relative_to(run).as_posix(),archive=target.relative_to(run).as_posix(),original=receipt(raw),compressed=receipt(compressed))
        if path.suffix=='.md':
            entry['status']=re.search(r'^Status: (pass|blocked|fail)$',raw.decode(),re.M).group(1)
        files.append(entry)
    publish(run/'archive.json',source,'archive','Lossless capture archive receipts',dict(files=files,scope='Original bytes retained in gzip. Plain Markdown paths are navigation documents, not replacement captures.'))
    # Receipts are durable before replacing any source path. Keep every Markdown
    # target readable; no dangling Markdown links, including future hand-offs.
    for entry in files:
        path = run/entry['path']
        if path.suffix=='.md':
            path.write_text(f"{INDEX}\n\n# {path.stem}\n\nRun: `{source['run_id']}`. Original status: **{entry['status']}**.\n\n"
                f"[Original Markdown capture]({path.name}.gz) · [Archive receipts](archive.json)\n\n"
                "This plain file is a navigation document, not a capture. The gzip retains every original byte. "
                f"Uncompressed bytes: {entry['original']['bytes']}; SHA-256: `{entry['original']['sha256']}`.\n\n"
                f"Read with `gzip -dc {path.name}.gz`. No measurement or acceptance status was changed.\n")
        else:path.unlink()
    return files


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('run',type=Path)
    args=parser.parse_args()
    print(json.dumps(dict(files=len(archive(args.run)))))
