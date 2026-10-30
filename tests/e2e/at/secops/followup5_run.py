#!/usr/bin/env python3
"""Run on the shared devstack implementation; never seed projected rows."""
import argparse
import json
import os
import subprocess
import tempfile
from run import ROOT, PRIVATE, PUBLIC, PREFIX, PORT_BASE, SUITES, command, inventory, summarize_go, utc, write_json


def components(env):
    records=[]
    for name,ats,package,required in SUITES:
        args=['go','test','-json','-tags','e2e,integration','-p','1','-parallel','2','-count=1','-timeout','8m','-run','^('+'|'.join(required)+')$',package]
        cmd,raw=command('followup5-'+name,args,env)
        record=dict(id=name,acceptance_cases=ats,evidence_level='component',**cmd,**summarize_go(raw,required,cmd['exit_code']))
        records.append(record)
        print(name+': '+record['status'],flush=True)
    write_json(PUBLIC/'components.json',records)
    return records


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('--components-only',action='store_true')
    args=parser.parse_args()
    PRIVATE.mkdir(parents=True,exist_ok=True)
    (ROOT/'.agent/tmp').mkdir(parents=True,exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='secops-f5.',dir=ROOT/'.agent/tmp') as scratch:
        env=dict(os.environ,GOTOOLCHAIN='local',CGO_ENABLED='0',PYTHONDONTWRITEBYTECODE='1',TMPDIR=scratch,GOTMPDIR=scratch,E2E_RESET='1',FVPN_E2E_PREFIX=PREFIX,FVPN_E2E_PORT_BASE=PORT_BASE,FVPN_E2E_STATE=str(ROOT/'.agent/e2e-state-w4-secops'))
        if args.components_only:
            return 0 if all(r['status']=='passed' for r in components(env)) else 1
        before=inventory(env)
        if any(v['exit_code'] or v['count'] for v in before.values()):
            raise SystemExit('assigned namespace occupied; refusing to attach or tear down')
        head=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip()
        roadmap=subprocess.check_output(['git','log','-100','--oneline','HEAD'],cwd=ROOT,text=True)
        scale='roadmap: w3-devstack-e2e follow-up 10 merged' in roadmap
        record={'started_at':utc(),'source_revision':head,'scale60_enabled':scale,'inventory_before':before}
        try:
            up,_=command('followup5-e2e-up',['make','-f','mk/e2e.mk','e2e-up'],env,1200);record['up']=up
            if up['exit_code']==0:
                test='TestAT28_AT31_Followup5'
                guard_log=(PRIVATE/'followup5-fault-guard.log').open('wb')
                (PRIVATE/'followup5-fault-guard.log').chmod(0o600)
                guard=subprocess.Popen(['python3','tests/e2e/at/secops/fault_guard.py'],cwd=ROOT,env=env,stdout=guard_log,stderr=subprocess.STDOUT)
                cmd,raw=command('followup5-live',['go','test','-json','-tags','e2e,integration','-p','1','-parallel','2','-count=1','-timeout','70m','-run','^'+test+'$','./tests/e2e/at/secops'],dict(env,FVPN_SECOPS_FOLLOWUP5='1',FVPN_SECOPS_LOGIN_BURST='1',FVPN_SECOPS_SCALE60='1' if scale else '0'),4260)
                record['live']=dict(cmd,**summarize_go(raw,[test],cmd['exit_code']))
                if guard.poll() is None:guard.terminate()
                guard.wait(timeout=30)
                guard_log.close()
                record['worker_fault_guard']='fault-injection.json'
        finally:
            record['down'],_=command('followup5-e2e-down',['make','-f','mk/e2e.mk','e2e-down'],env,180)
            record['inventory_after']=inventory(env)
            record['finished_at']=utc()
            write_json(PUBLIC/'followup5-run.json',record)
        components(env)
    # The record renderer keeps partial live measurements separate from the
    # complete five-case acceptance claim. Operator rows remain open.
    subprocess.run(['python3','tests/e2e/at/secops/followup5_records.py'],cwd=ROOT,check=True)
    return 2


if __name__=='__main__':
    raise SystemExit(main())
