#!/usr/bin/env python3
"""TEST ONLY immutable first-contact and stream joins. No transport repair."""
import argparse
from collections import Counter, defaultdict
import csv
from decimal import Decimal
import json
from pathlib import Path

from analyze_followup26 import notes, verify_file
from analyze_kernel_capture import publish
from analyze_native_capture import bound, offsets, wall_ns
from decode_kernel_trace import decode as trace_decode
from decode_native_debug import decode as debug_decode
from followup27_io import read


def sequence_ranges(values):
    out = []
    for value in sorted(set(values)):
        if out and value == out[-1][1] + 1:
            out[-1][1] = value
        else:
            out.append([value, value])
    return out


def interval(send, packet):
    """Bounds include the whole socket write; do not clamp negative lower bounds."""
    return dict(lower_ns=str(int(packet['mono_lower_ns'])-int(send['after']['mono_after_ns'])),
                upper_ns=str(int(packet['mono_upper_ns'])-int(send['before']['mono_before_ns'])))


def stamp(packet):
    return int(Decimal(packet['timestamp'])*10**9)


def handshake_groups(packets):
    groups = {}
    for role in ('member', 'node'):
        for p in packets[role]:
            wg = p.get('wireguard', {})
            if wg.get('type') not in (1, 2):
                continue
            key = wg['message_sha256']
            item = groups.setdefault(key, dict(wireguard=wg, hops=[]))
            item['hops'].append(dict(role=role, source_loopback=wg.get('source_loopback'), destination_loopback=wg.get('destination_loopback'), **{k:p.get(k) for k in
                ('packet','timestamp','mono_lower_ns','mono_upper_ns','interface','packet_type','ip_version','src','dst','udp')}))
    return sorted(groups.values(), key=lambda g:min(stamp(h) for h in g['hops']))


def initiation_origin(group):
    # SLL2 marks local loopback copies as incoming too. The pinned harness uses
    # local WG port 51820; relay-delivered loopback copies must not become a
    # second local initiation. Underlay outgoing direction is unambiguous.
    local = {'addr-12ca17b49af22894', 'addr-eff8e7ca506627fe'}  # 127.0.0.1, ::1
    outgoing = [h for h in group['hops'] if h['packet_type']==4 or
                (h['interface']=='lo' and h['src'] in local and h['udp']['src_port']==51820)]
    return min(outgoing,key=stamp) if outgoing else None


