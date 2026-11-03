#!/usr/bin/env python3
"""TEST ONLY foreground receipt, disk scratch, and nested host admission."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time
sys.dont_write_bytecode=True
import host

name,*cmd=sys.argv[1:]
dest=host.DEST
log=dest/(name+'.txt'); receipt=dest/(name+'.json')
log.parent.mkdir(parents=True,exist_ok=True)
assert not log.exists() and not receipt.exists(),'never overwrite an attempt'
env=dict(os.environ,PYTHONDONTWRITEBYTECODE='1')
if 'FVPN_F30_REAL_TOOLS' not in env:
    bindir=host.ROOT/'.agent/tmp/fvpn-w1-netlab-f30-tools'; bindir.mkdir(parents=True,exist_ok=True)
    env['FVPN_F30_REAL_TOOLS']=json.dumps({n:shutil.which(n) for n in ('go','make','docker')})
    for n in ('go','make','docker'):
        shutil.copyfile(dest/'admit-tool.py',bindir/n); (bindir/n).chmod(0o755)
    env['PATH']=str(bindir)+os.pathsep+env['PATH']
start=datetime.datetime.now(datetime.timezone.utc).isoformat(); at=time.monotonic()
p=subprocess.run([sys.executable,'tests/netlab/scripts/disktmp.py',*cmd],cwd=host.ROOT,env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,check=False)
body=host.public(p.stdout.decode(errors='replace'))
log.write_text('\n'.join(line.rstrip() for line in body.splitlines()).rstrip()+'\n' if body else '')
receipt.write_text(json.dumps(dict(start_utc=start,elapsed_seconds=time.monotonic()-at,command=[host.public(x) for x in cmd],exit_code=p.returncode,output_sha256=hashlib.sha256(log.read_bytes()).hexdigest()),indent=2)+'\n')
print(body[-7000:]);print('Receipt:',receipt.relative_to(host.ROOT),'exit:',p.returncode)
sys.exit(p.returncode)
