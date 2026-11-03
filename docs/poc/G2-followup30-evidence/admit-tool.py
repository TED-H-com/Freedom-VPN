#!/usr/bin/env python3
"""TEST ONLY PATH wrapper: admission immediately before every nested command."""
import json
import os
from pathlib import Path
import sys
import subprocess
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
if name=='go' and sys.argv[1:2]==['test'] and os.environ.get('FVPN_NETLAB')=='1' and os.environ.get('FVPN_F30_RUN'):
    target=host.DEST/(os.environ['FVPN_F30_RUN']+'-quiet.jsonl')
    entered=time.monotonic()
    for attempt in range(120):
        time.sleep(1)
        rows=[json.loads(line) for line in target.read_text().splitlines()]
        last=rows[-1]
        if last.get('quiet') and last.get('monotonic_seconds',0)>entered and time.monotonic()-last['monotonic_seconds']<=5:
            break
    else:raise RuntimeError('post-startup quiet check not met; no live test launch')
    binary=subprocess.check_output(['docker','exec',os.environ['FVPN_LAB_PREFIX'],'sha256sum','/usr/local/bin/fvpn-netlab'],text=True).split()[0]
    (host.DEST/(os.environ['FVPN_F30_RUN']+'-binary.json')).write_text(json.dumps(dict(sha256=binary,quiet_check_utc=last['utc'],wait_seconds=time.monotonic()-entered),indent=2)+'\n')
    if not host.hold()['free']:raise RuntimeError('not admitted after quiet check')
if name=='docker' and '/opt/netlab/scripts/guard_observe.py' in sys.argv:
    start=dict(wall_ns=str(time.time_ns()),mono_ns=str(time.monotonic_ns()))
    code=subprocess.run([real,*sys.argv[1:]]).returncode
    end=dict(wall_ns=str(time.time_ns()),mono_ns=str(time.monotonic_ns()))
    with (host.DEST/'guard-command-cost.jsonl').open('a') as log:
        log.write(json.dumps(dict(command=[host.public(x) for x in sys.argv[1:]],start=start,end=end,exit_code=code))+'\n')
    sys.exit(code)
os.execve(real,[name,*sys.argv[1:]],dict(os.environ))
