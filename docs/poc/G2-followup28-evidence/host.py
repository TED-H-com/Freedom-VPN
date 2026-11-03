#!/usr/bin/env python3
"""TEST ONLY host admission. No host network mutation and no quietness waiver."""
import datetime
import json
import os
from pathlib import Path
import shlex
import subprocess
import time

ROOT = Path(__file__).resolve().parents[3]
DEST = Path(__file__).resolve().parent
HOLD = Path(os.environ['FVPN_HOLD_SH'])  # the orchestrator's admission script; its private path is redacted for publication


def public(text):
    return text.replace(str(ROOT), '<worktree>').replace(str(Path.home()), '<home>')


def hold():
    proc = subprocess.run(['bash', str(HOLD)], text=True, capture_output=True, check=False)
    return dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(), exit_code=proc.returncode, output=public(proc.stdout + proc.stderr), free=proc.returncode == 0 and proc.stdout.strip() == 'free')


def processes():
    probe = subprocess.run(['pgrep', '-af', 'gates.sh|mk/at-|go test'], capture_output=True, text=True, check=False)
    rows = subprocess.check_output(['ps','-eo','pid=,ppid=,args='],text=True)
    parents, candidates = {}, {}
    for line in rows.splitlines():
        parts = line.split(None,2)
        if len(parts) != 3: continue
        pid, parent, args = int(parts[0]), int(parts[1]), parts[2]
        parents[pid] = parent
        try: words = shlex.split(args)
        except ValueError: continue
        if not words: continue
        exe = Path(words[0]).name
        suite = ((exe in ('make','gmake') and any(w.startswith('mk/at-') or w in ('at-isolation','at-members','node-test-netlab') for w in words)) or
                 (exe in ('bash','sh') and len(words)>1 and Path(words[1]).name=='gates.sh'))
        go_test = exe == 'go' and len(words)>1 and words[1]=='test'
        if suite or go_test:
            candidates[pid] = dict(pid=pid, ppid=parent, args=public(args), suite=suite, go_test=go_test)
    roots=[]
    for pid,row in candidates.items():
        parent=parents.get(pid); seen=set()
        while parent and parent not in seen and parent not in candidates:
            seen.add(parent); parent=parents.get(parent)
        if parent not in candidates: roots.append(row)
    return dict(pgrep=public(probe.stdout), pgrep_exit=probe.returncode, actual=roots,
                suites=[r for r in roots if r['suite'] or (r['go_test'] and ('-tags' in r['args'] or '/tests/e2e/at/' in r['args']))])


def admission(name, quiet=False):
    target=DEST/(name+'.json'); target.parent.mkdir(parents=True,exist_ok=True)
    assert not target.exists(), 'never overwrite a preflight'
    rows=[]; start=time.monotonic()
    while True:
        row=dict(hold=hold(), before=processes())
        if quiet:
            row['vmstat']=subprocess.check_output(['vmstat','5','7'],text=True)
            numbers=[x.split() for x in row['vmstat'].splitlines() if x.strip() and x.split()[0].isdigit()]
            row['current']=[dict(r=int(x[0]), idle=int(x[14])) for x in numbers[1:]]
            row['quiet']=len(row['current'])==6 and all(x['r']<=3 and x['idle']>=80 for x in row['current'])
        row['after']=processes(); row['hold_after']=hold(); rows.append(row)
        waited=time.monotonic()-start
        accepted=row['hold']['free'] and row['hold_after']['free'] and (row['quiet'] and not row['before']['actual'] and not row['after']['actual'] if quiet else len(row['before']['suites'])<2 and len(row['after']['suites'])<2)
        target.write_text(json.dumps(dict(test_only=True,wait_seconds=waited,admitted=accepted,ignore_first_since_boot_row=True,rows=rows),indent=2)+'\n')
        print(f'{name}: wait={waited:.1f}s admitted={accepted} actual={len(row["after"]["actual"])} quiet={row.get("quiet")}',flush=True)
        if accepted: return
        if quiet and waited>=5400: raise RuntimeError('90-minute quiet admission expired; no run')
        time.sleep(60)


if __name__=='__main__':
    import sys
    admission(sys.argv[1],len(sys.argv)>2 and sys.argv[2]=='quiet')
