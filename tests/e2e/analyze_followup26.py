#!/usr/bin/env python3
"""TEST ONLY: immutable serving/packet joins; preserve missing evidence as missing."""
import argparse
from datetime import datetime, timezone
from decimal import Decimal
import hashlib
import json
from pathlib import Path
from analyze_native_capture import analyze
from analyze_kernel_capture import publish, receipt
from decode_kernel_trace import decode


def notes(record):
    return {o['name']: o['detail'] for o in record.get('observations', [])}


def seconds(value):
    # PostgreSQL receipt precision is microseconds; packet timestamps are also
    # wall-clock observations. Never label this an exact kernel close instant.
    delta=datetime.fromisoformat(value.replace('Z', '+00:00'))-datetime(1970,1,1,tzinfo=timezone.utc)
    return Decimal(delta.days*86400+delta.seconds)+Decimal(delta.microseconds)/Decimal(1000000)


def relative_ms(value, syn):
    return str((seconds(value)-Decimal(syn))*1000) if syn is not None else None


def closing_reports(observations, serving_at, end):
    return [o for o in observations if seconds(serving_at) <= seconds(o['observed_at']) <= seconds(end)
            and not o['member_forwarding_enabled']]


def verify_file(path, expected):
    raw = path.read_bytes()
    receipt(raw, expected)
    assert path.stat().st_mode & 0o777 == 0o600, 'private artifact permissions'
    return raw


def main(run, private):
    assert not (run/'serving-packet-joins.json').exists(), 'already analyzed; immutable'
    central = json.loads((run/'central-record.json').read_text())
    cn = {k: json.loads(v) for k, v in notes(central).items() if k != 'producer_export_error'}
    result = dict(rows=[], receipts_verified=0, limitations=[])
    prior={}
    for row in cn.get('producer_timing',[]):
        key=row['run_id'];seq=int(row['sequence'])
        assert seq==prior.get(key,0)+1, 'producer sequence gap'
        prior[key]=seq
    result['producer_sequences']=prior
    if not prior:result['limitations'].append('No producer timing stream in this first attempt.')
    if 'producer_receipt' in cn and cn['producer_receipt'].get('sha256'):
        verify_file(private/'producer.jsonl', cn['producer_receipt']); result['receipts_verified'] += 1
    debug_path = run/'native-debug-export.json'
    if debug_path.exists():
        dn = notes(json.loads(debug_path.read_text()))
        for role in ['member', 'node']:
            expected=json.loads(dn[role])
            verify_file(private/(role+'-debug.log'), expected);result['receipts_verified'] += 1
    for identity, gate in [('initial-selection', 'admission-serving'), ('regrant-session', 'regrant-serving')]:
        path=run/(identity+'.json')
        if not path.exists():
            result['rows'].append(dict(id=identity, measured=False, reason='no measurement row; inspect admission'))
            continue
        doc=json.loads(path.read_text());n=notes(doc)
        row=dict(id=identity, status=doc['status'], measured='transfer' in n)
        if 'tracer_verifier_replay' in n:
            replay=json.loads(n['tracer_verifier_replay'])
            verify_file(private/'traced-node-ruleset.raw',replay);result['receipts_verified']+=1
            row['tracer_verifier_replay']=replay
        result['rows'].append(row)
        if 'capture' in n:
            cap=json.loads(n['capture'])
            for role, point in cap['points'].items():
                verify_file(private/identity/(role+'.pcap'), point['private_pcap_receipt']);result['receipts_verified'] += 1
        else:
            row['reason']=n.get('prerequisite','capture unavailable');continue
        if not row['measured']:
            row['reason']='request not launched; retain prerequisite/capture failure';continue
        capture=analyze(run,path)
        row['capture']=capture
        before=next(s for s in cap['samples'] if s['label']=='before-first-transfer')
        syn=capture['member_syns'][0] if capture['member_syns'] else None
        stamp=syn['timestamp'] if syn else None
        row['first_syn']=syn
        row['syns']=len(capture['member_syns'])
        row['node_syns']=len([p for p in capture['hops']['node'] if p['interface']=='fvmb0' and p['tcp']['src_port']==20001 and p['tcp']['flags']==2 and p['packet_type']!=4])
        row['origin_packets']=len(capture['hops']['origin'])
        row['origin_receipt']=json.loads(n['origin_arrival'])
        row['transfer']=json.loads(n['transfer'])
        row['session_samples']=[dict(label=s['label'],points={role:{k:s['points'][role].get(k) for k in ['start','end','peer_present','peer_status','connected','wireguard_peer_matched','wireguard_endpoint','latest_handshake_unix_seconds','native_latest_handshake_at','uses_kernel_interface','local','remote','routes']} for role in ['member','node']}) for s in cap['samples']]
        gp=run/(gate+'.json')
        if gp.exists():
            gn=notes(json.loads(gp.read_text())); row['serving_gate']=dict(status=json.loads(gp.read_text())['status'],completed_at=json.loads(gn['completed_at']),accepted=json.loads(gn.get('accepted','null')))
            closes=closing_reports(cn['observations'],json.loads(gn['completed_at']),cap['points']['node']['stopped_after']['wall'])
        else:closes=[]
        row['closed_reports_in_capture']=[dict(o,reported_relative_to_first_syn_ms=relative_ms(o['observed_at'],stamp)) for o in closes]
        start=int(cap['points']['node']['launch_before']['mono_before_ns']); end=int(cap['points']['node']['stopped_after']['mono_after_ns'])
        row['producer_baselines_in_capture']=[]
        for p in cn.get('producer_timing',[]):
            b=p.get('baseline')
            if b and start<=int(b['start_ns'])<=end:
                v=dict(p)
                if syn:v['relative_to_first_syn_ns']={'lower':str(int(b['start_ns'])-int(syn['mono_upper_ns'])),'upper':str(int(b['end_ns'])-int(syn['mono_lower_ns']))}
                row['producer_baselines_in_capture'].append(v)
        if 'kernel_trace' in n:
            trace=json.loads(n['kernel_trace']);row['kernel_trace']={}
            for role, info in trace['roles'].items():
                candidates=[private/'kernel'/identity, private/'kernel']
                if identity=='initial-selection':candidates.insert(0,private/'kernel-initial-retained')
                found=None
                for folder in candidates:
                    p=folder/(role+'-trace.txt')
                    if p.exists() and hashlib.sha256(p.read_bytes()).hexdigest()==info['trace_receipt']['sha256']:found=folder;break
                assert found is not None, 'missing original trace receipt'
                raw=verify_file(found/(role+'-trace.txt'),info['trace_receipt']);result['receipts_verified']+=1
                errors=verify_file(found/(role+'-trace-errors.txt'),info['error_receipt']);result['receipts_verified']+=1
                parsed=decode(raw)
                assert not errors and not parsed['unparsed_lines'], 'trace error or undecoded lines'
                row['kernel_trace'][role]=parsed
        row['reported_cause_status']='accepted report plus producer timing when present; exact dropping rule is not inferred'
    result['scope']='Central rows are accepted reports, not continuous node state. Producer baseline brackets are direct local writer observations; packet clocks retain measured offset bounds. Fresh reproduction never rewrites historical attribution. F14-01 remains distinct from first selection.'
    publish(run/'serving-packet-joins.json',central,'serving-packet-joins','Follow-up 26 central, baseline and native session joins',result)
    return result

if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('run',type=Path);p.add_argument('private',type=Path);a=p.parse_args();main(a.run,a.private)
