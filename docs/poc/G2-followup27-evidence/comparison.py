#!/usr/bin/env python3
"""TEST ONLY. Run the predeclared quartet once; never retry a leg."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import time
import host

ROOT=host.ROOT
DEST=host.DEST
plan=json.loads((DEST/'comparison-plan.json').read_text())

def command(name,args):
    return subprocess.run([sys.executable,str(DEST/'run.py'),name,*args],cwd=ROOT,check=False).returncode

def exclusive(name):
    rows=[];start=time.monotonic();path=DEST/(name+'.json')
    assert not path.exists()
    while True:
        snapshot=host.processes();rows.append(snapshot)
        clear=not snapshot['timing_sensitive']
        path.parent.mkdir(parents=True,exist_ok=True)
        path.write_text(json.dumps(dict(wait_seconds=time.monotonic()-start,admitted=clear,rows=rows),indent=2)+'\n')
        if clear:return
        print(name+': waiting for another timing-sensitive run',flush=True)
        time.sleep(60)

for leg in plan['legs']:
    label=leg['label']; prefix='fvpn-w1-netlab-f27-'+label.lower()
    binary=plan['binaries'][leg['arm']]
    assert hashlib.sha256((ROOT/binary['path']).read_bytes()).hexdigest()==binary['sha256']
    print('Starting setup for '+label,flush=True)
    exclusive('comparison/'+label+'/pre-setup')
    args=['make','-f','mk/netlab.mk','-o','netlab-build','netlab-up','FVPN_LAB_PREFIX='+prefix,'NETLAB_BIN='+binary['path']]
    try:
        up=command('comparison/'+label+'/up',args)
        if up:
            print(label+': setup failed, no retry',flush=True)
            continue
        host.admission('comparison/'+label+'/admission',timing=True)
        exclusive('comparison/'+label+'/start-processes')
        env=['env','FVPN_NETLAB=1','FVPN_LAB_PREFIX='+prefix,'FVPN_NETLAB_F27_COST=1']
        test=['go','test','-tags','netlab','-count=1','-timeout','10m','-p','1','-parallel','1','-v','./tests/netlab','-run']
        for phase,pattern in [('cost','^TestISO01_F27_VerifyCost$'),('reload','^TestAT10_AT13_ReloadRevokeBaseline$')]:
            # Check between phases too; neither shell text nor a queued wrapper
            # is mistaken for an actual other-lane suite.
            exclusive('comparison/'+label+'/'+phase+'-processes')
            command('comparison/'+label+'/'+phase,env+['FVPN_NETLAB_EVIDENCE_DIR=docs/poc/G2-followup27-evidence/comparison/'+label+'/'+phase]+test+[pattern])
        (DEST/'comparison'/label/'after-host.json').write_text(json.dumps(host.sample(),indent=2)+'\n')
    finally:
        command('comparison/'+label+'/down',['make','-f','mk/netlab.mk','netlab-down','FVPN_LAB_PREFIX='+prefix])
    print('Finished '+label,flush=True)
