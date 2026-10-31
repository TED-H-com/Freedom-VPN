#!/usr/bin/env python3
"""TEST ONLY F8-02 measurement, invoked inside the isolated lab container.

No policy edits, MSS fixups, retries or native-NetBird claims. Header captures
and counters distinguish local PMTU errors, forwarded quoted errors and the
v1 IPv6 block. Synthetic payloads are never written to evidence.
"""
import json
import os
from pathlib import Path
import re
import signal
import socket
import struct
import subprocess
import sys
import threading
import time

BASE = Path('/scratch/f802')
V4_MEMBER, V4_ORIGIN = '100.64.1.1', '203.0.113.10'
V6_MEMBER, V6_ORIGIN = 'fd00:db8:a1::1', '2001:db8:203::10'


def run(ns, *cmd, check=True):
    prefix = ['ip', 'netns', 'exec', ns] if ns else []
    p = subprocess.run(prefix + list(cmd), text=True, capture_output=True, check=False)
    if check and p.returncode:
        raise RuntimeError(f'{ns} {cmd}: {p.stdout} {p.stderr}')
    return dict(code=p.returncode, stdout=p.stdout, stderr=p.stderr)


def write(name, value):
    (BASE / name).write_text(json.dumps(value, indent=2) + '\n')


def serve():
    def tcp_client(conn):
        with conn:
            conn.settimeout(15)
            try:
                while data := conn.recv(4096):
                    conn.sendall(data)
            except OSError:
                pass
    def tcp_loop(s):
        while True:
            conn, _ = s.accept()
            threading.Thread(target=tcp_client, args=(conn,), daemon=True).start()
    def udp_loop(s, version):
        while True:
            data, addr = s.recvfrom(4096)
            write('last-udp-v' + str(version) + '.json', dict(source=addr[0], port=addr[1], size=len(data)))
            s.sendto(data, addr)
    for version, family, addr in [(4, socket.AF_INET, V4_ORIGIN), (6, socket.AF_INET6, V6_ORIGIN)]:
        for kind, port in [(socket.SOCK_STREAM, 9011), (socket.SOCK_DGRAM, 9010)]:
            s = socket.socket(family, kind)
            s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            s.bind((addr, port))
            if kind == socket.SOCK_STREAM:
                s.listen(16)
                threading.Thread(target=tcp_loop, args=(s,), daemon=True).start()
            else:
                threading.Thread(target=udp_loop, args=(s, version), daemon=True).start()
    (BASE / 'server.ready').touch()
    threading.Event().wait()


def bulk(version):
    family, dst = (socket.AF_INET, V4_ORIGIN) if version == 4 else (socket.AF_INET6, V6_ORIGIN)
    start = time.monotonic()
    data = b'TEST ONLY\n' * 6406 + b'end!'
    assert len(data) == 64064
    received, error, connected, matches = 0, '', False, True
    with socket.socket(family, socket.SOCK_STREAM) as s:
        s.settimeout(12)
        try:
            s.connect((dst, 9011))
            connected = True
            mss = s.getsockopt(socket.IPPROTO_TCP, socket.TCP_MAXSEG)
            s.sendall(data)
            while received < len(data):
                s.settimeout(max(.001, 12 - (time.monotonic() - start)))
                chunk = s.recv(len(data) - received)
                if not chunk:
                    error = 'eof'
                    break
                matches = matches and chunk == data[received:received+len(chunk)]
                received += len(chunk)
            if not matches:
                error = 'mismatched_echo'
        except OSError as exc:
            error = type(exc).__name__
    print(json.dumps(dict(family=version, connected=connected, bytes=received, expected=64064,
                         error=error, matches=matches, elapsed_ms=round((time.monotonic()-start)*1000, 3),
                         socket_mss=mss if connected else None)))


