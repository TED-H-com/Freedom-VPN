#!/usr/bin/env python3
"""TEST ONLY PATH wrapper: admission immediately before every nested command."""
import json
import os
from pathlib import Path
import sys
import time
sys.dont_write_bytecode=True
root=Path.cwd()
while not (root/'AGENTS.md').exists():
    if root.parent==root: raise RuntimeError('not in worktree')
    root=root.parent
sys.path.insert(0,str(root/'docs/poc/G2-followup30-evidence'))
import host
name=Path(sys.argv[0]).name
real=json.loads(os.environ['FVPN_F30_REAL_TOOLS'])[name]
if name in ('make','docker') or (name=='go' and sys.argv[1:2]==['test']):
    for attempt in range(10):
        row=host.hold()
        row.update(tool=name, args=[host.public(x) for x in sys.argv[1:]])
        with (host.DEST/'tool-admission.jsonl').open('a') as log:
            log.write(json.dumps(row)+'\n')
        if row['free']: break
        print('Waiting 60 seconds for host admission: '+name,file=sys.stderr,flush=True)
        time.sleep(60)
    else: raise RuntimeError('bounded host admission wait exhausted')
os.execve(real,[name,*sys.argv[1:]],dict(os.environ))
