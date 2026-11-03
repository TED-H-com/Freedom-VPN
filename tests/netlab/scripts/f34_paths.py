#!/usr/bin/env python3
"""TEST ONLY D-146/D-147, entirely inside the privileged lane container.

Private pcaps are decoded to header-only JSON and deleted; no payload export.
"""
import json
from pathlib import Path
import signal
import socket
import struct
import subprocess
import sys
import threading
import time

from mtu_measure import checksum, run
from mtu_analysis import classes
from pcap_headers import decode, compare_udp

BASE = Path('/scratch/f34')
MEMBER = '100.64.1.1'
PORT = 9034
DATA = b'TEST ONLY\n' * 6406 + b'end!'


def write(name, value):
    (BASE / name).write_text(json.dumps(value, indent=2) + '\n')


def server(dst):
    def handle(c):
        with c:
            c.settimeout(15)
            try:
                op = c.recv(1)
                if op == b'D':
                    c.sendall(DATA)
                    c.shutdown(socket.SHUT_WR)
                else:
                    b = bytearray()
                    while len(b) < len(DATA):
                        part = c.recv(len(DATA)-len(b))
                        if not part:
                            break
                        b.extend(part)
                    c.sendall(struct.pack('!I', len(b)) + bytes([b == DATA]))
            except OSError:
                pass
    def tcp(s):
        while True:
            c, _ = s.accept()
            threading.Thread(target=handle, args=(c,), daemon=True).start()
    def udp(s):
        while True:
            b, a = s.recvfrom(4096)
            s.sendto(b'X' * 1280 if b == b'D' else b, a)
    for kind in (socket.SOCK_STREAM, socket.SOCK_DGRAM):
        s = socket.socket(socket.AF_INET, kind)
        s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        s.bind((dst, PORT))
        if kind == socket.SOCK_STREAM:
            s.listen(8)
        threading.Thread(target=tcp if kind == socket.SOCK_STREAM else udp, args=(s,), daemon=True).start()
    (BASE / 'ready').touch()
    threading.Event().wait()


def client(dst, direction, small=False):
    got, error, connected, pmtu, mss = 0, '', False, None, None
    start = time.monotonic()
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.settimeout(5)
        if small:
            s.setsockopt(socket.IPPROTO_TCP, socket.TCP_MAXSEG, 1000)
        try:
            s.connect((dst, PORT))
            connected = True
            if direction == 'upload':
                s.sendall(b'U' + DATA)
                result = b''
                while len(result) < 5:
                    b = s.recv(5-len(result))
                    if not b:
                        break
                    result += b
                if len(result) == 5 and result[4] == 1:
                    got = struct.unpack('!I', result[:4])[0]
            else:
                s.sendall(b'D')
                result = bytearray()
                while len(result) < len(DATA):
                    s.settimeout(max(.001, 5-(time.monotonic()-start)))
                    b = s.recv(len(DATA)-len(result))
                    if not b:
                        break
                    result.extend(b)
                got = len(result)
                if result != DATA[:got]:
                    error = 'data mismatch'
        except OSError as exc:
            error = type(exc).__name__
        if connected:
            pmtu = s.getsockopt(socket.IPPROTO_IP, 14)  # IP_MTU
            mss = s.getsockopt(socket.IPPROTO_TCP, socket.TCP_MAXSEG)
        # Abort on close so a failed black-hole flow cannot retransmit into
        # the next independently captured phase. The RST is itself counted.
        s.setsockopt(socket.SOL_SOCKET, socket.SO_LINGER, struct.pack('ii', 1, 0))
    print(json.dumps(dict(direction=direction, bytes=got, expected=len(DATA), connected=connected,
                          error=error, pmtu=pmtu, socket_mss=mss, elapsed_ms=(time.monotonic()-start)*1000)))


def udp_client(dst):
    with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as s:
        s.settimeout(1)
        s.sendto(b'D', (dst, PORT))
        try:
            data = s.recv(4096)
            print(json.dumps(dict(bytes=len(data), expected=1280, error='')))
        except OSError as exc:
            print(json.dumps(dict(bytes=0, expected=1280, error=type(exc).__name__)))


def unsolicited():
    # Valid ICMP checksum, but no quoted conntrack tuple; directed at a member.
    quote = struct.pack('!BBHHHBBH4s4s', 0x45, 0, 28, 1, 0x4000, 64, 17, 0,
                        socket.inet_aton(MEMBER), socket.inet_aton('203.0.113.34'))
    quote = quote[:10] + struct.pack('!H', checksum(quote)) + quote[12:]
    body = struct.pack('!BBHHH', 3, 4, 0, 0, 1000) + quote + struct.pack('!HHHH', 49999, 49998, 8, 0)
    body = body[:2] + struct.pack('!H', checksum(body)) + body[4:]
    with socket.socket(socket.AF_INET, socket.SOCK_RAW, socket.IPPROTO_ICMP) as s:
        s.bind(('192.168.34.1', 0))
        s.sendto(body, (MEMBER, 0))
    print(json.dumps(dict(injected=True, type=3, code=4)))


