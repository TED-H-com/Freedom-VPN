"""TEST ONLY real rollback with independent admitted/revoked packet captures."""
import json
from pathlib import Path
import select
import socket
import struct
import subprocess
import sys
import time

from ipv6_underlay import capture_start, capture_stop, links, write
from mtu_measure import run

BASE = Path('/scratch/f22-rollback')
SCRIPT = str(Path(__file__).resolve())


def sender(source):
    admitted = source == 'admitted'
    addr = '100.64.1.1' if admitted else '100.64.1.2'
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    s.bind((addr, 40022))
    s.setblocking(False)
    rows = []
    seq = 0 if admitted else 1 << 32
    end = time.monotonic() + 20
    (BASE / (source+'.ready')).touch()
    try:
        while time.monotonic() < end and not (BASE/'stop').exists():
            seq += 1
            rows.append(dict(sequence=seq, send_mono_ns=str(time.monotonic_ns()), send_wall_ns=str(time.time_ns())))
            s.sendto(struct.pack('!Q',seq), ('203.0.113.10',9000))
            due = time.monotonic()+.005
            while time.monotonic()<due:
                ready,_,_ = select.select([s],[],[],max(0,due-time.monotonic()))
                if ready:
                    data,_ = s.recvfrom(128)
                    rows.append(dict(reply_sequence=struct.unpack('!Q',data)[0], read_mono_ns=str(time.monotonic_ns())))
    finally:
        write(BASE/(source+'-probes.json'), rows)
        s.close()


def measure(model):
    BASE.mkdir()
    write(BASE/'links.json',links())
    captures = capture_start(BASE,[('admitted','fv-a1'),('revoked','fv-a2'),('node',None),('origin','fv-inet')])
    senders=[]
    operations=[]
    bounds=dict(test_only=True, load=dict(active=False, reason='quiet-host fixture; no injected CPU load'), start_ns=str(time.monotonic_ns()))
    def apply(fault=False):
        args=['fvpn-netlab','apply','-f',model,'-trace',str(BASE/'apply.jsonl')]
        if fault: args.append('-fail-verification-read')
        start=time.monotonic_ns()
        result=run(None,*args,check=False)
        operations.append(dict(fault_requested=fault,start_ns=str(start),end_ns=str(time.monotonic_ns()),**result))
        write(BASE/'operations.json',operations)
        if (result['code']==0)==fault: raise RuntimeError('unexpected Apply result')
    try:
        apply()
        for source,ns in [('admitted','fv-a1'),('revoked','fv-a2')]:
            senders.append(subprocess.Popen(['ip','netns','exec',ns,'python3',SCRIPT,'sender',source]))
        end=time.monotonic()+5
        while not all((BASE/(s+'.ready')).exists() for s in ['admitted','revoked']):
            if time.monotonic()>end: raise RuntimeError('probe startup')
            time.sleep(.01)
        time.sleep(.2)
        for attempt in range(3):
            apply(True)
            write(BASE/('baseline-'+str(attempt)+'.json'),json.loads(run(None,'nft','-j','list','table','inet','fvpn')['stdout']))
            time.sleep(.2)
            apply()
            time.sleep(.2)
    finally:
        (BASE/'stop').touch()
        for p in senders: p.wait(timeout=5)
        bounds['stop_ns']=str(time.monotonic_ns())
        write(BASE/'injection-bounds.json',bounds)
        headers=capture_stop(BASE,captures)
    trace=[json.loads(line) for line in (BASE/'apply.jsonl').read_text().splitlines()]
    indices={x['ifname']:x['ifindex'] for x in json.loads((BASE/'links.json').read_text())['node']}
    ing={r['probe_sequence']:r for r in headers['node'] if 'probe_sequence' in r and r['ifindex']==indices['fv-fh'] and r['direction']=='in'}
    out={r['probe_sequence']:r for r in headers['node'] if 'probe_sequence' in r and r['ifindex']==indices['fv-ih'] and r['direction']=='out'}
    guards=[(int(t['batches'][0]['start']['wall_ns']),int(t['batches'][1]['end']['wall_ns'])) for t in trace if t['deadline']['guard_released']]
    loss=[dict(sequence=n,ingress_wall_ns=r['wall_ns'],overlaps_completed_guard=any(a<=int(r['wall_ns'])<=b for a,b in guards)) for n,r in ing.items() if n not in out]
    windows=[]
    for i,t in enumerate(trace):
        if not t.get('test_read_fault_injected'): continue
        baseline=next(b for b in t['batches'] if b['kind']=='baseline')
        release=next(b for b in trace[i+1]['batches'] if b['kind']=='guard_release')
        a,b=int(baseline['start']['wall_ns']),int(release['end']['wall_ns'])
        windows.append(dict(failed_apply_index=i,baseline_command=baseline,recovery_release=release,
                            outer_ns=str(int(release['end']['mono_ns'])-int(baseline['start']['mono_ns'])),
                            closed_ns=str(int(release['start']['mono_ns'])-int(baseline['end']['mono_ns'])),
                            lost_admitted=sum(x['sequence']<1<<32 and a<=int(x['ingress_wall_ns'])<=b for x in loss)))
    counts={source:dict(ingresses=sum((n<1<<32)==admitted for n in ing),egresses=sum((n<1<<32)==admitted for n in out),losses=sum((x['sequence']<1<<32)==admitted for x in loss)) for source,admitted in [('admitted',True),('revoked',False)]}
    summary=dict(test_only=True,hold_ms=200,counts=counts,losses=loss,windows=windows)
    write(BASE/'summary.json',summary)
    assert counts['admitted']['egresses']>0 and counts['revoked']['ingresses']>0 and counts['revoked']['egresses']==0
    assert len(windows)==3 and all(w['lost_admitted']>0 for w in windows)
    assert trace[-1]['verified']
    print(json.dumps(dict(counts=counts,windows=windows)))

if __name__=='__main__':
    if sys.argv[1]=='sender': sender(sys.argv[2])
    else: measure(sys.argv[2])
