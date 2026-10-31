#!/usr/bin/env python3
"""Fresh, sequential transport and restore runs on the assigned lab binding."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import uuid

from followup10_run import cleanup_extra_restore_objects, cleanup_disk_state
from run import ROOT, PRIVATE, PUBLIC, PREFIX, PORT_BASE, command, inventory, summarize_go, utc, write_json


def main(phase=None):
    phase = phase or os.environ.get('FVPN_SECOPS_FOLLOWUP11', 'transports')
    if phase not in ('transports', 'restore'):
        raise SystemExit('expected transports or restore')
    if os.getenv('FVPN_E2E_PREFIX', PREFIX) != PREFIX or os.getenv('FVPN_E2E_PORT_BASE', PORT_BASE) != PORT_BASE:
        raise SystemExit('assigned w4-secops/30000 binding only')
    PRIVATE.mkdir(parents=True, exist_ok=True)
    (ROOT/'.agent/tmp').mkdir(parents=True, exist_ok=True)
    main_revision = subprocess.check_output(['git','merge-base','HEAD','origin/main'],cwd=ROOT,text=True).strip()
    log = subprocess.check_output(['git','log','--format=%s',main_revision],cwd=ROOT,text=True)
    d099 = any(line.startswith('roadmap: w3-wire-nodes follow-up 29 merged') for line in log.splitlines())
    record = dict(driver_run_id=str(uuid.uuid4()),source_revision=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),
                  application_base=main_revision, phase=phase, started_at=utc(), d099_roadmap_present=d099,
                  prepare=dict(prefix=PREFIX,port_base=PORT_BASE,reset=True,grace_seconds=90))
    label = 'followup11-'+phase
    with tempfile.TemporaryDirectory(prefix=label+'.',dir=ROOT/'.agent/tmp') as scratch:
        env=dict(os.environ, PATH=str(Path.home()/'.local/go/bin')+':'+os.getenv('PATH',''),
                 GOTOOLCHAIN='local',CGO_ENABLED='0',PYTHONDONTWRITEBYTECODE='1',TMPDIR=scratch,GOTMPDIR=scratch,
                 E2E_RESET='1',FVPN_E2E_PREFIX=PREFIX,FVPN_E2E_PORT_BASE=PORT_BASE,FVPN_E2E_GRACE_SECONDS='90',
                 GOFLAGS='-p=2',GOMAXPROCS='2',FVPN_E2E_STATE=str(ROOT/'.agent/e2e-state-w4-secops'),
                 FVPN_SECOPS_FOLLOWUP10='1',FVPN_SECOPS_FOLLOWUP11=phase)
        before=inventory(env); record['inventory_before']=before
        if any(v['count'] or v['exit_code'] for v in before.values()):
            raise SystemExit('own namespace occupied; refusing to attach or tear down')
        (PUBLIC/(label+'-live.json')).unlink(missing_ok=True)
        write_json(PUBLIC/(label+'-run.json'),record)
        try:
            print(label+': fresh e2e-up',flush=True)
            record['up'],_=command(label+'-e2e-up',['make','-f','mk/e2e.mk','e2e-up'],env,1500)
            if record['up']['exit_code']==0:
                name='TestAT30_Followup10_LiveRestoreReconcile' if phase=='restore' else 'TestAT29_AT31_Followup11_Transports'
                args=['go','test','-json','-tags','e2e,integration','-p','1','-parallel','2','-count=1','-timeout','45m','-run','^'+name+'$','./tests/e2e/at/secops']
                cmd,raw=command(label+'-live',args,env,2760)
                record['live']=dict(cmd,**summarize_go(raw,[name],cmd['exit_code']))
        finally:
            cleanup_extra_restore_objects(env)
            for suffix in ('grant_expiry','lease_expiry','device_revoke'):
                name='fvpn-w4-secops-member-f11-'+suffix.replace('_','-')
                subprocess.run(['docker','rm','-fv',name],env=env,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
                subprocess.run(['docker','volume','rm',name],env=env,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
            record['down'],_=command(label+'-e2e-down',['make','-f','mk/e2e.mk','e2e-down'],env,300)
            cleanup_extra_restore_objects(env)
            record['state_cleanup']=cleanup_disk_state(env)
            record['inventory_after']=inventory(env);record['finished_at']=utc()
            write_json(PUBLIC/(label+'-run.json'),record)
    # Root summaries are rendered only after both independent phases finish.
    return 0 if record.get('live',{}).get('status')=='passed' and record.get('state_cleanup',{}).get('removed') else 2


if __name__=='__main__': raise SystemExit(main())