def main(run, private):
    destination = run/'measurement.json'
    assert not destination.exists(), 'immutable measurement already exists'
    source = json.loads(read(run/'native-session.json'))
    n = notes(source)
    capture = json.loads(n['capture'])
    decoded = json.loads(n['decode'])
    packets, checks, clocks = {}, {}, {}
    verified = 0
    for role, decode in zip(sorted(capture['points']), decoded):
        p = capture['points'][role]
        verify_file(private/'native-session'/(role+'.pcap'),p['private_pcap_receipt'])
        verified += 1
        rows = [json.loads(line) for line in read(run/'headers/native-session'/(role+'.headers.jsonl')).splitlines()]
        complete = p['statistics_ok'] and p['process_stopped'] and p['kernel_drops']==0 and p['captured']==decode['records']==len(rows) and decode['header_errors']==0 and decode['snap_truncated_records']==0 and decode['file_complete']
        checks[role] = dict(complete=complete, captured=p['captured'], kernel_drops=p['kernel_drops'], header_errors=decode['header_errors'], truncations=decode['snap_truncated_records'])
        assert complete, f'{role}: incomplete capture; retain original instead of attributing absence'
        clocks[role] = offsets(capture,role)
        indexes = {link['ifindex']:link['ifname'] for s in capture['samples'] for link in s['points'][role]['links'] or []}
        packets[role] = [dict(row, **bound(row,clocks[role]), interface=indexes.get(row.get('interface_index'),'unknown')) for row in rows]
    initial = json.loads((run/'initial-selection.json').read_text())
    initial_notes = notes(initial)
    transfer_start = next(s for s in capture['samples'] if s['label']=='before-first-transfer')
    start = wall_ns(transfer_start['points']['member']['start']['wall'])
    syns = {role:[p for p in rows if p.get('tcp',{}).get('src_port')==20001 and p['tcp']['flags']==2 and stamp(p)>=start and
                  (role=='origin' or (role=='member' and p['interface']=='fvnb0' and p['packet_type']==4) or
                   (role=='node' and p['interface']=='fvmb0' and p['packet_type']!=4))] for role,rows in packets.items()}
    first = syns['member'][0] if syns['member'] else None
    match = next((p for p in syns['node'] if first and p['tcp']==first['tcp']),None)
    transit = dict(member_packet=first['packet'],node_packet=match['packet'],lower_ns=str(int(match['mono_lower_ns'])-int(first['mono_upper_ns'])),upper_ns=str(int(match['mono_upper_ns'])-int(first['mono_lower_ns']))) if match else None
    groups = handshake_groups(packets)
    member_inits = []
    for g in groups:
        if g['wireguard']['type']!=1:continue
        first_hop = initiation_origin(g)
        if first_hop is None or first_hop['role']!='member':continue
        responses = [r for r in groups if r['wireguard']['type']==2 and r['wireguard']['receiver_index']==g['wireguard']['sender_index']]
        member_inits.append(dict(first=first_hop,sender_index=g['wireguard']['sender_index'],message_sha256=g['wireguard']['message_sha256'],response_hashes=[r['wireguard']['message_sha256'] for r in responses]))
    member_inits.sort(key=lambda x:stamp(x['first']))
    for i in range(1,len(member_inits)):
        member_inits[i]['since_previous_initiation_ns']=str(stamp(member_inits[i]['first'])-stamp(member_inits[i-1]['first']))

    stream_path = run/'member-stream.json'
    stream_record = json.loads(read(stream_path)) if stream_path.exists() or stream_path.with_suffix('.json.gz').exists() else None
    sends = json.loads(notes(stream_record).get('sends','[]')) if stream_record else []
    stream_packets = {role:defaultdict(list) for role in packets}
    for role,rows in packets.items():
        for p in rows:
            if 'fixture_sequence' not in p:continue
            if role=='member' and not(p['interface']=='fvnb0' and p['packet_type']==4):continue
            if role=='node' and not(p['interface']=='fvmb0' and p['packet_type']!=4):continue
            stream_packets[role][int(p['fixture_sequence'])].append(p)
    missing, absent_member, intervals = [], [], []
    with (run/'stream-intervals.csv').open('x',newline='') as f:
        writer = csv.DictWriter(f,fieldnames=['sequence','sent','send_mono_lower_ns','send_mono_upper_ns','member_packet','node_packet','origin_packet','lower_ns','upper_ns'])
        writer.writeheader()
        for send in sends:
            seq = int(send['sequence'])
            assert seq == len(intervals), 'sender sequence gap'
            assert send['before']['boot_id']==send['after']['boot_id']==clocks['node']['boot_id'], 'different monotonic domains'
            assert int(send['before']['mono_before_ns'])<=int(send['after']['mono_after_ns'])
            m,nd,o = [stream_packets[role].get(seq,[]) for role in ('member','node','origin')]
            i = interval(send,nd[0]) if nd else dict(lower_ns=None,upper_ns=None)
            row = dict(sequence=str(seq),sent=send['sent'],send_mono_lower_ns=send['before']['mono_before_ns'],send_mono_upper_ns=send['after']['mono_after_ns'],member_packet=m[0]['packet'] if m else None,node_packet=nd[0]['packet'] if nd else None,origin_packet=o[0]['packet'] if o else None,**i)
            writer.writerow(row);intervals.append(row)
            if not nd:missing.append(seq)
            if not m:absent_member.append(seq)
    ingress = sorted([p for rows in stream_packets['node'].values() for p in rows],key=stamp)
    gaps = []
    for before,after in zip(ingress,ingress[1:]):
        duration = stamp(after)-stamp(before)
        if duration<10**9:continue
        lo,hi=stamp(before),stamp(after)
        in_window=[s for s in sends if lo<=wall_ns(s['before']['wall'])<=hi]
        gaps.append(dict(previous=before,next=after,duration_ns=str(duration),sends=len(in_window),missing_sent_sequences=[s['sequence'] for s in in_window if int(s['sequence']) in missing],handshake_hashes=[g['wireguard']['message_sha256'] for g in groups if any(lo-10**9<=stamp(h)<=hi+10**9 for h in g['hops'])]))
    upper = sorted(int(x['upper_ns']) for x in intervals if x['upper_ns'] is not None)
    delays = [x for x in intervals if x['upper_ns'] is not None and int(x['upper_ns'])>10**9]
    initial_result = dict(status=initial['status'],transfer=json.loads(initial_notes.get('transfer','null')),origin_receipt=json.loads(initial_notes.get('origin_arrival','null')),member_syns=len(syns['member']),node_syns=len(syns['node']),origin_syns=len(syns['origin']),first_syn=first,first_syn_transit=transit)
    trace = json.loads(initial_notes.get('kernel_trace','{}'))
    assert set(trace.get('roles',{}))<= {'member'}, 'measured node tracer forbidden'
    trace_results = {}
    for role,info in trace.get('roles',{}).items():
        raw = verify_file(private/'kernel/initial-selection'/(role+'-trace.txt'),info['trace_receipt']);verified+=1
        errors = verify_file(private/'kernel/initial-selection'/(role+'-trace-errors.txt'),info['error_receipt']);verified+=1
        assert not errors and info['removed'] and info['stopped']
        trace_results[role]=trace_decode(raw)
        assert trace_results[role]['unparsed_lines']==0
    debug = {}
    debug_notes = notes(json.loads((run/'native-debug-export.json').read_text()))
    for role in ('member','node'):
        raw=verify_file(private/(role+'-debug.log'),json.loads(debug_notes[role]));verified+=1
        debug[role]=debug_decode(raw)
    central_record=json.loads((run/'central-record.json').read_text());cn=notes(central_record)
    central={k:json.loads(v) for k,v in cn.items() if k in ('observations','leases','desired','producer_timing')}
    if 'producer_receipt' in cn:
        verify_file(private/'producer.jsonl',json.loads(cn['producer_receipt']));verified+=1
    gate=json.loads((run/'admission-serving.json').read_text());gate_notes=notes(gate)
    gate_at=wall_ns(json.loads(gate_notes['completed_at']))
    closed=[o for o in central['observations'] if wall_ns(o['observed_at'].replace('+00:00','Z'))>=gate_at and not o['member_forwarding_enabled']]
    stream_result=dict(measured=bool(sends),sender_status=stream_record['status'] if stream_record else None,attempts=len(sends),send_errors=sum(not s['sent'] for s in sends),member_sequences=len(stream_packets['member']),node_sequences=len(stream_packets['node']),origin_sequences=len(stream_packets['origin']),missing_ingress=len(missing),missing_ingress_ranges=sequence_ranges(missing),absent_member_ranges=sequence_ranges(absent_member),gaps_at_least_one_second=gaps,delivered_after_one_second=delays,transit_upper_ns=dict(min=str(upper[0]),median=str(upper[len(upper)//2]),p99=str(upper[min(len(upper)-1,int(len(upper)*.99))]),max=str(upper[-1])) if upper else None,intervals_file='stream-intervals.csv',diagnostic_threshold='1 second selects detailed gap/delay views only; all missing packets and all intervals remain counted')
    result=dict(capture_checks=checks,clock_offsets=clocks,private_receipts_verified=verified,admission=json.loads(gate_notes['accepted']),first_selection=initial_result,stream=stream_result,handshakes=groups,member_initiations=member_inits,loopback_source_responses=[g for g in groups if g['wireguard']['type']==2 and g['wireguard']['source_loopback']],native_debug=debug,member_trace=trace_results,closed_reports_after_admission=closed,scope='Fresh reduced population, no historical row reclassification. Structural WireGuard headers and socket-write bounds are not cryptographic acceptance or an exact kernel drop callsite. All loss remains loss. Full native-session samples retain route tables, endpoint/peer state and independent wg handshake readbacks.')
    publish(destination,source,'measurement','Follow-up 27 first-contact, handshake and member-send/ingress joins',result)
    print(json.dumps(dict(run=run.name,first=initial_result['status'],syns=initial_result['member_syns'],transit=transit,stream_attempts=len(sends),missing_ingress=len(missing),gaps=len(gaps),loopback_responses=len(result['loopback_source_responses']),verified=verified)))


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('run',type=Path);parser.add_argument('private',type=Path)
    args=parser.parse_args();main(args.run,args.private)
