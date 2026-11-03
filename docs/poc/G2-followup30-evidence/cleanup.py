#!/usr/bin/env python3
"""TEST ONLY inventory and removal of this task's local scratch tools."""
import json
from pathlib import Path
import shutil
import subprocess
import sys

D=Path(__file__).resolve().parent
ROOT=D.parents[2]
prefix='fvpn-w1-netlab-f30'
checks={
 'containers':['docker','ps','-a','--format','{{.Names}}'],
 'images':['docker','image','ls','--format','{{.Repository}}:{{.Tag}}','--filter','reference='+prefix+'*'],
 'networks':['docker','network','ls','--format','{{.Name}}'],
 'volumes':['docker','volume','ls','--format','{{.Name}}']}
inventory={name:[x for x in subprocess.check_output(cmd,text=True).splitlines() if x.startswith(prefix)] for name,cmd in checks.items()}
before=inventory
if inventory['images']:
    subprocess.run(['docker','image','rm',*inventory['images']],check=True)
inventory={name:[x for x in subprocess.check_output(cmd,text=True).splitlines() if x.startswith(prefix)] for name,cmd in checks.items()}
assert not any(inventory.values()),inventory
removed=[]
for name in (prefix+'-fvpn-netlab',prefix+'-tools'):
    path=ROOT/'.agent/tmp'/name
    if path.is_dir():shutil.rmtree(path)
    elif path.exists():path.unlink()
    removed.append(str(path.relative_to(ROOT)))
remaining=[str(p.relative_to(ROOT)) for p in (ROOT/'.agent/tmp').glob(prefix+'*')]
assert not remaining,remaining
result=dict(test_only=True,owned=inventory,removed_scratch=removed,remaining_task_scratch=remaining)
(D/(sys.argv[1]+'.json')).write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result,indent=2))
