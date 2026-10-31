"""TEST ONLY F8-02: IPv4 bulk over direct kernel WireGuard/IPv6 underlay.

Same 64 complete 1001-byte payload lines and 12-second deadline as native
acceptance. At most three MTU-1280 attempts; stop on the first failure. This
is a kernel transport mechanism fixture, not a NetBird ICE/platform replay.
"""
import json
from pathlib import Path
import re
import signal
import socket
import subprocess
import sys
import threading
import time

from mtu_measure import run
from pcap_headers import decode, compare_udp

BASE = Path('/scratch/f21-ipv6')
SCRIPT = str(Path(__file__).resolve())
ORIGIN = '203.0.113.10'
PORT = 9446


def write(path, value):
    path.write_text(json.dumps(value,indent=2)+'\n')


def endpoint(server=False):
    def transfer(s, label):
        s.setsockopt(socket.IPPROTO_TCP,socket.TCP_NODELAY,1)
        started = time.monotonic()
        row = dict(role=label,read_bytes=0,write_bytes=0,complete_lines=0,partial_line_bytes=0,error='none')
        buffer = b''
        def send(data):
            s.settimeout(max(.001,12-(time.monotonic()-started)))
            s.sendall(data)
            row['write_bytes'] += len(data)
        def line():
            nonlocal buffer
            while b'\n' not in buffer:
                s.settimeout(max(.001,12-(time.monotonic()-started)))
                data = s.recv(2048)
                row['read_bytes'] += len(data)
                if not data:
                    raise EOFError()
                buffer += data
            part, buffer = buffer.split(b'\n',1)
            return part+b'\n'
        try:
            if label == 'origin':
                assert line().startswith(b'MEMBER ')
                send(b'OK\n')
                while True:
                    part = line()
                    assert len(part) == 1001
                    row['complete_lines'] += 1
                    send(b'ECHO '+part)
            else:
                s.connect((ORIGIN,PORT))
                row['local_port'] = s.getsockname()[1]
                send(b'MEMBER 019a0000-0000-7000-8000-0000000000a1\n')
                assert line() == b'OK\n'
                payload = b'x'*1000+b'\n'
                send(payload*64)
                for _ in range(64):
                    assert line() == b'ECHO '+payload
                    row['complete_lines'] += 1
        except (OSError,EOFError) as exc:
            row['error'] = type(exc).__name__
        finally:
            row['partial_line_bytes'] = len(buffer)
            row['elapsed_ms'] = round((time.monotonic()-started)*1000,3)
            row['validated_bytes'] = row['complete_lines']*1001 if label == 'member' else None
            if label == 'origin':
                with (BASE/'origin-endpoints.jsonl').open('a') as f:
                    f.write(json.dumps(row)+'\n')
            else:
                print(json.dumps(row))
            s.close()
    if not server:
        transfer(socket.socket(), 'member')
        return
    s = socket.socket()
    s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1)
    s.bind((ORIGIN,PORT))
    s.listen()
    (BASE/'server.ready').touch()
    while True:
        conn,_ = s.accept()
        threading.Thread(target=transfer,args=(conn,'origin'),daemon=True).start()


def capture_start(folder, roles):
    captures = []
    for role, ns in roles:
        cmd = (['ip','netns','exec',ns] if ns else [])+['tcpdump','--immediate-mode','-U','-nn','-B','32768','-s','65535','-i','any','-w',str(folder/(role+'.pcap')),'tcp port 9446 or udp or icmp or ip6']
        with (folder/(role+'.err')).open('w') as err:
            proc = subprocess.Popen(cmd,stdout=subprocess.DEVNULL,stderr=err)
        captures.append((role,proc))
        limit = time.monotonic()+5
        while 'listening on' not in (folder/(role+'.err')).read_text():
            if proc.poll() is not None or time.monotonic()>limit:
                raise RuntimeError('capture startup '+role)
            time.sleep(.01)
    return captures


def capture_stop(folder,captures):
    for _,proc in captures:
        proc.send_signal(signal.SIGINT)
    all_rows, checks = {}, {}
    for role,proc in captures:
        proc.wait(timeout=5)
        rows = decode(folder/(role+'.pcap'))
        with (folder/(role+'.headers.jsonl')).open('w') as f:
            for row in rows:
                f.write(json.dumps(row)+'\n')
        stats = (folder/(role+'.err')).read_text()
        n = int(re.search(r'(\d+) packets captured',stats)[1])
        drops = int(re.search(r'(\d+) packets dropped by kernel',stats)[1])
        checks[role] = dict(decoded=len(rows),captured=n,kernel_drops=drops,complete=n==len(rows) and drops==0)
        all_rows[role] = rows
    write(folder/'capture-checks.json',checks)
    if not all(c['complete'] for c in checks.values()):
        raise RuntimeError('capture incomplete')
    return all_rows


