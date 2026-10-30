#!/usr/bin/env python3
"""Read-only simultaneous cohort observation; only the assigned lab database."""
import datetime as dt
import json
import subprocess
import time
from run import PUBLIC, write_json

QUERY = """SELECT json_build_object(
 'authorized_users',(SELECT count(DISTINCT member_id) FROM member_devices WHERE status='authorized'),
 'authorized_devices',(SELECT count(*) FROM member_devices WHERE status='authorized'),
 'current_binding_devices',(SELECT count(DISTINCT b.device_id) FROM device_bindings b JOIN member_devices d ON d.id=b.device_id WHERE d.status='authorized' AND b.valid_from<=now() AND b.valid_until>now()),
 'finite_active_members',(SELECT count(*) FROM entitlements e JOIN members m ON m.id=e.member_id WHERE m.status='active' AND e.valid_from<=now() AND e.effective_service_end>now() AND isfinite(e.effective_service_end)),
 'native_identity_members',(SELECT count(DISTINCT member_id) FROM netbird_user_bindings)
)"""


def main():
    command=['docker','exec','fvpn-w4-secops-postgres','psql','-U','fvpn','-d','fvpn','-Atc',QUERY]
    record={'expected':{'authorized_users':20,'authorized_devices':60,'current_binding_devices':60,'finite_active_members':20,'native_identity_members':20},'command':command,'status':'blocked','scope':'one SQL snapshot of product projections; no writes, no serving/capacity claim','evidence':'native-scale.json'}
    start=time.monotonic()
    while time.monotonic()-start<420:
        result=subprocess.run(command,capture_output=True,text=True,timeout=10,check=False)
        record['exit_code']=result.returncode
        record['last_attempt_utc']=dt.datetime.now(dt.timezone.utc).isoformat().replace('+00:00','Z')
        if result.returncode:
            record['reason']='own database no longer reachable; no substitute used'
            break
        record['actual']=json.loads(result.stdout)
        record['last_successful_observation_utc']=record['last_attempt_utc']
        if record['actual']==record['expected']:
            record['status']='passed'
            break
        time.sleep(2)
    record['poll_seconds']=round(time.monotonic()-start,3)
    write_json(PUBLIC/'native-cohort.json',record)
    print('simultaneous native cohort: '+record['status'])
    return 0 if record['status']=='passed' else 2


if __name__=='__main__':
    raise SystemExit(main())
