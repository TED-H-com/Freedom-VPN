#!/usr/bin/env python3
"""Read-only operation-reference observer for the owned assembled stack."""
import json
from pathlib import Path
import subprocess
import time
from datetime import datetime, timezone


def capture(root):
    run_id=json.loads((root/'lab-AT10.json').read_text())['run_id']
    query="""SELECT coalesce(jsonb_agg(jsonb_build_object('id',id,'kind',kind,'resource_id',resource_id,
    'desired_generation',desired_generation::text,'status',status,'steps',steps,'observed_refs',observed_refs,
    'created_at',created_at,'completed_at',completed_at) ORDER BY created_at),'[]'::jsonb)
    FROM operations WHERE resource_type='node'"""
    result=dict(run_id=run_id,scope='Read-only snapshots of actual operation steps and report references in the same owned assembled stack; no API or database mutation.',snapshots=[])
    prior=None
    while True:
        started=datetime.now(timezone.utc).isoformat().replace('+00:00','Z')
        proc=subprocess.run(['docker','exec','fvpn-w4-isolation-postgres','sh','-c',
            'exec psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "$1"','hold-operations',query],capture_output=True,text=True)
        if proc.returncode:
            result['end']='owned database unavailable; last successful snapshot retained'
            break
        ops=json.loads(proc.stdout)
        now=datetime.now(timezone.utc).isoformat().replace('+00:00','Z')
        if ops!=prior:
            result['snapshots'].append(dict(read_started=started,read_completed=now,operations=ops))
            prior=ops
        result['last_successful_read']=now
        path=root/'generation-holds-central.json'
        if path.exists() and json.loads(path.read_text())['run_id']==run_id:
            result['end']='final read after the assembled central report export'
            break
        (root/'publication-operation-history.json').write_text(json.dumps(result,indent=2)+'\n')
        time.sleep(1)
    (root/'publication-operation-history.json').write_text(json.dumps(result,indent=2)+'\n')
    print('Retained',len(result['snapshots']),'operation snapshots;',result['end'])


if __name__=='__main__':
    import sys
    capture(Path(sys.argv[1]))
