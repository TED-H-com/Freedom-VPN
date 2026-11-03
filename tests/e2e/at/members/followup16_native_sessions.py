"""Decode original full-population regrant and AT02 sibling captures read-only."""
from pathlib import Path
from collections import defaultdict
from decimal import Decimal
import hashlib
import json
import subprocess
import sys

from evidence_io import load, publish

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path.cwd()/'tests/e2e'))
from analyze_native_capture import offsets, bound, wall_ns
from analyze_followup27 import handshake_groups

ROOT = Path('docs/e2e/at/members')


def write(path, value):
    path.write_text(json.dumps(value, indent=2)+'\n')
    publish(path)


def main():
    cases = {c:load(ROOT/(c+'.json')) for c in ['AT01','AT02','AT03']}
    run = cases['AT02']['run_id']
    base = ROOT/'history/follow-up-16-captures'/run
    base.mkdir(parents=True,exist_ok=True)
    private = Path('.agent/tmp/f16-native')/run
    entries=[]
    for case, doc in cases.items():
        for step in doc['steps']:
            if step['name'].startswith('native capture completeness regrant-') or step['name']=='native capture completeness AT02-sibling':
                cap=json.loads(step['actual'])
                entries.append((case,step,cap))
    assert len(entries)>=4, 'missing original regrant or sibling capture'
    results=[]
    for case,step,cap in entries:
        doc=cases[case]
        label=step['name'].removeprefix('native capture completeness ')
        name=case+'-'+label
        destination=base/'headers'/name
        directory=next(p.parent for p in private.glob('*/capture.json') if json.loads(p.read_text())['id']==cap['id'])
        roles=sorted(cap['points'])
        for role in roles:
            raw=(directory/(role+'.pcap')).read_bytes();pin=cap['points'][role]['private_pcap_receipt']
            assert str(len(raw))==pin['bytes'] and hashlib.sha256(raw).hexdigest()==pin['sha256']
        decoded=json.loads(subprocess.check_output(['python3','-B','tests/e2e/decode_capture.py','--stable-addresses','--wireguard-headers','--fixture-sequences','--out',str(destination)]+[str(directory/(r+'.pcap')) for r in roles],text=True))
        for d in decoded:d['file']=Path(d['file']).name
        packets,clocks,checks={},{},{}
        for role,d in zip(roles,decoded):
            p=cap['points'][role]
            raw=[json.loads(line) for line in (destination/(role+'.headers.jsonl')).read_text().splitlines()]
            complete=p['statistics_ok'] and p['process_stopped'] and p['kernel_drops']==0 and p['captured']==d['records']==len(raw) and d['header_errors']==d['snap_truncated_records']==0 and d['file_complete']
            checks[role]=dict(complete=complete,records=len(raw),kernel_drops=p['kernel_drops'],decoder=d)
            clocks[role]=offsets(cap,role)
            indexes={link['ifindex']:link['ifname'] for s in cap['samples'] for link in s['points'][role]['links'] or []}
            packets[role]=[dict(row,**bound(row,clocks[role]),interface=indexes.get(row.get('interface_index'),'unknown')) for row in raw]
        groups=handshake_groups(packets)
        loopback=[]
        for role in ['member','node']:
            for p in packets[role]:
                wg=p.get('wireguard',{})
                if wg.get('source_loopback'):
                    key=wg.get('message_sha256')
                    joins=[dict(role=other,packet=q['packet'],interface=q['interface'],timestamp=q['timestamp'],wireguard=q['wireguard']) for other in ['member','node'] for q in packets[other] if key and q.get('wireguard',{}).get('message_sha256')==key]
                    loopback.append(dict(role=role,packet=p,matching_message_hops=joins))
        result=dict(case=case,capture_id=cap['id'],label=label,original_completeness_step=step,capture_checks=checks,clock_offsets=clocks,handshake_groups=groups,loopback_source_packets=loopback,cause_status='unproven',scope='Structural WireGuard headers and opaque hashes are not cryptographic authentication or an exact kernel drop callsite. Full original periodic samples retain independent WG latest-handshake, peer state, routes and IPv6 counters. No node nft observer was installed.')
        result['sample_index']=[dict(label=s['label'],points={r:{k:s['points'][r].get(k) for k in ['start','end','latest_handshake_unix_seconds','wireguard_peer_matched','wireguard_endpoint','local','remote','connected','routes','kernel']} for r in ['member','node']}) for s in cap['samples']]
        requests=[]
        for sample in cap['samples']:
            if not sample['label'].startswith('before-transfer-'):continue
            port=int(sample['label'].removeprefix('before-transfer-'))
            lo=wall_ns(sample['points']['member']['start']['wall'])
            after=next(s for s in cap['samples'] if s['label']=='after-transfer-'+str(port))
            hi=wall_ns(after['points']['member']['end']['wall'])
            syns={}
            for role in roles:
                syns[role]=[p for p in packets[role] if p.get('tcp',{}).get('src_port')==port and p['tcp']['flags']==2 and int(Decimal(p['timestamp'])*10**9)>=lo and (role=='origin' or role=='member' and p['interface']=='fvnb0' and p['packet_type']==4 or role=='node' and p['interface']=='fvmb0' and p['packet_type']!=4)]
            first=syns['member'][0] if syns['member'] else None
            match=next((p for p in syns['node'] if first and p['tcp']==first['tcp']),None)
            transit=dict(lower_ns=str(int(match['mono_lower_ns'])-int(first['mono_upper_ns'])),upper_ns=str(int(match['mono_upper_ns'])-int(first['mono_lower_ns']))) if match else None
            # Port belongs to one original transfer; don't borrow an earlier receipt.
            def matches_port(observation):
                return any(p.get('filter')=='controlled-origin TCP/443, client port '+str(port) for p in observation.get('packet_observations',{}).values())
            forwarding=[f for f in doc['forwarding'] if matches_port(f)]
            retained=[]
            for original_step in doc['steps']:
                if original_step['name']!='F14-01 original re-grant attempt retained':continue
                attempt=json.loads(original_step['actual'])
                if any(matches_port(f) for f in attempt['forwarding_attempts']):
                    retained.append(dict(step_name=original_step['name'],step_at=original_step['at'],step_status=original_step['status'],device_id=attempt['device_id'],cause_status=attempt['cause_status']))
            requests.append(dict(source_port=port,before_sample=sample['label'],after_sample=after['label'],request_window_start_wall_ns=str(lo),request_window_end_wall_ns=str(hi),syns=syns,first_syn_transit=transit,original_forwarding_matches=forwarding,unique_original_filter_join=len(forwarding)==1,retained_regrant_attempts=retained,scope='Window includes command/readback overhead. Exact client-port AF_PACKET filter joins the original forwarding record and its authenticated origin receipt; TCP headers alone are not HTTP evidence. Setup/negative transfers remain distinct from the retained original regrant attempt.'))
        result['requests']=requests
        if label=='AT02-sibling':
            sender=json.loads(next(s for s in cases['AT02']['steps'] if s['name']=='D-135 continuous admitted sibling capture')['actual'])['sender']
            observed={r:defaultdict(list) for r in roles}
            for role in roles:
                for p in packets[role]:
                    if 'fixture_sequence' not in p:continue
                    if role=='member' and not(p['interface']=='fvnb0' and p['packet_type']==4):continue
                    if role=='node' and not(p['interface']=='fvmb0' and p['packet_type']!=4):continue
                    observed[role][p['fixture_sequence']].append(p)
            joins=[]
            for a in sender['sends']:
                seq=a['sequence'];nd=observed['node'].get(seq,[])
                joins.append(dict(send=a,hops={r:observed[r].get(seq,[]) for r in roles},send_to_node_bounds=dict(lower_ns=str(int(nd[0]['mono_lower_ns'])-int(a['mono_after_ns'])),upper_ns=str(int(nd[0]['mono_upper_ns'])-int(a['mono_before_ns']))) if nd else None))
            result['every_send_packet_join']=joins
            result['missing_sequences_by_point']={r:[a['sequence'] for a in sender['sends'] if a['sequence'] not in observed[r]] for r in roles}
            result['denominator']=len(sender['sends'])
        write(base/(name+'-original-capture.json'),cap)
        write(base/(name+'-analysis.json'),result)
        results.append(dict(case=case,name=name,capture_id=cap['id'],capture_checks=checks,loopback_packets=len(loopback),request_count=len(requests),missing_sequences_by_point=result.get('missing_sequences_by_point')))
        for f in destination.glob('*'):publish(f)
        print(name,'complete',all(c['complete'] for c in checks.values()),'loopback',len(loopback),'requests',len(requests))
    write(base/'session-index.json',dict(run_id=run,source_commit=cases['AT02']['source_commit'],rows=results,decoder_sha256=hashlib.sha256(Path('tests/e2e/decode_capture.py').read_bytes()).hexdigest()))
    files=sorted(p for p in base.rglob('*') if p.is_file() and p.name!='SHA256SUMS')
    (base/'SHA256SUMS').write_text(''.join(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+str(p.relative_to(base))+'\n' for p in files))

if __name__=='__main__':main()
