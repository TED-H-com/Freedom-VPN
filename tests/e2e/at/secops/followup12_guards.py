#!/usr/bin/env python3
"""Verify archive provenance, live references, scope hashes and public hygiene."""
import argparse
import hashlib
import json
import subprocess
from run import ROOT, PUBLIC, scan


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("--archive-only",action="store_true",help="verify immutable archives and current scope pins without claiming live execution")
    args=parser.parse_args()
    index=json.loads((PUBLIC/'history/index.json').read_text())
    for name in ('follow_up_9','follow_up_10','follow_up_11'):
        path=PUBLIC/'history'/name.replace('_','-')
        for entry in index[name]['snapshot_files']:
            raw=(path/entry['file']).read_bytes()
            assert hashlib.sha256(raw).hexdigest()==entry['sha256'],entry['file']
            if name=='follow_up_11':
                merged=subprocess.check_output(['git','show','219599e:docs/e2e/at/secops/'+entry['file']],cwd=ROOT)
                assert raw==merged,entry['file']
            if name=='follow_up_10':
                merged=subprocess.check_output(['git','show','2f1f939:docs/e2e/at/secops/'+entry['file']],cwd=ROOT)
                assert raw==merged,entry['file']
    old=PUBLIC/'history/follow-up-9/restore-reconcile.json'
    merged=subprocess.check_output(['git','show','2f1f939:docs/e2e/at/secops/history/follow-up-9/restore-reconcile.json'],cwd=ROOT)
    assert old.read_bytes()==merged,'D-098 follow-up 9 archive moved or changed'
    scope=json.loads((PUBLIC/'scope-review.json').read_text())
    attempts=json.loads((PUBLIC/'followup12-attempts.json').read_text())
    for attempt in attempts['attempts']:
        for entry in attempt['files']:
            assert hashlib.sha256((PUBLIC/entry['file']).read_bytes()).hexdigest()==entry['sha256'],entry['file']
    prerequisites=json.loads((PUBLIC/'followup12-prerequisites.json').read_text())
    for entry in prerequisites['read_documents']:
        raw=subprocess.check_output(['git','show',entry['revision']+':'+entry['path']],cwd=ROOT)
        assert hashlib.sha256(raw).hexdigest()==entry['sha256'],entry['path']
    count=0
    for section,key in [('follow_up_12','records'),('follow_up_11','records'),('follow_up_10','records'),('handover_follow_up_6','reviewed_documents'),('follow_up_7','files'),('handover_follow_up_13','reviewed_documents')]:
        for entry in scope.get(section,{}).get(key,[]):
            assert hashlib.sha256((ROOT/entry['path']).read_bytes()).hexdigest()==entry['sha256'],entry['path']
            count+=1
    if args.archive_only:
        print(f'PASS: merged Follow-up 10/11 archives, unchanged Follow-up 9 restore and {count} current SHA-256 pins; live execution not checked')
        return
    run_ids = set()
    driver_ids = set()
    for phase in ('transports','restore'):
        run=json.loads((PUBLIC/f'followup12-{phase}-run.json').read_text())
        live=json.loads((PUBLIC/f'followup12-{phase}-live.json').read_text())
        assert run['follow_up']==live['follow_up']==12,phase
        assert run['phase']==phase and run['finished_at'],phase
        assert run['d099_roadmap_present'] is True,phase
        assert run['driver_run_id'] not in driver_ids,phase
        assert live['run_id'] not in run_ids,phase
        run_ids.add(live['run_id'])
        driver_ids.add(run['driver_run_id'])
        assert run['live']['status']=='passed',phase
        assert live['go_test_failed'] is False,phase
        assert run['source_revision']==live['source_revision'],phase
        assert run['state_cleanup']['removed'] is True,phase
        assert all(v['count']==0 and v['exit_code']==0 for v in run['inventory_after'].values()),phase
        assert not any(row['known_secret_matches'] or row['device_flow_key_lines'] or not row['read_ok'] for row in live['secret_scan']),phase
    for path in PUBLIC.iterdir():
        if path.suffix in ('.json','.md'):
            assert not scan(path.read_bytes()),path.name
    print(f'PASS: merged Follow-up 10/11 archives, unchanged Follow-up 9 restore, {count} current SHA-256 pins, two distinct live runs, cleanup and capture hygiene')


if __name__=='__main__':main()
