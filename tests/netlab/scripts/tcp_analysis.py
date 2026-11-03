#!/usr/bin/env python3
"""TEST ONLY packet correlation, separate from the unchanged D-120 verdict.

Retain every packet observation and every retransmitted sequence interval.
An outer bracket match is correlation, not proof of the atomic commit time.
"""
import argparse
from collections import Counter, defaultdict
import gzip
import json
from pathlib import Path

from tcp_observe import conntrack, validate_stats


def read(path):
    data=path.read_bytes() if path.exists() else gzip.decompress(Path(str(path)+'.gz').read_bytes())
    return data.decode()


def rows(path):
    return [json.loads(x) for x in read(path).splitlines() if x.strip()]


def brackets(applies):
    out=[]
    for a in applies:
        batches={b['kind']:b for b in a['batches']}
        i,r=batches['guard_install'],batches['guard_release']
        if not i['ok'] or not r['ok']: raise ValueError('failed guard command')
        offset=int(a['start']['wall_ns'])-int(a['start']['mono_ns'])
        v=a['timing']['verifications']
        # Exact wall samples for command brackets; verifier monotonic brackets
        # are mapped with this Apply's paired anchor, not global wall time.
        out.append(dict(generation=a['generation'], verified=a['verified'],
                        install_start=int(i['start']['wall_ns']),install_end=int(i['end']['wall_ns']),
                        release_start=int(r['start']['wall_ns']),release_end=int(r['end']['wall_ns']),
                        verify_start=int(v[0]['start_ns'])+offset,verify_end=int(v[-1]['end_ns'])+offset,
                        wall_mono_drift_ns=abs(int(a['end']['wall_ns'])-int(a['end']['mono_ns'])-offset)))
    return out


def classify(wall, windows):
    """All overlaps; 2 us margin, never arbitrarily choose one generation."""
    hits=[]
    for w in windows:
        for phase,lo,hi in [('install','install_start','install_end'),('guard_certain','install_end','release_start'),('release','release_start','release_end'),('verify','verify_start','verify_end')]:
            if w[lo]-2000 <= wall <= w[hi]+2000:
                hits.append(dict(generation=w['generation'],phase=phase,delta_start_ns=wall-w[lo],delta_end_ns=wall-w[hi]))
    return hits


def key(row):
    # NAT changes addresses/ports, but not these header fields. IP ID + TSval
    # distinguish emitted retries from the copies of one packet at each hop.
    return tuple(row.get(k) for k in ('seq','ack','flags','data_length','ip_id','tsval','tsecr'))


def segment_kind(r):
    if r['flags'] & 2: return 'SYN-ACK' if r['flags'] & 16 else 'SYN'
    if r['data_length']: return 'data'
    if r['flags'] & 1: return 'FIN'
    if r['flags'] & 4: return 'RST'
    return 'ACK'


