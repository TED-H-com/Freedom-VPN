"""Recognize real suite executables, never strings in a waiting shell's script."""
import json
import os
from pathlib import Path
import subprocess
import time


def classify(args, cwd=''):
    if not args:
        return None
    exe=Path(args[0]).name
    words=set(args[1:])
    if exe in ('make','gmake'):
        if 'node-test-netlab' in words and 'mk/node.mk' in words:
            return 'agent-core netlab'
        if 'netlab-test' in words and 'mk/netlab.mk' in words and '/w1-netlab' in cwd:
            return 'netlab 27'
        if 'at-members' in words and 'mk/at-members.mk' in words:
            return 'members assembled'
        if any(a.startswith('mk/at-') for a in args):
            return 'acceptance suite'
    if exe=='go' and len(args)>1 and args[1]=='test' and '/w1-netlab' in cwd and any(a=='netlab' or a.startswith('-tags=') and 'netlab' in a for a in args):
        return 'netlab 27'
    if exe in ('sh','bash') and len(args)>1 and '-c' not in words and any(Path(a).name=='gates.sh' for a in args[1:]):
        return 'gates suite'
    return None


def processes():
    rows={}
    for directory in Path('/proc').iterdir():
        if not directory.name.isdigit(): continue
        try:
            args=(directory/'cmdline').read_bytes().decode().rstrip('\0').split('\0')
            stat=(directory/'stat').read_text().rsplit(')',1)[1].split()
            try: cwd=os.readlink(directory/'cwd')
            except OSError: cwd=''
            rows[int(directory.name)]=(int(stat[1]),classify(args,cwd))
        except (OSError,ValueError,UnicodeError): continue
    ancestors=set(); pid=os.getpid()
    while pid in rows:
        ancestors.add(pid);pid=rows[pid][0]
    selected={p:k for p,(_,k) in rows.items() if k and p not in ancestors}
    roots={}
    for pid,kind in selected.items():
        p=rows[pid][0]
        while p in rows and p not in selected: p=rows[p][0]
        if p not in selected: roots[pid]=kind
    return roots


def wait(evidence=None, exclusive=True):
    started=time.monotonic(); snapshots=[]
    while True:
        raw=subprocess.run(['pgrep','-af','gates.sh|mk/at-'],capture_output=True,text=True).stdout
        print('shared-host suite inventory:\n'+raw,flush=True)
        roots=processes()
        elapsed=time.monotonic()-started
        sensitive={p:k for p,k in roots.items() if k in ('agent-core netlab','netlab 27','members assembled')}
        snapshots.append(dict(wait_seconds=round(elapsed,3),other_roots=roots,timing_sensitive=sensitive))
        result=dict(wait_seconds=round(elapsed,3),snapshots=snapshots,exclusive=exclusive)
        if evidence:
            path=Path(evidence)/'suite-admission.jsonl';path.parent.mkdir(parents=True,exist_ok=True)
            with path.open('a') as f: f.write(json.dumps(snapshots[-1])+'\n')
        if len(roots)<2 and (not exclusive or not sensitive):
            print('suite admission: '+json.dumps(result),flush=True);return result
        if sensitive and elapsed>=5400:
            raise RuntimeError('timing-sensitive suite still active after 90 minutes; assembled run not started')
        print('suite admission waiting 30 seconds',flush=True)
        time.sleep(30)


if __name__=='__main__':
    wait(os.environ.get('FVPN_AT_EVIDENCE_DIR'))
