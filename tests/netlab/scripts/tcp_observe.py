#!/usr/bin/env python3
"""TEST ONLY TCP observation. Raw pcaps never leave private lab scratch.

Kernel packet timestamps and executor wall_ns share CLOCK_REALTIME. Paired
monotonic anchors bound clock checks; no invented monotonic packet timestamps.
This module does not import or change the D-120 verdict evaluator.
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
import time

ROLES = (("origin", "fv-a1"), ("node", None), ("responder", "fv-inet"))


def stamp():
    before = time.monotonic_ns()
    wall = time.time_ns()
    after = time.monotonic_ns()
    return dict(wall_ns=str(wall), mono_before_ns=str(before), mono_after_ns=str(after))


def decode(raw, interfaces):
    """Strict nano-PCAP / SLL2 / unfragmented IPv4 TCP header parser."""
    if len(raw) < 24 or raw[:4] != b'\x4d\x3c\xb2\xa1':
        raise ValueError('requires nanosecond little-endian pcap')
    major, minor, _, _, snaplen, link = struct.unpack_from('<HHIIII', raw, 4)
    if (major, minor, link) != (2, 4, 276) or snaplen < 80:
        raise ValueError('requires pcap v2.4 Linux cooked v2')
    pos, rows = 24, []
    while pos < len(raw):
        if len(raw) - pos < 16:
            raise ValueError('partial pcap record')
        sec, nano, size, original = struct.unpack_from('<IIII', raw, pos)
        pos += 16
        frame = raw[pos:pos + size]
        pos += size
        if nano >= 10**9 or size != original or size > snaplen or len(frame) != size or size < 60:
            raise ValueError('truncated or invalid pcap packet')
        proto, reserved, index, _, direction, addrlen = struct.unpack_from('!HHIHBB', frame)
        if proto != 0x800 or reserved or direction not in (0, 4) or addrlen > 8 or str(index) not in interfaces:
            raise ValueError('unexpected cooked header or interface')
        ip = frame[20:]
        ihl = (ip[0] & 15) * 4
        length = struct.unpack_from('!H', ip, 2)[0]
        frag = struct.unpack_from('!H', ip, 6)[0]
        if ip[0] >> 4 != 4 or ihl < 20 or length < ihl + 20 or length != len(ip) or frag & 0x3fff or ip[9] != 6:
            raise ValueError('invalid, fragmented or non-TCP IPv4 packet')
        tcp = ip[ihl:]
        sport, dport, seq, ack = struct.unpack_from('!HHII', tcp)
        head = (tcp[12] >> 4) * 4
        if head < 20 or head > len(tcp) or tcp[12] & 14 or 9001 not in (sport, dport):
            raise ValueError('invalid TCP header or wrong probe port')
        flags = ((tcp[12] & 1) << 8) | tcp[13]
        row = dict(packet=len(rows) + 1, wall_ns=str(sec * 10**9 + nano),
                   interface=interfaces[str(index)], direction='out' if direction == 4 else 'in',
                   source=socket.inet_ntoa(ip[12:16]), destination=socket.inet_ntoa(ip[16:20]),
                   source_port=sport, destination_port=dport, seq=seq, ack=ack,
                   flags=flags, flags_text=[n for bit, n in enumerate(('FIN','SYN','RST','PSH','ACK','URG','ECE','CWR','NS')) if flags & (1 << bit)],
                   ip_id=struct.unpack_from('!H', ip, 4)[0], ip_length=length,
                   data_length=len(tcp) - head, window=struct.unpack_from('!H', tcp, 14)[0])
        opts, at = tcp[20:head], 0
        while at < len(opts):
            kind = opts[at]
            if kind == 0:
                break
            if kind == 1:
                at += 1
                continue
            if at + 2 > len(opts) or opts[at+1] < 2 or at + opts[at+1] > len(opts):
                raise ValueError('truncated TCP option')
            size = opts[at+1]
            if kind == 2:
                if size != 4: raise ValueError('bad MSS option')
                row['mss'] = struct.unpack_from('!H', opts, at+2)[0]
            elif kind == 8:
                if size != 10: raise ValueError('bad timestamp option')
                row['tsval'], row['tsecr'] = struct.unpack_from('!II', opts, at+2)
            elif kind == 5:
                if size < 10 or (size - 2) % 8: raise ValueError('bad SACK option')
                row['sack'] = [list(struct.unpack_from('!II', opts, i)) for i in range(at+2, at+size, 8)]
            at += size
        rows.append(row)
    return rows


def validate_stats(text, count):
    captured = re.search(r'^(\d+) packets captured$', text, re.M)
    dropped = re.search(r'^(\d+) packets dropped by kernel$', text, re.M)
    received = re.search(r'^(\d+) packets received by filter$', text, re.M)
    if not captured or not dropped or not received or int(captured[1]) != count or int(dropped[1]):
        raise ValueError('incomplete TCP capture or kernel drops')
    return dict(captured=count, received_by_filter=int(received[1]), dropped_by_kernel=int(dropped[1]))


def conntrack(text):
    """Parse filtered conntrack -L -p tcp -o extended,id, never hide a bad row."""
    rows = []
    for line in text.splitlines():
        match = re.fullmatch(r'ipv4\s+2\s+tcp\s+6\s+(\d+)\s+(\w+)\s+src=(\S+) dst=(\S+) sport=(\d+) dport=(\d+)\s+(?:\[UNREPLIED\]\s+)?src=(\S+) dst=(\S+) sport=(\d+) dport=(\d+)\s+(.*)', line.strip())
        if not match:
            raise ValueError('invalid conntrack record')
        v = match.groups()
        if v[2] != '100.64.1.1' or v[3] != '203.0.113.10' or v[5] != '9001' or v[6] != '203.0.113.10' or v[8] != '9001':
            raise ValueError('wrong conntrack flow')
        for addr in (v[2], v[3], v[6], v[7]):
            socket.inet_pton(socket.AF_INET, addr)
        ident = re.search(r'\bid=(\d+)\b', v[10])
        if not ident: raise ValueError('missing conntrack id')
        rows.append(dict(timeout_seconds=int(v[0]), state=v[1], source_port=int(v[4]),
                         reply_destination=v[7], reply_port=int(v[9]), id=ident[1],
                         assured='[ASSURED]' in line, unreplied='[UNREPLIED]' in line))
    if len(rows) > 1:
        raise ValueError('ambiguous probe conntrack flows')
    return rows


def start(base):
    meta = dict(test_only=True, clock='CLOCK_REALTIME', start=stamp(), roles={})
    for role, ns in ROLES:
        prefix = ['ip', 'netns', 'exec', ns] if ns else []
        interfaces = {str(r['ifindex']): r['ifname'] for r in json.loads(subprocess.check_output(prefix + ['ip', '-j', 'link']))}
        path = Path(base + '-' + role)
        cmd = prefix + ['tcpdump', '--immediate-mode', '--time-stamp-precision=nano', '-U', '-nn', '-s', '0', '-i', 'any', '-y', 'LINUX_SLL2', '-w', str(path) + '.pcap', 'tcp port 9001']
        with Path(str(path) + '.err').open('w') as err:
            proc = subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=err, start_new_session=True)
        Path(str(path) + '.pid').write_text(str(proc.pid))
        until = time.monotonic() + 5
        while 'listening on' not in Path(str(path) + '.err').read_text():
            if proc.poll() is not None or time.monotonic() > until:
                raise RuntimeError('TCP capture did not start: ' + role)
            time.sleep(.01)
        meta['roles'][role] = dict(interfaces=interfaces, ready=stamp())
    Path(base + '-meta.json').write_text(json.dumps(meta, indent=2) + '\n')


def stop(base):
    meta = json.loads(Path(base + '-meta.json').read_text())
    for role, _ in ROLES:
        path = base + '-' + role
        os.kill(int(Path(path + '.pid').read_text()), signal.SIGINT)
        until = time.monotonic() + 5
        while 'packets dropped by kernel' not in Path(path + '.err').read_text():
            if time.monotonic() > until: raise RuntimeError('TCP capture did not stop: ' + role)
            time.sleep(.01)
        rows = decode(Path(path + '.pcap').read_bytes(), meta['roles'][role]['interfaces'])
        meta['roles'][role]['stats'] = validate_stats(Path(path + '.err').read_text(), len(rows))
        Path(path + '.jsonl').write_text(''.join(json.dumps(r) + '\n' for r in rows))
        Path(path + '.pcap').unlink()
    meta['end'] = stamp()
    Path(base + '-meta.json').write_text(json.dumps(meta, indent=2) + '\n')


if __name__ == '__main__':
    {'start': start, 'stop': stop}[sys.argv[1]](sys.argv[2])