def correlate(captures, windows):
    observations=[]
    for role,data in captures.items():
        for r in data:
            r=dict(r,role=role, observation=f'{role}:{r["packet"]}')
            r['flow_direction']='request' if r['destination_port']==9001 else 'reply'
            r['guard_positions']=classify(int(r['wall_ns']),windows)
            observations.append(r)
    # Every transmission is emitted at exactly one endpoint. Capture records
    # with no endpoint emission remain explicit unmatched observations.
    emitted=sorted([r for r in observations if r['direction']=='out' and
                    ((r['role']=='origin' and r['flow_direction']=='request') or
                     (r['role']=='responder' and r['flow_direction']=='reply'))],key=lambda r:int(r['wall_ns']))
    lookup=defaultdict(list)
    for r in observations: lookup[(r['flow_direction'],key(r))].append(r)
    history=defaultdict(list); ack_history=defaultdict(set); isn={}; segments=[]; used=set()
    for r in emitted:
        direction=r['flow_direction']; identity=(direction,key(r))
        group=lookup[identity]
        endpoints=[x for x in group if x in emitted]
        if len(endpoints)!=1: raise ValueError('ambiguous emitted packet identity')
        path={}
        for x in group:
            stage=x['role']+'_'+x['direction']+'_'+x['interface']
            if stage in path: raise ValueError('duplicate packet stage')
            path[stage]=x
            used.add(x['observation'])
        if r['flags'] & 2 and not r['flags'] & 16:
            isn.setdefault(direction,r['seq'])
        if r['flags'] & 2 and r['flags'] & 16:
            isn.setdefault(direction,r['seq'])
        if direction not in isn: raise ValueError('missing handshake/initial sequence')
        relative=(r['seq']-isn[direction]) % (2**32)
        length=r['data_length']+bool(r['flags']&2)+bool(r['flags']&1)
        overlaps=[p['number'] for lo,hi,p in history[direction] if length and relative<hi and lo<relative+length]
        pure_ack=not length and r['flags']==16
        ack_tuple=(r['seq'],r['ack'])
        item=dict(number=len(segments)+1,direction=direction,kind=segment_kind(r),
                  seq=r['seq'],ack=r['ack'],relative_seq=relative,sequence_length=length,
                  data_length=r['data_length'],flags=r['flags_text'],wall_ns=r['wall_ns'],
                  retransmission=bool(overlaps),retransmits_segments=overlaps,
                  repeat_ack=pure_ack and ack_tuple in ack_history[direction],stages=path)
        if overlaps:
            item['since_original_ns']=int(r['wall_ns'])-int(segments[overlaps[0]-1]['wall_ns'])
            item['since_previous_overlapping_ns']=int(r['wall_ns'])-int(segments[overlaps[-1]-1]['wall_ns'])
        if length: history[direction].append((relative,relative+length,item))
        if pure_ack: ack_history[direction].add(ack_tuple)
        node=[p for p in group if p['role']=='node']
        item['node_guard_overlap']=any(h['phase'] in ('install','guard_certain','release') for p in node for h in p['guard_positions'])
        times=[int(p['wall_ns']) for p in group]
        item['crossed_boundaries']=[dict(generation=w['generation'],boundary=b) for w in windows for b in ('install_start','install_end','release_start','release_end','verify_start','verify_end') if min(times)<=w[b]<=max(times)]
        segments.append(item)
    return segments,[r for r in observations if r['observation'] not in used]


def snapshots(applies):
    result=[]
    for a in applies:
        entries=a.get('test_only_tcp_snapshots',[])
        if [r['phase'] for r in entries] != ['before_apply','after_apply']: raise ValueError('missing TCP snapshots')
        for r in entries:
            for name in ('conntrack','socket'):
                if not r[name]['ok']: raise ValueError('failed read-only snapshot: '+name)
                lo,hi=int(r[name]['start']['mono_ns']),int(r[name]['end']['mono_ns'])
                if lo>hi: raise ValueError('reversed snapshot')
                if r['phase']=='before_apply' and hi>int(a['start']['mono_ns']): raise ValueError('snapshot inside Apply')
                if r['phase']=='after_apply' and lo<int(a['end']['mono_ns']): raise ValueError('snapshot inside Apply')
            result.append(dict(generation=a['generation'],phase=r['phase'],conntrack=conntrack(r['conntrack']['output']),reads=r))
    return result


def failed_reads(events, segments, windows):
    """Retain failed/malformed reads with their sends; never infer success."""
    failures=[]
    for e in events:
        if e['event']!='tcp_read' or (not e.get('outcome') and e.get('size',0)>0): continue
        send=[x for x in events if x['event']=='tcp_send' and x['sequence']==e['sequence']]
        if len(send)!=1: raise ValueError('missing or duplicate TCP send')
        lo,hi=int(send[0]['wall_ns']),int(e['wall_ns'])
        failures.append(dict(sequence=e['sequence'],send=send[0],read=e,elapsed_ns=int(e['mono_ns'])-int(send[0]['mono_ns']),
                             segments_during_read=[s['number'] for s in segments if lo<=int(s['wall_ns'])<=hi],
                             guard_windows=[w['generation'] for w in windows if lo<=w['verify_end'] and w['install_start']<=hi]))
    return failures