def checksum(data):
    if len(data) % 2:
        data += b'\0'
    n = sum(struct.unpack('!' + 'H' * (len(data)//2), data))
    while n >> 16:
        n = (n & 65535) + (n >> 16)
    return (~n) & 65535


def error_packet(version, direction, nat_address='192.0.2.37', nat_port=40010):
    # Forwarded fragmentation-needed/PTB quotes a controlled UDP tuple. v4
    # uses the actual responder-observed SNAT tuple of the preceding exchange.
    # v6 has no established member flow: explicitly a block-policy negative.
    to_member = direction == 'to-member'
    if version == 4:
        src, dst = (nat_address, V4_ORIGIN) if to_member else (V4_ORIGIN, V4_MEMBER)
        sport, dport = (nat_port, 9010) if to_member else (9010, 40010)
        udp = struct.pack('!HHHH', sport, dport, 40, 0) + b'\0' * 8
        quote = struct.pack('!BBHHHBBH4s4s', 0x45, 0, 60, 123, 0x4000, 64, 17, 0,
                            socket.inet_aton(src), socket.inet_aton(dst))
        quote = quote[:10] + struct.pack('!H', checksum(quote)) + quote[12:]
        packet = struct.pack('!BBHHH', 3, 4, 0, 0, 1280) + quote + udp
        packet = packet[:2] + struct.pack('!H', checksum(packet)) + packet[4:]
        with socket.socket(socket.AF_INET, socket.SOCK_RAW, socket.IPPROTO_ICMP) as s:
            s.bind((V4_ORIGIN if to_member else V4_MEMBER, 0))
            s.sendto(packet, (nat_address if to_member else V4_ORIGIN, 0))
    else:
        src, dst = (V6_MEMBER, V6_ORIGIN) if to_member else (V6_ORIGIN, V6_MEMBER)
        sport, dport = (40010, 9010) if to_member else (9010, 40010)
        a, b = socket.inet_pton(socket.AF_INET6, src), socket.inet_pton(socket.AF_INET6, dst)
        udp = struct.pack('!HHHH', sport, dport, 16, 0) + b'\0' * 8
        pseudo = a+b+struct.pack('!I3xB', len(udp), 17)
        udp = udp[:6] + struct.pack('!H', checksum(pseudo+udp)) + udp[8:]
        quote = struct.pack('!IHBB', 6 << 28, len(udp), 17, 64)+a+b+udp
        packet = struct.pack('!BBHI', 2, 0, 0, 1280)+quote
        with socket.socket(socket.AF_INET6, socket.SOCK_RAW, socket.IPPROTO_ICMPV6) as s:
            s.bind((V6_ORIGIN if to_member else V6_MEMBER, 0))
            s.sendto(packet, (V6_MEMBER if to_member else V6_ORIGIN, 0))
    print(json.dumps(dict(family=version, direction=direction, advertised_mtu=1280, injected=True)))


def udp_seed():
    with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as s:
        s.bind((V4_MEMBER, 40010))
        s.settimeout(2)
        s.sendto(b'\0' * 32, (V4_ORIGIN, 9010))
        data, _ = s.recvfrom(128)
        assert data == b'\0' * 32
    print('controlled UDP exchange complete')


def snapshot():
    return dict(wall_ns=str(time.time_ns()),
                nft=run(None, 'nft', '-a', 'list', 'table', 'inet', 'fvpn')['stdout'],
                tc={dev: {kind: json.loads(run(None, 'tc', '-j', '-s', kind, 'show', 'dev', dev)['stdout'])
                          for kind in ('qdisc', 'class', 'filter')} for dev in ('fv-fh', 'fv-ih')})


def mtu(value):
    for ns, dev in [(None, 'fv-ih'), ('fv-inet', 'fv-ii')]:
        run(ns, 'ip', 'link', 'set', dev, 'mtu', str(value))
    write('links-' + str(value) + '.json', {
        'node': json.loads(run(None, 'ip', '-j', 'link', 'show', 'dev', 'fv-ih')['stdout']),
        'origin': json.loads(run('fv-inet', 'ip', '-j', 'link', 'show', 'dev', 'fv-ii')['stdout'])})


def reconnect():
    # Recreate only this fixture's member kernel WireGuard interface. Do not
    # claim NetBird down/up, ICE negotiation, relay or route reconciliation.
    run('fv-a1', 'ip', 'link', 'del', 'fv-wg0')
    run('fv-a1', 'ip', 'link', 'add', 'fv-wg0', 'type', 'wireguard')
    peer = Path('/scratch/keys/fwd.pub').read_text().strip()
    run('fv-a1', '/usr/local/bin/wg', 'set', 'fv-wg0', 'private-key', '/scratch/keys/a1',
        'peer', peer, 'endpoint', '192.0.2.1:51820', 'allowed-ips', '0.0.0.0/0,::/0')
    run('fv-a1', 'ip', 'addr', 'add', V4_MEMBER+'/32', 'dev', 'fv-wg0')
    run('fv-a1', 'ip', '-6', 'addr', 'add', V6_MEMBER+'/128', 'dev', 'fv-wg0', 'nodad')
    run('fv-a1', 'ip', 'link', 'set', 'fv-wg0', 'up', 'mtu', '1420')
    run('fv-a1', 'ip', 'route', 'add', 'default', 'dev', 'fv-wg0')
    run('fv-a1', 'ip', '-6', 'route', 'add', 'default', 'dev', 'fv-wg0')
    run('fv-a1', 'ethtool', '-K', 'fv-wg0', 'gro', 'off', 'gso', 'off', 'tso', 'off', check=False)
    return str(time.time_ns())


def measure():
    if not Path('/scratch/topology.ok').exists():
        raise RuntimeError('requires the isolated netlab topology')
    BASE.mkdir(exist_ok=True)
    captures, server, rows = [], None, []
    script = str(Path(__file__).resolve())
    def phase(name, ns, *cmd):
        before = snapshot()
        start = str(time.time_ns())
        result = run(ns, *cmd, check=False)
        row = dict(name=name, start_ns=start, end_ns=str(time.time_ns()), result=result,
                   before=before, after=snapshot())
        rows.append(row)
        write('measurements.json', rows)
        return row
    try:
        server = subprocess.Popen(['ip', 'netns', 'exec', 'fv-inet', 'python3', script, 'serve'])
        limit = time.monotonic()+5
        while not (BASE/'server.ready').exists():
            if server.poll() is not None or time.monotonic() > limit:
                raise RuntimeError('dual-stack echo fixture did not start')
            time.sleep(.01)
        for role, ns in [('member', 'fv-a1'), ('node', None), ('origin', 'fv-inet')]:
            cmd = (['ip', 'netns', 'exec', ns] if ns else []) + [
                'tcpdump', '--immediate-mode', '-nn', '-tt', '-v', '-l', '-s', '160', '-i', 'any',
                'icmp or icmp6 or tcp port 9011 or udp port 9010']
            with (BASE/(role+'.txt')).open('w') as out, (BASE/(role+'.err')).open('w') as err:
                proc = subprocess.Popen(cmd, stdout=out, stderr=err)
            captures.append((role, proc))
            limit = time.monotonic()+5
            while 'listening on' not in (BASE/(role+'.err')).read_text():
                if proc.poll() is not None or time.monotonic() > limit:
                    raise RuntimeError('capture did not start: '+role)
                time.sleep(.01)
        write('initial.json', snapshot())
        # Matrix ordering: IPv4 1500, IPv6 1500, both 1500, both 1280.
        # Families here are inner IP; the tunnel underlay stays IPv4.
        for label, value, families in [('v4-1500', 1500, [4]), ('v6-1500', 1500, [6]),
                                      ('dual-1500', 1500, [4, 6]), ('dual-1280', 1280, [4, 6])]:
            mtu(value)
            for family in families:
                connected_at = reconnect()
                # No sleep, warm-up probe or snapshot before the first transfer.
                start = str(time.time_ns())
                result = run('fv-a1', 'python3', script, 'bulk', str(family))
                rows.append(dict(name=f'reconnect-{label}-v{family}', reconnect_end_ns=connected_at,
                                 start_ns=start, end_ns=str(time.time_ns()), result=result, after=snapshot()))
                write('measurements.json', rows)
        # Retain the established tunnel for the steady-state comparison.
        for family in (4, 6):
            phase(f'steady-1280-v{family}', 'fv-a1', 'python3', script, 'bulk', str(family))
        for family, dst in [(4, V4_ORIGIN), (6, V6_ORIGIN)]:
            phase(f'pmtu-large-v{family}', 'fv-a1', 'ping', '-'+str(family), '-M', 'do', '-c', '1', '-W', '2', '-s', '1360', dst)
            phase(f'pmtu-small-v{family}', 'fv-a1', 'ping', '-'+str(family), '-M', 'do', '-c', '1', '-W', '2', '-s', '1200', dst)
        phase('udp-seed-v4', 'fv-a1', 'python3', script, 'udp-seed')
        nat = json.loads((BASE/'last-udp-v4.json').read_text())
        for family in (4, 6):
            for direction, ns in [('to-member', 'fv-inet'), ('to-origin', 'fv-a1')]:
                phase(f'quoted-error-v{family}-{direction}', ns, 'python3', script, 'error', str(family), direction, nat['source'], str(nat['port']))
        write('final.json', snapshot())
    finally:
        for _, proc in captures:
            proc.send_signal(signal.SIGINT)
        for role, proc in captures:
            proc.wait(timeout=5)
        if server is not None:
            server.terminate()
            server.wait(timeout=5)
        mtu(1500)
    checks = {}
    for role, _ in captures:
        text = (BASE/(role+'.txt')).read_text()
        stats = (BASE/(role+'.err')).read_text()
        headers = len(re.findall(r'^\d+\.\d+\s+\S+\s+(?:In|Out)\s+IP(?:6)? ', text, re.M))
        count = re.search(r'(\d+) packets captured', stats)
        drops = re.search(r'(\d+) packets dropped by kernel', stats)
        ok = bool(count and drops and int(count[1]) == headers and int(drops[1]) == 0)
        checks[role] = dict(parsed_headers=headers, captured=int(count[1]) if count else None,
                            dropped=int(drops[1]) if drops else None, ok=ok)
    write('capture-checks.json', checks)
    if not all(r['ok'] for r in checks.values()):
        raise RuntimeError('MTU capture count/drop check failed')
    print(json.dumps(dict(test_only=True, phases=len(rows), captures=checks,
                         transfers=[dict(name=r['name'], **json.loads(r['result']['stdout']))
                                    for r in rows if 'reconnect-' in r['name'] or 'steady-' in r['name']]), indent=2))


if __name__ == '__main__':
    mode = sys.argv[1]
    if mode == 'bulk':
        bulk(int(sys.argv[2]))
    elif mode == 'error':
        error_packet(int(sys.argv[2]), sys.argv[3], sys.argv[4], int(sys.argv[5]))
    else:
        {'serve': serve, 'measure': measure, 'udp-seed': udp_seed}[mode]()
