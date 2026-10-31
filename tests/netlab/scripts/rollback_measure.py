"""TEST ONLY AT02 consult: real failed Apply, baseline, and explicit retry.

A one-shot verifier read fault is deliberate; it does not reconstruct members'
HEALTH_CHECK_FAILED cause. No logging is inserted inside the guard bracket.
"""
import json
from pathlib import Path
import select
import socket
import struct
import subprocess
import sys
import time

from ipv6_underlay import capture_start, capture_stop, write, links
from mtu_measure import run

BASE=Path('/scratch/f21-rollback')
SCRIPT=str(Path(__file__).resolve())


def sender():
    s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM)
    s.bind(('100.64.1.1',40021))
    s.setblocking(False)
    rows=[]
    due=time.monotonic()
    (BASE/'sender.ready').touch()
    seq=0
    until=time.monotonic()+15
    while time.monotonic()<until and not (BASE/'sender.stop').exists():
        seq+=1
        payload=struct.pack('!Q',seq)
        rows.append(dict(sequence=seq,send_mono_ns=str(time.monotonic_ns()),send_wall_ns=str(time.time_ns())))
        s.sendto(payload,('203.0.113.10',9000))
        due+=.005
        while time.monotonic()<due:
            ready,_,_=select.select([s],[],[],max(0,due-time.monotonic()))
            if ready:
                data,_=s.recvfrom(128)
                rows.append(dict(reply_sequence=struct.unpack('!Q',data)[0],read_mono_ns=str(time.monotonic_ns())))
    write(BASE/'probes.json',rows)
    s.close()


def measure(model):
    BASE.mkdir()
    write(BASE/'links.json',links())
    captures=capture_start(BASE,[('member','fv-a1'),('node',None),('origin','fv-inet')])
    sender_proc=None
    operations=[]
    def apply(fail=False):
        args=['fvpn-netlab','apply','-f',model,'-trace',str(BASE/'apply.jsonl')]
        if fail:
            args+=['-fail-verification-read']
        result=run(None,*args,check=False)
        operations.append(dict(fault_requested=fail,**result))
        write(BASE/'commands.json',operations)
        if (result['code']==0)==fail:
            raise RuntimeError('unexpected Apply result')
    try:
        apply()
        sender_proc=subprocess.Popen(['ip','netns','exec','fv-a1','python3',SCRIPT,'sender'])
        end=time.monotonic()+5
        while not (BASE/'sender.ready').exists():
            if sender_proc.poll() is not None or time.monotonic()>end:
                raise RuntimeError('sender startup')
            time.sleep(.01)
        time.sleep(.2)
        for attempt in range(3):
            apply(True)
            baseline=run(None,'nft','-j','list','table','inet','fvpn')['stdout']
            write(BASE/('baseline-'+str(attempt)+'.json'),json.loads(baseline))
            # Explicit observation hold, not an agent retry interval or SLA.
            time.sleep(.2)
            apply()
            time.sleep(.2)
    finally:
        (BASE/'sender.stop').touch()
        if sender_proc is not None:
            sender_proc.wait(timeout=5)
        headers=capture_stop(BASE,captures)
    trace=[json.loads(x) for x in (BASE/'apply.jsonl').read_text().splitlines()]
    node_if={x['ifname']:x['ifindex'] for x in json.loads((BASE/'links.json').read_text())['node']}
    ingress={r['probe_sequence']:r for r in headers['node'] if 'probe_sequence' in r and r['ifindex']==node_if['fv-fh'] and r['direction']=='in'}
    egress={r['probe_sequence']:r for r in headers['node'] if 'probe_sequence' in r and r['ifindex']==node_if['fv-ih'] and r['direction']=='out'}
    guards=[(int(t['batches'][0]['start']['wall_ns']),int(t['batches'][1]['end']['wall_ns'])) for t in trace if t['deadline']['guard_released']]
    losses=[dict(sequence=n,ingress_wall_ns=r['wall_ns'],overlaps_completed_guard=any(a<=int(r['wall_ns'])<=b for a,b in guards)) for n,r in ingress.items() if n not in egress]
    windows=[]
    for i,t in enumerate(trace):
        if not t.get('test_read_fault_injected'):
            continue
        baseline=next(b for b in t['batches'] if b['kind']=='baseline')
        recovery=trace[i+1]
        release=next(b for b in recovery['batches'] if b['kind']=='guard_release')
        a,b=int(baseline['start']['wall_ns']),int(release['end']['wall_ns'])
        missing=[x for x in losses if a<=int(x['ingress_wall_ns'])<=b]
        windows.append(dict(failed_apply_index=i,baseline_command=baseline,recovery_release=release,
                            baseline_to_recovery_outer_ms=(b-a)/1e6,
                            definitely_closed_ms=(int(release['start']['mono_ns'])-int(baseline['end']['mono_ns']))/1e6,
                            lost_ingresses=len(missing),first_drop_wall_ns=missing[0]['ingress_wall_ns'] if missing else None,
                            last_drop_wall_ns=missing[-1]['ingress_wall_ns'] if missing else None))
    summary=dict(test_only=True,deliberate_baseline_observation_hold_ms=200,ingresses=len(ingress),egresses=len(egress),
                 losses=losses,loss_count=len(losses),losses_outside_completed_guards=sum(not x['overlaps_completed_guard'] for x in losses),windows=windows)
    write(BASE/'summary.json',summary)
    assert len(windows)==3 and all(w['lost_ingresses']>0 for w in windows)
    assert summary['losses_outside_completed_guards']>0
    assert any(n in egress for n in ingress) and trace[-1]['verified']
    print(json.dumps({k:v for k,v in summary.items() if k!='losses'}))


if __name__=='__main__':
    if sys.argv[1]=='sender':
        sender()
    else:
        measure(sys.argv[2])