def validate_read_coverage(events, segments):
    """A successful read needs captured handshake and received byte coverage.

    Capture counters alone cannot detect a wrong/empty capture filter. Merge
    TCP sequence ranges so retransmitted bytes cannot fill a capture hole.
    This checks observation completeness, not echo contents or D-120 verdicts.
    """
    reads=[e for e in events if e['event']=='tcp_read' and not e.get('outcome') and e.get('size',0)>0]
    if not reads: return
    paths={
        'request': {'origin_out_fv-wg0','node_in_fv-fh','node_out_fv-ih','responder_in_fv-ii'},
        'reply': {'responder_out_fv-ii','node_in_fv-ih','node_out_fv-fh','origin_in_fv-wg0'},
    }
    for direction,kind in (('request','SYN'),('reply','SYN-ACK')):
        if not any(s['direction']==direction and s['kind']==kind and paths[direction]<=s['stages'].keys() for s in segments):
            raise ValueError('successful read without fully captured handshake')
    needed=0
    for e in reads:
        needed+=e['size']
        ranges=[]
        for s in segments:
            arrival=s['stages'].get('origin_in_fv-wg0')
            if s['direction']=='reply' and s['data_length'] and arrival and int(arrival['wall_ns'])<=int(e['wall_ns'])+2000:
                lo=s['relative_seq']+int('SYN' in s['flags'])
                ranges.append((lo,lo+s['data_length']))
        cursor=1
        for lo,hi in sorted(ranges):
            if lo>cursor: break
            cursor=max(cursor,hi)
        if cursor<needed+1:
            raise ValueError('TCP read bytes missing from origin return capture')


def analyze(directory):
    applies=rows(directory/'at10-apply.jsonl'); events=rows(directory/'stream-a1.log.jsonl')
    windows=brackets(applies)
    meta=json.loads(read(directory/'at10-tcp-meta.json'))
    captures={role:rows(directory/('at10-tcp-'+role+'.jsonl')) for role in ('origin','node','responder')}
    for role,data in captures.items():
        validate_stats(read(directory/('at10-tcp-'+role+'.err')),len(data))
        if meta['roles'][role]['stats']['captured']!=len(data): raise ValueError('metadata capture mismatch')
    segments,unmatched=correlate(captures,windows)
    validate_read_coverage(events,segments)
    snaps=snapshots(applies)
    failures=failed_reads(events,segments,windows)
    guard_crossings=[s['number'] for s in segments if any(b['boundary'] in ('install_start','install_end','release_start','release_end') for b in s['crossed_boundaries'])]
    return dict(test_only=True,verdict_input=False,summary=dict(applies=len(applies),verified=sum(a['verified'] for a in applies),
        tcp_reads=sum(e['event']=='tcp_read' for e in events),tcp_failures=len(failures),segments=len(segments),
        retransmissions=sum(s['retransmission'] for s in segments),repeat_acks=sum(s['repeat_ack'] for s in segments),
        node_guard_overlap_segments=sum(s['node_guard_overlap'] for s in segments),guard_boundary_crossing_segments=guard_crossings,
        capture_packets={k:len(v) for k,v in captures.items()},unmatched_observations=len(unmatched),
        max_apply_wall_mono_drift_ns=max(w['wall_mono_drift_ns'] for w in windows)),
        clock=meta,windows=windows,failures=failures,segments=segments,unmatched=unmatched,snapshots=snaps)


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('directory',type=Path);p.add_argument('output',type=Path)
    args=p.parse_args();result=analyze(args.directory)
    args.output.write_text(json.dumps(result,indent=2)+'\n')
    print(json.dumps(result['summary'],indent=2))