def router_setup():
    run(None, 'ip', 'netns', 'add', 'fv-f34origin')
    run('fv-f34origin', 'ip', 'link', 'set', 'lo', 'up')
    run(None, 'ip', 'link', 'add', 'fv-f34r', 'type', 'veth', 'peer', 'name', 'fv-f34o')
    run(None, 'ip', 'link', 'set', 'fv-f34r', 'netns', 'fv-inet')
    run(None, 'ip', 'link', 'set', 'fv-f34o', 'netns', 'fv-f34origin')
    for ns, dev, addr in [('fv-inet', 'fv-f34r', '192.0.2.41/30'), ('fv-f34origin', 'fv-f34o', '192.0.2.42/30')]:
        run(ns, 'ip', 'link', 'set', dev, 'up', 'mtu', '1000' if ns == 'fv-inet' else '1500')
        run(ns, 'ip', 'addr', 'add', addr, 'dev', dev)
        run(ns, 'ethtool', '-K', dev, 'gro', 'off', 'gso', 'off', 'tso', 'off')
    run('fv-f34origin', 'ip', 'addr', 'add', '203.0.113.34/32', 'dev', 'lo')
    run('fv-f34origin', 'ip', 'route', 'add', 'default', 'via', '192.0.2.41')
    run('fv-inet', 'ip', 'route', 'add', '203.0.113.34/32', 'via', '192.0.2.42')
    run('fv-inet', 'ip', 'addr', 'add', '192.168.34.1/32', 'dev', 'fv-ii')
    run('fv-inet', 'ip', 'route', 'add', '192.0.2.37/32', 'via', '192.0.2.37', 'dev', 'fv-ii', 'src', '192.168.34.1')
    run('fv-inet', 'sysctl', '-w', 'net.ipv4.ip_forward=1')
    run(None, 'ip', 'route', 'add', '192.168.34.1/32', 'via', '192.0.2.38', 'dev', 'fv-ih')
    write('router-setup.json', {ns: run(ns, 'ip', '-j', 'route')['stdout'] for ns in ['fv-inet', 'fv-f34origin']})


def router_cleanup():
    run(None, 'ip', 'route', 'del', '192.168.34.1/32')
    run('fv-inet', 'sysctl', '-w', 'net.ipv4.ip_forward=0')
    run('fv-inet', 'ip', 'route', 'del', '192.0.2.37/32')
    run('fv-inet', 'ip', 'route', 'del', '203.0.113.34/32')
    run('fv-inet', 'ip', 'addr', 'del', '192.168.34.1/32', 'dev', 'fv-ii')
    run(None, 'ip', 'netns', 'del', 'fv-f34origin')


def snapshot():
    raw = json.loads(run(None, 'nft', '-j', 'list', 'table', 'inet', 'fvpn')['stdout'])
    counters = {o['counter']['name']: {k: o['counter'][k] for k in ('packets', 'bytes')}
                for o in raw['nftables'] if 'counter' in o}
    return dict(nft=raw, counters=counters, tc={dev: classes(run(None, 'tc', '-s', 'class', 'show', 'dev', dev)['stdout'])
                                              for dev in ('fv-ih', 'fv-fh')})


def capture_start(name, ns):
    args = (['ip', 'netns', 'exec', ns] if ns else [])
    with (BASE/(name+'.err')).open('w') as err:
        p = subprocess.Popen(args + ['tcpdump', '--immediate-mode', '-U', '-n', '-B', '65536', '-s', '0', '-i', 'any', '-w', str(BASE/(name+'.pcap')),
                                     'icmp or tcp port 9034 or udp'], stdout=subprocess.DEVNULL, stderr=err)
    until = time.monotonic()+3
    while 'listening on' not in (BASE/(name+'.err')).read_text():
        if time.monotonic() > until or p.poll() is not None:
            raise RuntimeError('capture readiness '+name)
        time.sleep(.01)
    return p


