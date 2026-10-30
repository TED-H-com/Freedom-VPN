#!/usr/bin/env python3
"""Fresh standard lab, TEST ONLY finite grace, bounded own-prefix cleanup."""
import os
import subprocess
import tempfile
import uuid
from run import ROOT, PRIVATE, PUBLIC, PREFIX, PORT_BASE, command, inventory, summarize_go, utc, write_json


def main():
    PRIVATE.mkdir(parents=True,exist_ok=True)
    (ROOT/'.agent/tmp').mkdir(parents=True,exist_ok=True)
    source=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip()
    record={'driver_run_id':str(uuid.uuid4()),'source_revision':source,'started_at':utc(),
            'expected':'fleet API short lease and finite grace; real proxy principals; 60-peer counted traffic and node telemetry',
            'prepare':{'grace_seconds':90,'reset':True,'prefix':PREFIX,'port_base':PORT_BASE}}
    with tempfile.TemporaryDirectory(prefix='secops-f7.',dir=ROOT/'.agent/tmp') as scratch:
        env=dict(os.environ,GOTOOLCHAIN='local',CGO_ENABLED='0',PYTHONDONTWRITEBYTECODE='1',TMPDIR=scratch,GOTMPDIR=scratch,E2E_RESET='1',FVPN_E2E_GRACE_SECONDS='90',FVPN_E2E_PREFIX=PREFIX,FVPN_E2E_PORT_BASE=PORT_BASE,FVPN_E2E_STATE=str(ROOT/'.agent/e2e-state-w4-secops'))
        before=inventory(env);record['inventory_before']=before
        if any(v['count'] or v['exit_code'] for v in before.values()):
            raise SystemExit('own namespace occupied; refusing to attach or tear down')
        write_json(PUBLIC/'followup7-run.json',record)
        build,_=command('followup7-proxyprobe',['go','build','-o','.agent/secops/build/proxyprobe','./tests/e2e/at/secops/proxyprobe'],env,120)
        record['proxyprobe_build']=build
        if build['exit_code']:return 1
        try:
            print('follow-up 7: fresh finite-grace e2e-up',flush=True)
            record['up'],_=command('followup7-e2e-up',['make','-f','mk/e2e.mk','e2e-up'],env,1500)
            if record['up']['exit_code']==0:
                name='TestAT28_AT31_Followup7'
                args=['go','test','-json','-tags','e2e,integration','-p','1','-parallel','2','-count=1','-timeout','65m','-run','^'+name+'$','./tests/e2e/at/secops']
                cmd,raw=command('followup7-live',args,dict(env,FVPN_SECOPS_FOLLOWUP7='1'),3960)
                record['live']=dict(cmd,**summarize_go(raw,[name],cmd['exit_code']))
        finally:
            record['down'],_=command('followup7-e2e-down',['make','-f','mk/e2e.mk','e2e-down'],env,300)
            record['inventory_after']=inventory(env);record['finished_at']=utc()
            write_json(PUBLIC/'followup7-run.json',record)
    subprocess.run(['python3','tests/e2e/at/secops/followup7_records.py'],cwd=ROOT,check=True)
    # Full cases remain blocked if any required lab/operator row is unproven.
    return 2 if __import__('json').loads((PUBLIC/'followup7-verdict.json').read_text())['blocked'] else 0


if __name__=='__main__':raise SystemExit(main())
