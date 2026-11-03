#!/usr/bin/env python3
"""TEST ONLY required gates, foreground and in the requested order."""
import os
from pathlib import Path
import shutil
import subprocess
import sys
sys.dont_write_bytecode=True
import host

ROOT=host.ROOT
DEST=host.DEST
steps=[
 ('gate1',['make','-f','mk/netlab.mk','netlab-lint','netlab-unit','netlab-integration','netlab-build','FVPN_LAB_PREFIX=fvpn-w1-netlab-f27']),
 ('gate2',['make','-f','mk/netlab.mk','netlab-test','FVPN_LAB_PREFIX=fvpn-w1-netlab-f27']),
 ('gate3-agent',['go','test','-count=1','./node/agent/...','./node/updater/...','./packaging/installer/...']),
 ('gate3-reviews',['go','test','./node/agent','./node/enforce','-run','^TestReviewR1[6-9]_','-count=1']),
 ('gate3-consumers',['go','test','-count=1','./tests/e2e/at/usage/...','./tests/e2e/internal/counterread/...']),
 ('gate4',['go','test','-count=1','./tests/netlab','-run','^TestCommittedEvidenceFiles_ValidAndHygieneClean$']),
 ('gate5',['go','test','-count=1','./tests/handover/...']),
 ('gate6',['make','go-lint','go-with-disk-tmp=set -eu;']),
 ('gate7',['go','vet','./...']),
 ('gate8',['bash','-c','test -z "$(gofmt -l $(git ls-files \'*.go\'))"']),
 ('gate9',['git','diff','--check']),
]
# No shell trap: disktmp.py owns scratch lifetime around this whole runner,
# and each netlab target retains its own nested disk wrapper.
remaining=steps
if len(sys.argv)>1:
    # Explicit continuation after an out-of-scope failed gate: never rerun or
    # overwrite that attempt. The failed receipt remains the gate's verdict.
    assert len(sys.argv)==3 and sys.argv[1]=='--after'
    index=[name for name,_ in steps].index(sys.argv[2])
    remaining=steps[index+1:]
for name,cmd in remaining:
    print('Starting '+name,flush=True)
    env=dict(os.environ,PYTHONDONTWRITEBYTECODE='1')
    if name=='gate4':
        subprocess.run([sys.executable,str(DEST/'publish.py')],check=True)
        subprocess.run(['git','add','docs/poc/G2-followup27-evidence'],cwd=ROOT,check=True)
    if name=='gate2':
        host.admission('admission/gate2-before')
        bindir=ROOT/'.agent/tmp/fvpn-w1-netlab-f27-gate-bin'
        bindir.mkdir(exist_ok=True)
        shutil.copyfile(DEST/'gate-go.py',bindir/'go')
        (bindir/'go').chmod(0o755)
        env['FVPN_F27_REAL_GO']=shutil.which('go')
        env['PATH']=str(bindir)+os.pathsep+env['PATH']
        env['FVPN_NETLAB_EVIDENCE_DIR']='docs/poc/G2-followup27-evidence/lane'
    code=subprocess.run([sys.executable,str(DEST/'run.py'),name,*cmd],cwd=ROOT,env=env,check=False).returncode
    print(name+' exit='+str(code),flush=True)
    if code:
        print('Stopped gate sequence at failure; no automatic retry.',flush=True)
        sys.exit(code)
