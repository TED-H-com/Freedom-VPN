#!/usr/bin/env python3
"""TEST ONLY required foreground gates after the completed gate1/window."""
import os
import subprocess
import sys
sys.dont_write_bytecode=True
import host

steps=[
 ('gate2',['make','-f','mk/netlab.mk','netlab-test','FVPN_LAB_PREFIX=fvpn-w1-netlab-f28']),
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
remaining=steps
if len(sys.argv)>1:
    assert len(sys.argv)==3 and sys.argv[1]=='--after'
    remaining=steps[[n for n,_ in steps].index(sys.argv[2])+1:]
for name,cmd in remaining:
    env=dict(os.environ,PYTHONDONTWRITEBYTECODE='1')
    if name=='gate2':
        host.admission('admission/gate2-before')
        env.update(FVPN_F28_GATE2='1',FVPN_NETLAB_EVIDENCE_DIR='docs/poc/G2-followup28-evidence/lane')
    if name=='gate4':
        subprocess.run([sys.executable,str(host.DEST/'publish.py'),'--partial'],check=True)
        subprocess.run(['git','add','docs/poc/G2-followup28-evidence'],check=True)
    code=subprocess.run([sys.executable,str(host.DEST/'run.py'),name,*cmd],env=env,check=False).returncode
    print(name+' exit='+str(code),flush=True)
    if code:
        print('Stopped at failed gate; no retry. Explicit continuation required.',flush=True)
        sys.exit(code)
