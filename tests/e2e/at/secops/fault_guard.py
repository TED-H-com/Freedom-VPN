#!/usr/bin/env python3
"""Freeze only this stack's worker during control-loss faults.

Without this guard, live member reconciliation could remove NetBird groups at
service end and confound the independent kernel-timeout observation. No state
rows, routes or clock values are changed. Run beside the live deadline probe.
"""
import datetime
import hashlib
import json
import subprocess
import signal
import time
from pathlib import Path

ROOT=Path(__file__).resolve().parents[4]
OUT=ROOT/'docs/e2e/at/secops/fault-injection.json'
CONTROL='fvpn-w4-secops-control'
WORKER='fvpn-w4-secops-worker'


def utc():
    return datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00','Z')


def state(name):
    p=subprocess.run(['docker','inspect','--format','{{.State.Status}} {{.State.Paused}}',name],capture_output=True,text=True)
    return p.stdout.strip() if p.returncode==0 else 'absent'


def main():
    if state(CONTROL)!='running false' or state(WORKER)!='running false':
        raise SystemExit('guard requires its own running control and unpaused worker')
    rec={'started_at':utc(),'scope':'only own worker frozen while own control is stopped; prevents provider/peer reconciliation from confounding timeout proof','script_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),'events':[]}
    def terminate(_signum,_frame):
        raise SystemExit(2)
    signal.signal(signal.SIGTERM,terminate)
    began=time.monotonic();paused=False;cycles=0
    def record(action,result):
        rec['events'].append({'utc':utc(),'monotonic_elapsed_seconds':time.monotonic()-began,'command':['docker',action,WORKER],'exit_code':result.returncode})
        OUT.write_text(json.dumps(rec,indent=2)+'\n')
    try:
        while time.monotonic()-began<1200 and cycles<2:
            control=state(CONTROL)
            if control=='absent':break
            if control.startswith('exited') and not paused:
                result=subprocess.run(['docker','pause',WORKER],capture_output=True)
                paused=result.returncode==0;record('pause',result)
                if not paused:break
            elif control=='running false' and paused:
                result=subprocess.run(['docker','unpause',WORKER],capture_output=True)
                record('unpause',result)
                if result.returncode:break
                paused=False;cycles+=1
            time.sleep(.25)
    finally:
        if paused:record('unpause',subprocess.run(['docker','unpause',WORKER],capture_output=True))
        rec.update(finished_at=utc(),completed_cycles=cycles,status='passed' if cycles==2 else 'blocked')
        OUT.write_text(json.dumps(rec,indent=2)+'\n')
    print('worker fault guard cycles:',cycles)
    return 0 if cycles==2 else 2


if __name__=='__main__':raise SystemExit(main())