def links():
    return {role:json.loads(run(ns,'ip','-j','link','show')['stdout']) for role,ns in [('member','fv-a1'),('tunnel','fv-fwd'),('node',None),('origin','fv-inet')]}


def setup():
    for ns,dev,addr in [('fv-a1','fv-a1u','2001:db8:21::2/64'),('fv-fwd','fv-fa1','2001:db8:21::1/64')]:
        run(ns,'ip','-6','addr','add',addr,'dev',dev,'nodad')
        run(ns,'ip','link','set',dev,'mtu','1500')
        run(ns,'ip','link','set','fv-wg0','mtu','1280')
    # Only member a1's endpoint changes. The peer learns the reverse IPv6
    # endpoint from this authenticated traffic, just as for a new direct path.
    run('fv-a1','/usr/local/bin/wg','set','fv-wg0','peer',Path('/scratch/keys/fwd.pub').read_text().strip(),
        'endpoint','[2001:db8:21::1]:51820')


def measure():
    if not Path('/scratch/topology.ok').exists():
        raise RuntimeError('isolated lab topology required')
    BASE.mkdir()
    setup()
    server = subprocess.Popen(['ip','netns','exec','fv-inet','python3',SCRIPT,'serve'])
    campaigns=[]
    try:
        end=time.monotonic()+5
        while not (BASE/'server.ready').exists():
            if server.poll() is not None or time.monotonic()>end:
                raise RuntimeError('echo startup')
            time.sleep(.01)
        for attempt in range(4):
            mtu=1500 if attempt==0 else 1280
            run('fv-a1','ip','link','set','fv-a1u','mtu',str(mtu))
            folder=BASE/('control-1500' if attempt==0 else 'attempt-'+str(attempt))
            folder.mkdir()
            write(folder/'links.json',links())
            # No reconnect after lowering the member link, matching broader
            # fixture mutation. Before/after route and endpoint family readback.
            route=run('fv-a1','ip','-6','-j','route','get','2001:db8:21::1')
            write(folder/'route.json',json.loads(route['stdout']))
            captures=capture_start(folder,[('member','fv-a1'),('tunnel','fv-fwd'),('node',None),('origin','fv-inet')])
            try:
                result=run('fv-a1','python3',SCRIPT,'client')
                row=json.loads(result['stdout'])
                row.update(attempt=attempt,member_underlay_mtu=mtu,underlay_family='ipv6',overlay_family='ipv4')
                write(folder/'endpoint.json',row)
                time.sleep(.2)
            finally:
                headers=capture_stop(folder,captures)
            def udp(role,dst):
                return [r for r in headers[role] if r.get('family')==6 and r['dst']==dst and (r.get('sport')==51820 or r.get('dport')==51820 or 'fragment_id' in r)]
            analysis=dict(reverse=compare_udp(udp('tunnel','2001:db8:21::2'),udp('member','2001:db8:21::2')),
                          forward=compare_udp(udp('member','2001:db8:21::1'),udp('tunnel','2001:db8:21::1')),
                          ptb={role:[r for r in rows if r.get('protocol')==58 and r.get('icmp_type')==2] for role,rows in headers.items()})
            write(folder/'datagrams.json',analysis)
            campaigns.append(dict(directory=folder.name,endpoint=row,datagrams=analysis))
            write(BASE/'summary.json',dict(test_only=True,maximum_1280_attempts=3,rows=campaigns))
            if attempt==0 and row['validated_bytes']!=64064:
                raise RuntimeError('MTU-1500 positive control failed')
            if attempt and row['validated_bytes']!=64064:
                break
        print(json.dumps(dict(attempts=len(campaigns)-1,rows=[r['endpoint'] for r in campaigns])))
    finally:
        server.terminate()
        server.wait(timeout=5)
        run('fv-a1','ip','link','set','fv-a1u','mtu','1500')


if __name__=='__main__':
    {'measure':measure,'serve':lambda:endpoint(True),'client':endpoint}[sys.argv[1]]()