def phase(name, roles, ns, *args):
    before = snapshot()
    processes = [(role, capture_start(name+'-'+role, where)) for role, where in roles]
    try:
        result = run(ns, *args, check=False)
        time.sleep(.2)
        after = snapshot()
    finally:
        for _, p in processes:
            p.send_signal(signal.SIGINT)
        for _, p in processes:
            p.wait(timeout=3)
    rows = {}
    import re
    for role, _ in processes:
        stem = name+'-'+role
        rows[role] = decode(BASE/(stem+'.pcap'))
        stats = (BASE/(stem+'.err')).read_text()
        count = re.search(r'(\d+) packets? captured', stats)
        drops = re.search(r'(\d+) packets dropped by kernel', stats)
        if not count or int(count[1]) != len(rows[role]) or not drops or int(drops[1]):
            raise RuntimeError('incomplete capture '+stem)
        write(stem+'-headers.json', rows[role])
        (BASE/(stem+'.pcap')).unlink()
    delta = {k: {n: v-before['counters'].get(k, {}).get(n, 0) for n, v in val.items()}
             for k, val in after['counters'].items() if val != before['counters'].get(k)}
    node_links = json.loads(run(None, 'ip', '-j', 'link')['stdout'])
    indices = {l['ifname']: l['ifindex'] for l in node_links}
    counted = {direction: dict(bytes=sum(r['ip_length'] for r in rows['node'] if r.get('family') == 4 and
                                          r['direction'] == 'out' and r['ifindex'] == indices[dev] and
                                          (r.get('src') == '192.0.2.37' if direction == 'upload' else r.get('dst') == MEMBER)),
                               packets=sum(1 for r in rows['node'] if r.get('family') == 4 and r['direction'] == 'out' and
                                           r['ifindex'] == indices[dev] and
                                           (r.get('src') == '192.0.2.37' if direction == 'upload' else r.get('dst') == MEMBER)))
               for direction, dev in [('upload', 'fv-ih'), ('download', 'fv-fh')]}
    tc_delta = {dev: {k: {n: v-before['tc'][dev].get(k, {}).get(n, 0) for n, v in val.items()}
                      for k, val in after['tc'][dev].items() if val != before['tc'][dev].get(k)} for dev in after['tc']}
    mss = {role: [r for r in rs if 'mss' in r] for role, rs in rows.items()}
    result_doc = dict(name=name, result=result, before=before, after=after, counter_delta=delta,
                      forwarded=counted, tc_delta=tc_delta, mss=mss,
                      icmp={role: [r for r in rs if r.get('protocol') == 1] for role, rs in rows.items()})
    if 'fwd' in rows:
        sent = [r for r in rows['fwd'] if r.get('dst') == '192.0.2.2' and r.get('protocol') == 17]
        received = [r for r in rows['member'] if r.get('dst') == '192.0.2.2' and r.get('protocol') == 17]
        result_doc['outer_receive'] = compare_udp(sent, received)
    write(name+'.json', result_doc)
    return result_doc


def measure(mode):
    BASE.mkdir(exist_ok=True)
    script = str(Path(__file__).resolve())
    router = mode == 'router'
    dst, ns = ('203.0.113.34', 'fv-f34origin') if router else ('203.0.113.10', 'fv-inet')
    if router:
        router_setup()
    else:
        # ONLY the member receiving veth changes. Peer, inner tunnel and exit
        # MTUs remain unchanged. veth drops oversize on receive without PTB.
        run('fv-a1', 'ip', 'link', 'set', 'fv-a1u', 'mtu', '1280')
    write(mode+'-links.json', {str(where): json.loads(run(where, 'ip', '-j', 'link')['stdout'])
                             for where in [None, 'fv-a1', 'fv-fwd', 'fv-inet']})
    run('fv-a1', 'sysctl', '-w', 'net.ipv4.tcp_mtu_probing=0')
    run('fv-fwd', 'ethtool', '-K', 'fv-wg0', 'gro', 'off', 'gso', 'off', 'tso', 'off')
    (BASE/'ready').unlink(missing_ok=True)
    server_proc = subprocess.Popen(['ip', 'netns', 'exec', ns, 'python3', '-B', script, 'server', dst])
    rows = []
    try:
        until = time.monotonic()+3
        while not (BASE/'ready').exists():
            if server_proc.poll() is not None or time.monotonic() > until:
                raise RuntimeError('server readiness')
            time.sleep(.01)
        roles = [('member', 'fv-a1'), ('node', None), ('origin', ns), ('fwd', 'fv-fwd')]
        if router:
            roles.append(('router', 'fv-inet'))
        for direction in (['upload'] if router else ['upload', 'download']):
            rows.append(phase(mode+'-'+direction, roles, 'fv-a1', 'python3', '-B', script, 'client', dst, direction))
        if router:
            rows.append(phase('router-time-exceeded', roles, 'fv-a1', 'ping', '-n', '-t', '3', '-c', '1', '-W', '1', dst))
            rows.append(phase('router-unsolicited', roles, 'fv-inet', 'python3', '-B', script, 'unsolicited'))
            rows.append(phase('router-denied-upload', roles, 'fv-a1', 'ping', '-n', '-c', '1', '-W', '1', '192.168.34.1'))
        else:
            rows.append(phase('receive-small-mss', roles, 'fv-a1', 'python3', '-B', script, 'small', dst, 'download'))
            rows.append(phase('receive-udp', roles, 'fv-a1', 'python3', '-B', script, 'udp', dst))
        write(mode+'-summary.json', [{k: r[k] for k in ['name', 'result', 'counter_delta', 'forwarded', 'tc_delta', 'outer_receive']} for r in rows])
        print(json.dumps([dict(name=r['name'], result=r['result'], counters=r['counter_delta']) for r in rows], indent=2))
    finally:
        server_proc.terminate()
        server_proc.wait(timeout=3)
        if router:
            router_cleanup()
        else:
            run('fv-a1', 'ip', 'link', 'set', 'fv-a1u', 'mtu', '9000')


if __name__ == '__main__':
    mode = sys.argv[1]
    if mode == 'server':
        server(sys.argv[2])
    elif mode in ('client', 'small'):
        client(sys.argv[2], sys.argv[3], mode == 'small')
    elif mode == 'udp':
        udp_client(sys.argv[2])
    elif mode == 'unsolicited':
        unsolicited()
    else:
        measure(mode)
