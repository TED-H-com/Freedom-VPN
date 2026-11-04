#!/usr/bin/env python3
"""D-098: freeze root records before any Follow-up 18 record replacement."""
import hashlib
import json
import shutil
import subprocess

from run import ROOT, PUBLIC, utc, write_json

ARCHIVE = PUBLIC / 'history/before-follow-up-18'


def main():
    if ARCHIVE.exists():
        raise SystemExit('archive already exists; never overwrite history')
    source = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    files = sorted(p for p in PUBLIC.iterdir() if p.is_file())
    # Require exactly the committed bytes, not unreviewed edits or placeholders.
    for path in files:
        original = subprocess.check_output(['git', 'show', source+':'+str(path.relative_to(ROOT))], cwd=ROOT)
        if path.read_bytes() != original:
            raise SystemExit('root record differs from committed source: '+path.name)
    ARCHIVE.mkdir()
    pins = []
    for path in files:
        target = ARCHIVE / path.name
        shutil.copyfile(path, target)
        assert target.read_bytes() == path.read_bytes()
        pins.append(dict(file=path.name, sha256=hashlib.sha256(path.read_bytes()).hexdigest()))
    manifest = dict(source_revision=source, archived_at=utc(), snapshot_files=pins,
                    reason='D-098 before Follow-up 18 AT29 transport run; root bytes retained exactly')
    write_json(ARCHIVE / 'archive-index.json', manifest)
    index = json.loads((PUBLIC / 'history/index.json').read_text())
    index['before_follow_up_18'] = manifest
    write_json(PUBLIC / 'history/index.json', index)
    print(f'PASS: {len(pins)} root records archived byte-identically with SHA-256 pins')


if __name__ == '__main__':
    main()
