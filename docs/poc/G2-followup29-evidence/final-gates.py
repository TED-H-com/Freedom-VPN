#!/usr/bin/env python3
"""TEST ONLY required foreground gates after the completed gate1/window."""
import os
import subprocess
import sys
sys.dont_write_bytecode=True
import host

steps=[
 ('final-gate3-agent',['go','test','-count=1','./node/agent/...','./node/updater/...','./packaging/installer/...']),
 ('final-gate3-reviews',['go','test','./node/agent','./node/enforce','-run','^TestReviewR1[6-9]_','-count=1']),
 ('final-gate3-consumers',['go','test','-count=1','./tests/e2e/at/usage/...','./tests/e2e/internal/counterread/...']),
 ('final-gate4',['go','test','-count=1','./tests/netlab','-run','^TestCommittedEvidenceFiles_ValidAndHygieneClean$']),
 ('final-gate5',['go','test','-count=1','./tests/handover/...']),
 ('final-gate6',['make','go-lint','go-with-disk-tmp=set -eu;']),
 ('final-gate7',['go','vet','./...']),
 ('final-gate8',['bash','-c','test -z "$(gofmt -l $(git ls-files \'*.go\'))"']),
 ('final-gate9',['git','diff','--check']),
]
remaining=steps
if len(sys.argv)>1:
    assert len(sys.argv)==3 and sys.argv[1]=='--after'
    remaining=steps[[n for n,_ in steps].index(sys.argv[2])+1:]
for name,cmd in remaining:
    env=dict(os.environ,PYTHONDONTWRITEBYTECODE='1')
    if name=='final-gate4':
        subprocess.run([sys.executable,str(host.DEST/'publish.py'),'--partial'],check=True)
        subprocess.run(['git','add','docs/poc/G2-followup29-evidence'],check=True)
    code=subprocess.run([sys.executable,str(host.DEST/'run.py'),name,*cmd],env=env,check=False).returncode
    print(name+' exit='+str(code),flush=True)
    if code and name.startswith('final-gate3-'):
        print('Unchanged consumer failure retained; continuing required gates.',flush=True)
        continue
    if code:
        print('Stopped at failed gate; no retry. Explicit continuation required.',flush=True)
        sys.exit(code)
