#!/usr/bin/env python3
"""TEST ONLY exactly one predeclared diagnostic test; no retries."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time
sys.dont_write_bytecode=True
import host

ROOT,DEST=host.ROOT,host.DEST
PREFIX='fvpn-w1-netlab-f28-window'
claim=DEST/'window/claim.json';claim.parent.mkdir(exist_ok=True)
record=dict(source_commit=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),
            main=subprocess.check_output(['git','rev-parse','main'],text=True).strip(),
            declaration_sha256=hashlib.sha256((DEST/'declaration.md').read_bytes()).hexdigest(),
            binary_sha256=hashlib.sha256((ROOT/'.agent/tmp/fvpn-w1-netlab-f28-fvpn-netlab').read_bytes()).hexdigest(),
            test='TestAT10_AT13_ReloadRevokeBaseline', attempts=1)
with claim.open('x') as f: json.dump(record,f,indent=2);f.write('\n')

def run(name,args):
    return subprocess.run([sys.executable,str(DEST/'run.py'),'window/'+name,*args],cwd=ROOT,check=False).returncode

status=1
try:
    up=run('up',['make','-f','mk/netlab.mk','-o','netlab-build','netlab-up','FVPN_LAB_PREFIX='+PREFIX,'NETLAB_BIN=.agent/tmp/fvpn-w1-netlab-f28-fvpn-netlab'])
    if up: raise RuntimeError('setup failed; no retry')
    host.admission('window/quiet',quiet=True)
    immediate=dict(hold=host.hold(),processes=host.processes())
    (DEST/'window/immediate.json').write_text(json.dumps(immediate,indent=2)+'\n')
    if not immediate['hold']['free'] or immediate['processes']['actual']:
        raise RuntimeError('immediate admission failed; no run')
    status=run('reload',['env','FVPN_NETLAB=1','FVPN_NETLAB_TCP_OBSERVE=1','FVPN_LAB_PREFIX='+PREFIX,
        'FVPN_NETLAB_EVIDENCE_DIR=docs/poc/G2-followup28-evidence/window/reload',
        'go','test','-tags','netlab','-count=1','-timeout','10m','-p','1','-parallel','1','-v','./tests/netlab','-run','^TestAT10_AT13_ReloadRevokeBaseline$'])
    after=dict(hold=host.hold(),processes=host.processes())
    (DEST/'window/after.json').write_text(json.dumps(after,indent=2)+'\n')
finally:
    run('down',['make','-f','mk/netlab.mk','netlab-down','FVPN_LAB_PREFIX='+PREFIX])
    run('remove-image',['docker','image','rm',PREFIX+':lab'])
print('Diagnostic exit:',status,flush=True)
sys.exit(status)
