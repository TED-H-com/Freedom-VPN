#!/usr/bin/env python3
"""TEST ONLY direct stop observations plus read-only transaction-stage brackets."""
from datetime import datetime
import hashlib
import json
from pathlib import Path
import sqlite3
import subprocess
import tempfile
import os

ROOT = Path(__file__).resolve().parents[4]
EVIDENCE = ROOT / 'docs/e2e/at/updates'
BASE = EVIDENCE / 'history/follow-up-12-stop-journal'


def read(container, path):
    p = subprocess.run(['docker', 'exec', container, 'cat', path], capture_output=True, text=True, check=False)
    return p.returncode, p.stdout


def stages(db):
    return [dict(zip(('txn_id', 'seq', 'stage', 'at'), r)) for r in db.execute('SELECT txn_id,seq,stage,at FROM txn_events ORDER BY at,seq')]


def milliseconds(a, b):
    return round((datetime.fromisoformat(b.replace('Z', '+00:00')) - datetime.fromisoformat(a.replace('Z', '+00:00'))).total_seconds()*1000, 3)


def brackets(events):
    groups = {}
    for row in events:
        groups.setdefault(row['txn_id'], []).append(row)
    rows = []
    for txn, group in groups.items():
        for i, row in enumerate(group[:-1]):
            nxt = group[i+1]
            if (row['stage'], nxt['stage']) not in (('final_sampled', 'stopped'), ('stopped', 'migrated'), ('rolling_back', 'rolling_back')):
                continue
            rows.append(dict(txn_id=txn, stage=row['stage'], start=row['at'], end=nxt['at'], stage_bracket_ms=milliseconds(row['at'], nxt['at']), command_duration_ms=None, outcome='stop returned success' if row['stage']!='rolling_back' else 'rollback completed; Stop error is discarded by product', termination='not recorded'))
    return rows


def main():
    rec = json.loads((EVIDENCE/'AT23.json').read_text())
    result = dict(run_id=rec['run_id'], source_commit=rec['git'], scope='TEST ONLY service-command observation; no generated config, deadline or product binary change', baseline_run='2babf9b6-86a3-489d-a678-0bd447f2f8b0', baseline_direct_timing_available=False, rows=[], journals={}, findings=[], gaps=[])
    baseline = json.loads((BASE/'stages.json').read_text())
    result['baseline_pins'] = {str((BASE/'stages.json').relative_to(ROOT)): hashlib.sha256((BASE/'stages.json').read_bytes()).hexdigest()}
    for role in ('installer-node', 'node', 'node2'):
        container = 'fvpn-w4-updates-'+role
        code, raw = read(container, '/var/lib/vpn-node/e2e/stop-observation.jsonl')
        observations = [json.loads(line) for line in raw.splitlines()] if code==0 else []
        with tempfile.TemporaryDirectory(prefix='stop-journal-', dir=os.environ['TMPDIR']) as scratch:
            path=Path(scratch)/'journal.db'
            p=subprocess.run(['docker','cp',container+':/var/lib/vpn-node/update-journal/journal.db',str(path)],capture_output=True,check=False)
            if p.returncode==0:
                subprocess.run(['docker','cp',container+':/var/lib/vpn-node/update-journal/journal.db-wal',str(path)+'-wal'],capture_output=True,check=False)
                with sqlite3.connect(path) as db:
                    result['journals'][role]=stages(db)
        previous=brackets(baseline.get(role, []))
        current=brackets(result['journals'].get(role, []))
        stops=[r for r in observations if r['action']=='stop']
        for i, row in enumerate(stops):
            row.update(role=role, ordinal=i+1, duration_ms=round(row['duration_ns']/1000000,3))
            row['baseline']=previous[i] if i<len(previous) else dict(command_duration_ms=None, stage_bracket_ms=None, outcome='success inferred from archived installer success', termination='not recorded')
            txn=row['transaction']
            match=[r for r in current if r['txn_id']==txn.get('txn_id') and r['stage']==txn.get('stage')]
            row['current_stage_bracket']=match[0] if match else None
            sd=dict(line.split('=',1) for line in row.get('systemd_after','').splitlines() if '=' in line)
            wait=row['agent_wait_status']
            if row['agent_pid_before'] in ('','0'):
                termination='already stopped'
            elif sd.get('ExecMainCode')=='1':
                termination='exited'; row['agent_exit_code']=int(sd['ExecMainStatus'])
            elif sd.get('ExecMainCode') in ('2','3'):
                termination='killed'; row['agent_signal']=int(sd['ExecMainStatus'])
            elif wait>=0:
                termination='killed' if wait&127 else 'exited'; row['agent_exit_code']=wait>>8; row['agent_signal']=wait&127
            else:
                termination='unobserved'
            row['termination']=termination
            row['command_timeout_observed']=(row['exit_code']!=0 or sd.get('Result')=='timeout')
            row['lease_lost']=row['lease_before'].get('present') is True and row['lease_after'].get('present') is not True
            if row['command_timeout_observed'] or row['lease_lost'] or termination=='killed':
                result['findings'].append(dict(role=role, ordinal=i+1, owner='agent-core / updater stop handling', reason='stop error, killed agent, or lost stored lease', row_status='failed'))
            if termination=='unobserved':
                result['gaps'].append(f'{role} stop {i+1}: process wait status not observed')
            result['rows'].append(row)
        result.setdefault('start_observations',{})[role]=[r for r in observations if r['action']=='start']
        expected=len(current) if role!='installer-node' else 2
        if len(stops)!=expected or not stops:
            result['findings'].append(dict(role=role,owner='updates harness',reason=f'stop observation inventory {len(stops)} differs from expected {expected}',row_status='failed'))
    result['gaps'].append('Run 2babf9b6 has no direct stop command durations, timeouts or wait statuses. Regular retained journal brackets are comparisons of larger stages, not exact command durations; installer journal was not retained after teardown.')
    result['status']='failed' if result['findings'] else 'measured with historical comparison gap'
    (EVIDENCE/'agent-stops.json').write_text(json.dumps(result,indent=2)+'\n')
    print(f"agent stops: {len(result['rows'])}, findings={len(result['findings'])}, gaps={len(result['gaps'])}")
    return int(bool(result['findings']))


if __name__=='__main__':
    raise SystemExit(main())
