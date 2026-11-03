#!/usr/bin/env python3
"""Record every 5-second current sample; host quiet is context, not acceptance."""
import os
import datetime
import json
from pathlib import Path
import subprocess
import time

root=Path(__file__).resolve().parents[4]
private=root/'.agent/f17';private.mkdir(parents=True,exist_ok=True)
from suite_admission import wait
evidence=Path(os.environ.get('FVPN_AT_EVIDENCE_DIR',root/'docs/e2e/at/isolation'))
evidence.mkdir(parents=True,exist_ok=True)
wait(evidence)
started=time.monotonic();samples=[]
while True:
    at=datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00','Z')
    load=Path('/proc/loadavg').read_text().strip()
    processes=subprocess.run(['pgrep','-af','gates.sh|mk/at-'],capture_output=True,text=True).stdout
    with (private/'host-processes.txt').open('a') as f:f.write(at+'\n'+processes)
    raw=subprocess.check_output(['vmstat','5','7'],text=True)
    current=[line.split() for line in raw.splitlines()[2:]][1:]
    quiet=bool(current) and all(int(row[0])<=3 and int(row[14])>=80 for row in current)
    rows=[]
    for line in processes.splitlines():
        pid,_,command=line.partition(' ')
        if 'pgrep' not in command and pid.isdigit():rows.append({'pid':int(pid),'kind':'gates' if 'gates.sh' in command else 'mk/at suite'})
    samples.append(dict(at=at,load_average=load.split()[:3],processes=rows,vmstat=raw,quiet=quiet))
    elapsed=time.monotonic()-started
    result=dict(samples=samples,quiet=quiet,wait_seconds=round(elapsed,3),limit_seconds=1800,
                rule='all six current rows r<=3 and id>=80; first since-boot row ignored; host state is not a pass criterion')
    (evidence/'host-preflight.json').write_text(json.dumps(result,indent=2)+'\n')
    print(f'host preflight elapsed={elapsed:.1f}s quiet={quiet} r={[r[0] for r in current]} id={[r[14] for r in current]} load={load.split()[:3]}',flush=True)
    if quiet or elapsed>=1800:break
