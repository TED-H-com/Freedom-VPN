#!/usr/bin/env python3
"""TEST ONLY direct stop observations plus read-only transaction-stage brackets."""
from datetime import datetime
import hashlib
import io
import tarfile
import json
from pathlib import Path
import sqlite3
import subprocess
import tempfile
import os

ROOT = Path(__file__).resolve().parents[4]
EVIDENCE = ROOT / 'docs/e2e/at/updates'
BASE = EVIDENCE / 'history/follow-up-13'


def read(container, path):
    # Works for a running service or a stopped whole container. docker exec
    # cannot read a stopped container; retain the launch ID before stopping.
    p = subprocess.run(['docker', 'cp', container+':'+path, '-'], capture_output=True, check=False)
    if p.returncode:
        return p.returncode, ''
    with tarfile.open(fileobj=io.BytesIO(p.stdout)) as archive:
        files = [m for m in archive.getmembers() if m.isfile()]
        if len(files) != 1:
            return 1, ''
        return 0, archive.extractfile(files[0]).read().decode()


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
    result = dict(run_id=rec['run_id'], source_commit=rec['git'], scope='TEST ONLY service-command observation; no generated config, deadline or product binary change', baseline_run='759d244a-2dc4-460c-9f82-e37ce0ac7560', baseline_direct_timing_available=True, rows=[], journals={}, findings=[], gaps=[])
    baseline = json.loads((BASE/'agent-stops.json').read_text())
    result['baseline_pins'] = {str((BASE/'agent-stops.json').relative_to(ROOT)): hashlib.sha256((BASE/'agent-stops.json').read_bytes()).hexdigest()}
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
        previous=[r for r in baseline['rows'] if r['role']==role]
        current=brackets(result['journals'].get(role, []))
        stops=[r for r in observations if r['action']=='stop']
        for i, row in enumerate(stops):
            row.update(role=role, ordinal=i+1, duration_ms=round(row['duration_ns']/1000000,3))
            row['baseline']=previous[i] if i<len(previous) else None
            txn=row['transaction']
            match=[r for r in current if r['txn_id']==txn.get('txn_id') and r['stage']==txn.get('stage')]
            row['current_stage_bracket']=match[0] if match else None
            sd=dict(line.split('=',1) for line in row.get('systemd_after','').splitlines() if '=' in line)
            identity=row.get('agent_identity_before', {})
            receipt=row.get('parent_wait_receipt') or row.get('systemd_parent_wait_receipt')
            row['new_exit_observed']=False
            if receipt and not row.get('identity_error') and not row.get('receipt_error'):
                row['agent_exit_code']=receipt['exit_code']
                row['agent_signal']=receipt['signal']
                row['termination']='already stopped' if identity.get('already_stopped') else ('killed' if receipt['signal'] is not None else 'exited')
                row['new_exit_observed']=not identity.get('already_stopped')
                if role != 'installer-node':
                    # Read the exact retained ID again after every later start,
                    # even when the current ID names a different child.
                    code, raw=read(container, '/var/lib/vpn-node/e2e/agent-exits/'+identity['id']+'.json')
                    row['receipt_retained_after_restarts']=code==0 and json.loads(raw)==receipt
                    if not row['receipt_retained_after_restarts']:
                        result['findings'].append(dict(role=role, ordinal=i+1, owner='devstack', reason='retained parent receipt changed or missing', row_status='failed'))
            else:
                row['termination']='unobserved'
                result['gaps'].append(f'{role} stop {i+1}: matching parent receipt unavailable')
            row['command_failed']=row['exit_code']!=0
            row['command_timeout_observed']=sd.get('Result')=='timeout'
            row['reached_15s_budget']=row['duration_ns']>=15000000000
            row['lease_lost']=row['lease_before'].get('present') is True and row['lease_after'].get('present') is not True
            reasons=[]
            if row['command_failed'] or row['command_timeout_observed'] or row['reached_15s_budget']:
                reasons.append(('updater', 'stop failed, timed out, or reached 15 s budget'))
            if row.get('agent_signal') is not None or row.get('agent_exit_code', 0) not in (0, None):
                reasons.append(('agent-core', 'agent terminated by signal or nonzero exit'))
            if row.get('sigkill_attempt'):
                reasons.append(('agent-core / updater', 'supervisor attempted SIGKILL; actual outcome is the separate parent wait receipt'))
            if row['lease_lost']:
                reasons.append(('agent-core', 'stored lease lost across stop'))
            for owner, reason in reasons:
                result['findings'].append(dict(role=role, ordinal=i+1, owner=owner, reason=reason, row_status='failed'))
            row['status']='failed' if reasons else ('unobserved' if row['termination']=='unobserved' else 'measured')
            result['rows'].append(row)
        starts=[r for r in observations if r['action']=='start']
        result.setdefault('start_observations',{})[role]=starts
        for row in starts:
            if row['lease_before'].get('present') is True and row['lease_after'].get('present') is not True:
                result['findings'].append(dict(role=role, owner='agent-core', reason='stored lease lost after restart', row_status='failed'))
            retained=row.get('parent_wait_receipt') or row.get('retained_systemd_parent_wait_receipt')
            matching=[r for r in stops if r.get('agent_identity_before',{}).get('id')==row.get('agent_identity_before',{}).get('id')]
            if not retained or not matching or retained != (matching[-1].get('parent_wait_receipt') or matching[-1].get('systemd_parent_wait_receipt')):
                result['gaps'].append(f'{role} start: prior stop receipt not read back unchanged')
        expected=len(current) if role!='installer-node' else 2
        if len(stops)!=expected or not stops:
            result['findings'].append(dict(role=role,owner='updates harness',reason=f'stop observation inventory {len(stops)} differs from expected {expected}',row_status='failed'))
    result['historical_gap']='Updates F13-01: run 759d244a AT23 first exit stays unobserved; the new receipt is a new sample, never a retrospective fill.'
    result['systemd_scope']='Installer uses systemd parent waitid code/status and its UTC exit observation; encoded_wait_status is derived, sampled_raw_wait_status is an independent kernel sample. Devstack parent receipts apply to regular nodes.'
    result['status']='failed' if result['findings'] else ('incomplete' if result['gaps'] else 'measured')
    (EVIDENCE/'agent-stops.json').write_text(json.dumps(result,indent=2)+'\n')
    print(f"agent stops: {len(result['rows'])}, findings={len(result['findings'])}, gaps={len(result['gaps'])}")
    return int(bool(result['findings'] or result['gaps']))


if __name__=='__main__':
    raise SystemExit(main())
