#!/usr/bin/env python3
"""TEST ONLY foreground full-suite monitoring inside its admitted make ancestry."""
import json
import os
import subprocess
import threading
import time
import host


def run(real,args):
    name=os.environ['FVPN_F30_RUN'];rows=[];stop=threading.Event();lock=threading.Lock()
    target=host.DEST/(name+'-suite-quiet.jsonl');assert not target.exists()
    def monitor():
        quiet=host.Quiet(int(os.environ['FVPN_F30_RUNNER_PID']));tick=time.monotonic()
        while not stop.is_set():
            try:row=quiet.sample()
            except Exception as exc:row=dict(utc=host.utc(),monotonic_seconds=time.monotonic(),quiet=False,error=host.public(str(exc)))
            with lock:rows.append(row)
            with target.open('a') as f:f.write(json.dumps(row)+'\n')
            tick+=4;stop.wait(max(0,tick-time.monotonic()))
    thread=threading.Thread(target=monitor);thread.start();entered=time.monotonic();began=None;ended=None;code=125
    try:
        while time.monotonic()-entered<120:
            time.sleep(.2)
            with lock:ready=len(rows)>=2 and rows[-1]['quiet'];last=rows[-1] if rows else {}
            if ready:break
        else:raise RuntimeError('fresh post-startup quiet check not met; suite not launched')
        binary=host.admitted_docker(['exec',host.PREFIX,'sha256sum','/usr/local/bin/fvpn-netlab']).split()[0]
        (host.DEST/(name+'-binary.json')).write_text(json.dumps(dict(sha256=binary,quiet_check_utc=last['utc'],wait_seconds=time.monotonic()-entered),indent=2)+'\n')
        for _ in range(10):
            admission=host.hold()
            with (host.DEST/'tool-admission.jsonl').open('a') as f:f.write(json.dumps(dict(admission,tool='go',args=args))+'\n')
            if not admission['free']:
                time.sleep(60);continue
            began=time.monotonic();code=subprocess.run([real,*args]).returncode;ended=time.monotonic()
            if code!=125:break
            # Enforcement refused before a test ran. This is not a test result.
            time.sleep(60)
    finally:
        with lock:count=len(rows)
        until=time.monotonic()+6
        while time.monotonic()<until:
            with lock:advanced=len(rows)>count
            if advanced:break
            time.sleep(.05)
        stop.set();thread.join()
        observed=[r for r in rows if began is not None and r['monotonic_seconds']>=began and (ended is None or r['monotonic_seconds']-r.get('interval_seconds',0)<=ended)]
        result=dict(test_only=True,scope='actual full-suite process, with fresh preflight and a final covering interval',runner_pid=int(os.environ['FVPN_F30_RUNNER_PID']),monitor_pid=os.getpid(),suite_began_monotonic=began,suite_ended_monotonic=ended,checks=len(observed),qualified=bool(observed) and all(r['quiet'] and r.get('interval_seconds',99)<=5 for r in observed),exit_code=code,max_foreign_cpu=max((r.get('cpu_cores',{}).get('foreign',0) for r in observed),default=None),max_interval_seconds=max((r.get('interval_seconds',0) for r in observed),default=None))
        (host.DEST/(name+'-suite-qualification.json')).write_text(json.dumps(result,indent=2)+'\n')
    return code
