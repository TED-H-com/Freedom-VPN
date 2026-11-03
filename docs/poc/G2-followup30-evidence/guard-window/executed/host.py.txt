#!/usr/bin/env python3
"""TEST ONLY admission and proc CPU accounting; no host network mutation."""
import datetime
import json
import os
from pathlib import Path
import re
import subprocess
import time

ROOT=Path(__file__).resolve().parents[3]
DEST=Path(__file__).resolve().parent
HOLD=Path(os.environ['FVPN_HOLD_SH'])
PREFIX='fvpn-w1-netlab-f30'
HZ=os.sysconf('SC_CLK_TCK')

def public(s):
    return s.replace(str(HOLD),'<admission-script>').replace(str(ROOT),'<worktree>').replace(str(Path.home()),'<home>')

def utc():return datetime.datetime.now(datetime.timezone.utc).isoformat()

def hold():
    p=subprocess.run(['bash',str(HOLD)],capture_output=True,text=True)
    return dict(utc=utc(),exit_code=p.returncode,output=public(p.stdout+p.stderr),free=p.returncode==0 and p.stdout.strip()=='free')

def admitted_docker(args):
    row=hold()
    with (DEST/'tool-admission.jsonl').open('a') as f:f.write(json.dumps(dict(row,tool='docker',args=args))+'\n')
    if not row['free']:raise RuntimeError('Docker not admitted')
    return subprocess.check_output(['docker',*args],text=True)

def snapshot(runner,ids):
    now=time.monotonic();rows={};errors=[]
    for p in Path('/proc').iterdir():
        if not p.name.isdigit():continue
        try:
            raw=(p/'stat').read_text(); tail=raw[raw.rfind(')')+2:].split()
            pid=int(p.name);comm=raw[raw.find('(')+1:raw.rfind(')')]
            rows[pid]=dict(pid=pid,ppid=int(tail[1]),comm=comm,start=int(tail[19]),ticks=int(tail[11])+int(tail[12]),cgroup=(p/'cgroup').read_text())
        except (FileNotFoundError,ProcessLookupError):continue
        except PermissionError:errors.append(int(p.name))
    def descendant(pid):
        seen=set()
        while pid in rows and pid not in seen:
            if pid==runner:return True
            seen.add(pid);pid=rows[pid]['ppid']
        return False
    for r in rows.values():
        if descendant(r['pid']):kind='runner_descendants'
        elif any(i in r['cgroup'] for i in ids):kind='owned_containers'
        elif r['ppid']==2:kind='kernel_threads'
        elif re.fullmatch(r'dockerd|containerd|containerd-shim.*|docker-proxy|runc',r['comm']):kind='container_runtime'
        else:kind='foreign'
        r['class']=kind;del r['cgroup']
    return now,rows,errors

def external_suites(rows):
    p=subprocess.run(['pgrep','-af','gates.sh|mk/at-|go test|node-test-netlab|netlab-test'],capture_output=True,text=True)
    found=[]
    for line in p.stdout.splitlines():
        pid=int(line.split(None,1)[0]);row=rows.get(pid)
        if not row or row['class']=='runner_descendants':continue
        try:words=Path(f'/proc/{pid}/cmdline').read_bytes().split(b'\0');words=[w.decode(errors='replace') for w in words if w]
        except (FileNotFoundError,ProcessLookupError):continue
        if not words:continue
        exe=Path(words[0]).name
        actual=(exe=='go' and words[1:2]==['test']) or (exe in ('make','gmake') and any(w.startswith('mk/at-') or w in ('node-test-netlab','netlab-test') for w in words)) or (exe in ('sh','bash') and any(Path(w).name=='gates.sh' for w in words[1:3]))
        if actual:found.append(dict(pid=pid,command=public(' '.join(words))))
    return dict(pgrep_exit=p.returncode,foreign_suites=found)

class Quiet:
    def __init__(self,runner):
        self.runner=runner;self.previous=None;self.next_sample=0
    def sample(self):
        listing=admitted_docker(['ps','--no-trunc','--format','{{.ID}} {{.Names}} {{.Image}}'])
        own=[];foreign=[];shared=[]
        for line in listing.splitlines():
            ident,name,image=line.split(None,2)
            if name.startswith(PREFIX):own.append(ident)
            elif name=='fvpn-pg' and image=='postgres:17-alpine':shared.append(dict(name=name,image=image,cpu_class='foreign'))
            elif name.startswith('fvpn-'):foreign.append(name)
        now,rows,errors=snapshot(self.runner,own)
        result=dict(utc=utc(),monotonic_seconds=now,foreign_containers=foreign,shared_services=shared,owned_container_ids=own,proc_read_errors=errors,processes=external_suites(rows))
        result['vmstat']=subprocess.check_output(['vmstat'],text=True)
        if self.previous:
            before,old,_=self.previous;elapsed=now-before
            ticks={k:0 for k in ('runner_descendants','owned_containers','kernel_threads','container_runtime','foreign')};contributors=[]
            for pid,r in rows.items():
                prior=old.get(pid)
                delta=max(0,r['ticks']-prior['ticks']) if prior and prior['start']==r['start'] else r['ticks'] if r['start']/HZ>=before else 0
                ticks[r['class']]+=delta
                if delta and r['class']=='foreign':contributors.append(dict(pid=pid,comm=r['comm'],ticks=delta))
            result.update(interval_seconds=elapsed,cpu_cores={k:v/HZ/elapsed for k,v in ticks.items()},foreign_contributors=contributors,
                          disappeared_pids=[pid for pid in old if pid not in rows])
            result['quiet']=not errors and not foreign and not result['processes']['foreign_suites'] and result['cpu_cores']['foreign']<=.5
        else:result['quiet']=False
        self.previous=(now,rows,errors)
        return result
