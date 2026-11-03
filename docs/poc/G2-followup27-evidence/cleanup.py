#!/usr/bin/env python3
"""TEST ONLY: remove only this follow-up's named Docker and disk objects."""
import datetime
import json
from pathlib import Path
import shutil
import subprocess

DEST=Path(__file__).resolve().parent
ROOT=DEST.parents[2]
PREFIX='fvpn-w1-netlab-f27'

def owned(args):
    lines=subprocess.check_output(['docker',*args],text=True).splitlines()
    return [x for x in lines if x==PREFIX or x.startswith(PREFIX+'-') or x.startswith(PREFIX+':')]
def inventory():
    return dict(containers=owned(['ps','-a','--format','{{.Names}}']),
        images=owned(['image','ls','--format','{{.Repository}}:{{.Tag}}']),
        networks=owned(['network','ls','--format','{{.Name}}']),
        volumes=owned(['volume','ls','--format','{{.Name}}']))

before=inventory()
removed=[]
for kind,cmd in [('containers',['rm','-fv']),('networks',['network','rm']),
                 ('volumes',['volume','rm']),('images',['image','rm'])]:
    for name in before[kind]:
        subprocess.run(['docker',*cmd,name],check=True,stdout=subprocess.DEVNULL)
        removed.append(dict(kind=kind,name=name))
disk=[]
for path in sorted((ROOT/'.agent/tmp').glob(PREFIX+'*')):
    # Any surviving scratch needs the lab's container cleanup, never host sudo.
    if path.is_dir() and path.name.endswith('-scratch'):
        assert not any(path.iterdir()),str(path.relative_to(ROOT))
    if path.is_dir():shutil.rmtree(path)
    else:path.unlink()
    disk.append(str(path.relative_to(ROOT)))
after=inventory()
assert not any(after.values())
assert not list((ROOT/'.agent/tmp').glob(PREFIX+'*'))
record=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),before=before,
            removed=removed,disk_removed=disk,after=after)
(DEST/'cleanup.json').write_text(json.dumps(record,indent=2)+'\n')
print('Owned Docker objects after cleanup:',{k:len(v) for k,v in after.items()})
print('Owned task disk paths removed:',len(disk))
