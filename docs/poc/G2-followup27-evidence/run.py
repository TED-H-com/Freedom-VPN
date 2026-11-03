#!/usr/bin/env python3
"""TEST ONLY: foreground command receipt; invoke through disktmp.py."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

root = Path(__file__).resolve().parents[3]
dest = root / 'docs/poc/G2-followup27-evidence'
name, *cmd = sys.argv[1:]
log = dest / (name + '.txt')
receipt = dest / (name + '.json')
log.parent.mkdir(parents=True, exist_ok=True)
assert not log.exists() and not receipt.exists(), 'never overwrite an attempt'
start = datetime.datetime.now(datetime.timezone.utc).isoformat()
at = time.monotonic()
p = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False)
# Host paths are not public evidence. Keep output line contents, normalizing
# trailing whitespace so the public patch also passes git diff --check.
body = p.stdout.decode(errors='replace').replace(str(root), '<worktree>')
body = body.replace(str(Path.home()), '<home>')
log.write_text('\n'.join(line.rstrip() for line in body.splitlines()) + '\n')
receipt.write_text(json.dumps(dict(start_utc=start, elapsed_seconds=time.monotonic()-at,
    command=cmd, exit_code=p.returncode, output_sha256=hashlib.sha256(log.read_bytes()).hexdigest()), indent=2)+'\n')
print(body[-6000:])
print('receipt:', receipt.relative_to(root), 'exit:', p.returncode)
sys.exit(p.returncode)
