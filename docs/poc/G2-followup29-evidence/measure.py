#!/usr/bin/env python3
"""TEST ONLY one reserved foreground full-suite measurement, no retry."""
import datetime
import json
import os
from pathlib import Path
import subprocess
import sys
import time
sys.dont_write_bytecode = True
import host

name = sys.argv[1]
mode = sys.argv[2] if len(sys.argv)>2 else 'full'
assert mode in ('full','guard')
claim = host.DEST/(name+'-claim.json')
assert not claim.exists(), 'never retry a measurement'
claim.write_text(json.dumps(dict(start=datetime.datetime.now(datetime.timezone.utc).isoformat(),
    source=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),
    dirty=subprocess.check_output(['git','status','--short'],text=True),
    test_only=True, historical_f28_gate2='FAIL', historical_B1='FAIL'),indent=2)+'\n')
# One bounded foreground wait; a reservation is never silently renewed.
started=time.monotonic(); admission=[]; reserved=False
for attempt in range(30):
    p=subprocess.run(['bash',str(host.HOLD),'--reserve','20','netlab f29 '+name],text=True,capture_output=True)
    admission.append(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),exit_code=p.returncode,output=host.public(p.stdout+p.stderr)))
    (host.DEST/(name+'-reservation.json')).write_text(json.dumps(dict(wait_seconds=time.monotonic()-started,attempts=admission),indent=2)+'\n')
    if p.returncode==0:
        reserved=True; os.environ['FVPN_F29_RESERVATION_EXPIRES']=str(time.time()+20*60); break
    print('Reservation unavailable; waiting 60 seconds',flush=True); time.sleep(60)
if not reserved: raise RuntimeError('bounded reservation wait exhausted; no run')
code=1
try:
    print('Reserved; waiting 60 seconds for earlier admissions to surface',flush=True)
    time.sleep(60)
    host.admission('admission/'+name+'-before',quiet=True)
    env=dict(os.environ,FVPN_F29_MEASURE='1',FVPN_F29_RUN=name,
             FVPN_LAB_PREFIX='fvpn-w1-netlab-f29',FVPN_NETLAB_AT12_OBSERVE='1',
             FVPN_NETLAB_EVIDENCE_DIR='docs/poc/G2-followup29-evidence/'+name)
    command=['make','-f','mk/netlab.mk','netlab-test','FVPN_LAB_PREFIX=fvpn-w1-netlab-f29']
    if mode=='guard':
        env.update(FVPN_NETLAB_AT12_OBSERVE='0', FVPN_NETLAB_TCP_OBSERVE='1', FVPN_NETLAB_GUARD_LOG='1')
        command=[sys.executable,str(host.DEST/'guard-window.py')]
    code=subprocess.run([sys.executable,str(host.DEST/'run.py'),name,*command],env=env).returncode
finally:
    p=subprocess.run(['bash',str(host.HOLD),'--release'],text=True,capture_output=True)
    (host.DEST/(name+'-release.json')).write_text(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),exit_code=p.returncode,output=host.public(p.stdout+p.stderr)),indent=2)+'\n')
    if p.returncode: raise RuntimeError('release failed')
sys.exit(code)
