"""TEST ONLY remove precisely this task's owned Docker objects and scratch."""
import json
from pathlib import Path
import subprocess

PREFIX = 'fvpn-w1-netlab-f26'
OUT = Path('docs/poc/G2-followup26-evidence')


def owned(name):
    return name == PREFIX or name.startswith(PREFIX+'-')


def listed(args):
    return subprocess.check_output(['docker']+args,text=True).splitlines()


removed = []
for name in listed(['ps','-a','--format','{{.Names}}']):
    if owned(name):
        subprocess.run(['docker','rm','-fv',name],check=True)
        removed.append(['container',name])
for kind in ('network','volume'):
    for name in listed([kind,'ls','--format','{{.Name}}']):
        if owned(name):
            subprocess.run(['docker',kind,'rm',name],check=True)
            removed.append([kind,name])
for name in listed(['image','ls','--format','{{.Repository}}:{{.Tag}}']):
    if owned(name.split(':')[0]):
        subprocess.run(['docker','image','rm',name],check=True)
        removed.append(['image',name])
left = dict(containers=[s for s in listed(['ps','-a','--format','{{.Names}}']) if owned(s)],
            networks=[s for s in listed(['network','ls','--format','{{.Name}}']) if owned(s)],
            volumes=[s for s in listed(['volume','ls','--format','{{.Name}}']) if owned(s)],
            images=[s for s in listed(['image','ls','--format','{{.Repository}}:{{.Tag}}']) if owned(s.split(':')[0])])
for p in Path('.agent/tmp').glob(PREFIX+'*'):
    if p.is_file():
        p.unlink()
    elif p.is_dir():
        p.rmdir()  # Refuse to hide nonempty root-owned scratch.
assert all(not names for names in left.values()),left
(OUT/'cleanup.json').write_text(json.dumps(dict(test_only=True,removed=removed,remaining=left),indent=2)+'\n')
print('Owned containers/images/networks/volumes: 0/0/0/0')
