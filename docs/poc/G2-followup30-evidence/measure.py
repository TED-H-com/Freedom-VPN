#!/usr/bin/env python3
"""TEST ONLY bounded reservation, continuous quiet checks and foreground run."""
import json
import os
from pathlib import Path
import subprocess
import sys
import threading
import time
sys.dont_write_bytecode=True
import host

name,mode=sys.argv[1:]
assert mode in ('collector','guard','full')
if mode=='collector':assert name in ('collector1','collector2','collector3')
if mode=='guard':
    assert name=='guard-window'
    results=list(host.DEST.glob('collector*/collector-result.json'))
    assert any(json.loads(p.read_text())['pass_check'] for p in results),'live collector check must pass first'
claim=host.DEST/(name+'-claim.json');assert not claim.exists(),'no retry/overwrite'
claim.write_text(json.dumps(dict(utc=host.utc(),source=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),dirty=host.public(subprocess.check_output(['git','status','--short'],text=True)),mode=mode,historical_B1='FAIL'),indent=2)+'\n')
minutes=10 if mode=='collector' else 20
attempts=[];started=time.monotonic();reserved=False
for _ in range(10):
    p=subprocess.run(['bash',str(host.HOLD),'--reserve',str(minutes),'netlab f30 '+name],capture_output=True,text=True)
    attempts.append(dict(utc=host.utc(),exit_code=p.returncode,output=host.public(p.stdout+p.stderr)))
    (host.DEST/(name+'-reservation.json')).write_text(json.dumps(dict(wait_seconds=time.monotonic()-started,attempts=attempts),indent=2)+'\n')
    if p.returncode==0:reserved=True;break
    print('Reservation unavailable; waiting 60 seconds',flush=True);time.sleep(60)
if not reserved:raise RuntimeError('bounded reservation wait exhausted')
stop=threading.Event();records=[];lock=threading.Lock();code=1

def monitor():
    quiet=host.Quiet(os.getpid());tick=time.monotonic()
    while not stop.is_set():
        try:row=quiet.sample()
        except Exception as exc:row=dict(utc=host.utc(),quiet=False,error=host.public(str(exc)))
        with lock:records.append(row)
        with (host.DEST/(name+'-quiet.jsonl')).open('a') as f:f.write(json.dumps(row)+'\n')
        tick+=4
        stop.wait(max(0,tick-time.monotonic()))
try:
    print('Reserved; settling for 60 seconds',flush=True);time.sleep(60)
    thread=threading.Thread(target=monitor);thread.start()
    wait_start=time.monotonic()
    while True:
        time.sleep(1)
        with lock:ready=len(records)>=2 and records[-1]['quiet'];elapsed=time.monotonic()-wait_start
        if ready:break
        if elapsed>120:raise RuntimeError('quiet preflight not met in 120 seconds; no launch')
    (host.DEST/(name+'-preflight.json')).write_text(json.dumps(dict(wait_seconds=elapsed,settle_seconds=60),indent=2)+'\n')
    env=dict(os.environ,FVPN_F30_RUN=name,FVPN_LAB_PREFIX=host.PREFIX,FVPN_NETLAB_EVIDENCE_DIR='docs/poc/G2-followup30-evidence/'+name,
             FVPN_NETLAB_AT12_OBSERVE='0',FVPN_NETLAB_TCP_OBSERVE='1' if mode=='guard' else '0',FVPN_NETLAB_GUARD_LOG='1' if mode=='guard' else '0')
    if mode=='collector':cmd=[sys.executable,str(host.DEST/'collector-run.py'),name]
    elif mode=='guard':cmd=[sys.executable,str(host.DEST/'guard-window.py')]
    else:cmd=['make','-f','mk/netlab.mk','netlab-test','FVPN_LAB_PREFIX='+host.PREFIX]
    code=subprocess.run([sys.executable,'-B',str(host.DEST/'run.py'),name,*cmd],env=env).returncode
    if mode!='collector':
        cleanup=subprocess.run([sys.executable,'-B',str(host.DEST/'run.py'),name+'-cleanup',sys.executable,'-B',str(host.DEST/'cleanup.py'),name+'-inventory'],env=env).returncode
        code=code or cleanup
finally:
    stop.set()
    if 'thread' in locals():thread.join()
    measured=[r for r in records if 'interval_seconds' in r]
    (host.DEST/(name+'-qualification.json')).write_text(json.dumps(dict(test_only=True,checks=len(records),qualified=bool(measured) and all(r['quiet'] and r.get('interval_seconds',0)<=5 for r in records[1:]),max_check_interval_seconds=max((r['interval_seconds'] for r in measured),default=None),max_foreign_cpu=max((r['cpu_cores']['foreign'] for r in measured),default=None),exit_code=code,
       accounting='utime+stime differences for each PID/start-time pair. New PIDs use their accumulated ticks when born in the interval. Processes that exit between snapshots cannot be recovered from proc; disappeared PIDs are retained. Excluded classes are reported separately.'),indent=2)+'\n')
    p=subprocess.run(['bash',str(host.HOLD),'--release'],capture_output=True,text=True)
    (host.DEST/(name+'-release.json')).write_text(json.dumps(dict(utc=host.utc(),exit_code=p.returncode,output=host.public(p.stdout+p.stderr)),indent=2)+'\n')
    assert p.returncode==0,'release failed'
sys.exit(code)
