"""TEST ONLY remove precisely this task's owned Docker objects and scratch."""
import json
import re
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
# A rebuilt tag can leave its previous task-owned manifest dangling. Inspect
# only IDs emitted by our own builds; never prune or force-remove shared data.
known = set()
for log in [OUT/'interrupted-2026-11-01/mechanism-image.txt', OUT/'resume-image.txt', OUT/'gate2.txt']:
    if log.exists():
        known.update(re.findall(r'exporting manifest list (sha256:[0-9a-f]{64})',log.read_text()))
legacy = []
for ref in sorted(known):
    inspect = subprocess.run(['docker','image','inspect',ref],capture_output=True,text=True)
    if inspect.returncode:
        legacy.append(dict(ref=ref,status='absent'))
        continue
    info = json.loads(inspect.stdout)[0]
    foreign = [tag for tag in (info.get('RepoTags') or []) if not owned(tag.split(':')[0])]
    if foreign:
        legacy.append(dict(ref=ref,status='shared alias, left untouched'))
        continue
    subprocess.run(['docker','image','rm',ref],check=True)
    removed.append(['untagged-owned-build',ref])
    legacy.append(dict(ref=ref,status='removed'))
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
(OUT/'cleanup.json').write_text(json.dumps(dict(test_only=True,removed=removed,remaining=left,known_build_manifests=legacy),indent=2)+'\n')
print('Owned containers/images/networks/volumes: 0/0/0/0')
