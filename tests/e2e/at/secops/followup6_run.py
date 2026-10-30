#!/usr/bin/env python3
"""Two fresh instances of the shared lab; no projected row fixtures."""
import os
import shutil
import subprocess
import tempfile
import uuid
from run import ROOT, PRIVATE, PUBLIC, PREFIX, PORT_BASE, command, inventory, summarize_go, utc, write_json


def main():
    PRIVATE.mkdir(parents=True, exist_ok=True)
    (ROOT / '.agent/tmp').mkdir(parents=True, exist_ok=True)
    head=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip()
    record={'driver_run_id':str(uuid.uuid4()),'started_at':utc(),'source_revision':head,'phases':[],
            'expected':'unchanged sequential 20x3 profile, then exported 60-peer helper on a fresh stack',
            'scope':'admission and control diagnostics, not full mixed-traffic capacity'}
    with tempfile.TemporaryDirectory(prefix='secops-f6.',dir=ROOT/'.agent/tmp') as scratch:
        env=dict(os.environ,GOTOOLCHAIN='local',CGO_ENABLED='0',PYTHONDONTWRITEBYTECODE='1',TMPDIR=scratch,GOTMPDIR=scratch,E2E_RESET='1',FVPN_E2E_PREFIX=PREFIX,FVPN_E2E_PORT_BASE=PORT_BASE,FVPN_E2E_STATE=str(ROOT/'.agent/e2e-state-w4-secops'))
        before=inventory(env)
        if any(v['count'] or v['exit_code'] for v in before.values()):
            raise SystemExit('own namespace occupied: refusing to attach or tear down')
        write_json(PUBLIC/'followup6-run.json',record)
        watcher_log=(PRIVATE/'followup6-gateway.log').open('wb')
        (PRIVATE/'followup6-gateway.log').chmod(0o600)
        watcher=subprocess.Popen(['python3','tests/e2e/at/secops/followup6_gateway.py'],cwd=ROOT,stdout=watcher_log,stderr=subprocess.STDOUT,env=env)
        try:
            for phase in ('profile','helper'):
                entry={'phase':phase,'started_at':utc(),'inventory_before':inventory(env)}
                if any(v['count'] or v['exit_code'] for v in entry['inventory_before'].values()):
                    raise SystemExit('own namespace occupied: refusing to attach or tear down')
                print('follow-up 6 '+phase+': fresh e2e-up',flush=True)
                try:
                    entry['up'],_=command('followup6-'+phase+'-e2e-up',['make','-f','mk/e2e.mk','e2e-up'],env,1500)
                    if entry['up']['exit_code']==0:
                        name='TestAT28_AT31_Followup6'
                        args=['go','test','-json','-tags','e2e,integration','-p','1','-parallel','2','-count=1','-timeout','50m','-run','^'+name+'$','./tests/e2e/at/secops']
                        cmd,raw=command('followup6-'+phase+'-live',args,dict(env,FVPN_SECOPS_FOLLOWUP6=phase),3060)
                        entry['live']=dict(cmd,**summarize_go(raw,[name],cmd['exit_code']))
                        # The generic scanner writes a fixed private capture directory.
                        # Preserve each phase separately before the next fresh instance.
                        captured=PRIVATE/'captured-logs'
                        if captured.exists():
                            dest=PRIVATE/('followup6-'+phase+'-'+record['driver_run_id']+'-logs')
                            if dest.exists():raise RuntimeError('refusing to overwrite prior private run logs')
                            shutil.move(captured,dest)
                finally:
                    entry['down'],_=command('followup6-'+phase+'-e2e-down',['make','-f','mk/e2e.mk','e2e-down'],env,300)
                    entry['inventory_after']=inventory(env)
                    entry['finished_at']=utc()
                    record['phases'].append(entry)
                    write_json(PUBLIC/'followup6-run.json',record)
                print('follow-up 6 '+phase+': '+entry.get('live',{}).get('status','blocked'),flush=True)
        finally:
            record['finished_at']=utc()
            write_json(PUBLIC/'followup6-run.json',record)
            try:
                watcher.wait(timeout=45)
            except subprocess.TimeoutExpired:
                watcher.terminate()
                watcher.wait(timeout=10)
            watcher_log.close()
    subprocess.run(["python3","tests/e2e/at/secops/followup6_records.py"],cwd=ROOT,check=True)
    # Full AT28/31 gates remain explicit even when the runnable subset succeeds.
    return 1 if any(p.get("live",{}).get("status")=="failed" for p in record["phases"]) else 2


if __name__=='__main__':
    raise SystemExit(main())
