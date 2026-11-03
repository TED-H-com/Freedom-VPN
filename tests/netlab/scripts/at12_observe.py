#!/usr/bin/env python3
"""TEST ONLY all-interface packet metadata; no payloads or link addresses.

An AF_PACKET observer is independent of the suite's tcpdump lifecycle. Kernel
SO_TIMESTAMPNS timestamps and PACKET_STATISTICS retain timing and loss evidence.
Only derived synthetic lab headers are written. No production rule is changed.
"""
import json
import os
from pathlib import Path
import select
import signal
import socket
import struct
import subprocess
import sys
import time

ROLES = (("origin", "fv-a1"), ("node", None), ("responder", "fv-inet"))


def stamp():
    return dict(wall_ns=str(time.time_ns()), mono_ns=str(time.monotonic_ns()))


def headers(data, proto):
    row = dict(ethertype=proto, captured_length=len(data))
    if proto == 0x0800:
        if len(data) < 20 or data[0] >> 4 != 4:
            raise ValueError('invalid IPv4')
        head, size = (data[0] & 15) * 4, struct.unpack_from('!H', data, 2)[0]
        if head < 20 or size < head or size > len(data):
            raise ValueError('truncated IPv4')
        row.update(source=socket.inet_ntoa(data[12:16]), destination=socket.inet_ntoa(data[16:20]),
                   ip_length=size, ip_id=struct.unpack_from('!H', data, 4)[0], protocol=data[9],
                   fragment=struct.unpack_from('!H', data, 6)[0] & 0x3fff)
        if row['fragment']:
            return row
        body = data[head:size]
    elif proto == 0x86dd:
        if len(data) < 40 or data[0] >> 4 != 6:
            raise ValueError('invalid IPv6')
        size = 40 + struct.unpack_from('!H', data, 4)[0]
        if size > len(data):
            raise ValueError('truncated IPv6')
        row.update(source=socket.inet_ntop(socket.AF_INET6, data[8:24]),
                   destination=socket.inet_ntop(socket.AF_INET6, data[24:40]),
                   ip_length=size, protocol=data[6])
        body = data[40:size]
    else:
        # ARP and other link protocols are accounted for without link addresses.
        return row
    if row['protocol'] == 6:
        if len(body) < 20:
            raise ValueError('truncated TCP')
        head = (body[12] >> 4) * 4
        if head < 20 or head > len(body):
            raise ValueError('invalid TCP header length')
        sport, dport, seq, ack = struct.unpack_from('!HHII', body)
        flags = body[13]
        row.update(source_port=sport, destination_port=dport, seq=seq, ack=ack,
                   flags=flags, flags_text=[s for i, s in enumerate(('FIN','SYN','RST','PSH','ACK','URG','ECE','CWR')) if flags & (1 << i)],
                   data_length=len(body)-head, tcp_header_length=head)
    elif row['protocol'] == 17:
        if len(body) < 8:
            raise ValueError('truncated UDP')
        sport, dport, size = struct.unpack_from('!HHH', body)
        if size < 8 or size > len(body):
            raise ValueError('invalid UDP length')
        row.update(source_port=sport, destination_port=dport, data_length=size-8)
    elif row['protocol'] in (1, 58):
        if len(body) < 8:
            raise ValueError('truncated ICMP')
        row.update(icmp_type=body[0], icmp_code=body[1])
        if row['protocol'] == 1 and body[0] in (0, 8):
            row.update(icmp_id=struct.unpack_from('!H', body, 4)[0], icmp_sequence=struct.unpack_from('!H', body, 6)[0])
    return row


def record(base):
    # Linux SO_TIMESTAMPNS_OLD uses native longs on the lab's 64-bit targets.
    sock = socket.socket(socket.AF_PACKET, socket.SOCK_DGRAM, socket.htons(3))
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_RCVBUF, 4 * 1024 * 1024)
    sock.setsockopt(socket.SOL_SOCKET, 35, 1)
    count = 0
    errors = []
    start = stamp()
    Path(base + '.ready').write_text(json.dumps(start)+'\n')
    with Path(base + '.jsonl').open('x') as out:
        while not Path(base + '.stop').exists():
            if not select.select([sock], [], [], .05)[0]:
                continue
            data, ancillary, flags, address = sock.recvmsg(262144, 128)
            ts = [v for level, kind, v in ancillary if level == socket.SOL_SOCKET and kind == 35]
            try:
                if flags or len(ts) != 1:
                    raise ValueError('missing timestamp or truncated recvmsg')
                sec, ns = struct.unpack('@ll', ts[0])
                row = headers(data, address[1])
                row.update(packet=count+1, wall_ns=str(sec*10**9+ns), interface=address[0], packet_type=address[2])
                out.write(json.dumps(row)+'\n')
            except (ValueError, struct.error) as exc:
                errors.append(dict(packet=count+1, error=str(exc)))
            count += 1
    packets, drops = struct.unpack('II', sock.getsockopt(263, 6, 8))
    sock.close()
    meta = dict(test_only=True, start=start, end=stamp(), read_packets=count,
                socket_packets=packets, socket_drops=drops, errors=errors,
                complete=drops == 0 and not errors and packets == count)
    Path(base + '.meta.json').write_text(json.dumps(meta, indent=2)+'\n')
    Path(base + '.done').touch()


def start(base):
    for role, ns in ROLES:
        target = base + '-' + role
        assert not Path(target + '.ready').exists(), 'never reuse capture'
        prefix = ['ip', 'netns', 'exec', ns] if ns else []
        with Path(target + '.err').open('x') as err:
            proc = subprocess.Popen(prefix + [sys.executable, __file__, 'record', target],
                                    stdout=subprocess.DEVNULL, stderr=err, start_new_session=True)
        Path(target + '.pid').write_text(str(proc.pid))
        until = time.monotonic() + 5
        while not Path(target + '.ready').exists():
            if proc.poll() is not None or time.monotonic() > until:
                raise RuntimeError('observer not ready: '+role)
            time.sleep(.01)


def stop(base):
    for role, _ in ROLES:
        target = base + '-' + role
        Path(target + '.stop').touch()
        until = time.monotonic() + 5
        while not Path(target + '.done').exists():
            if time.monotonic() > until:
                os.kill(int(Path(target+'.pid').read_text()), signal.SIGTERM)
                raise RuntimeError('observer failed to finish: '+role)
            time.sleep(.01)
        meta = json.loads(Path(target + '.meta.json').read_text())
        if not meta['complete']:
            raise RuntimeError('incomplete observation: '+role)


if __name__ == '__main__':
    {'start': start, 'record': record, 'stop': stop}[sys.argv[1]](sys.argv[2])
