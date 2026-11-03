"""TEST ONLY: inside an owned netlab container; no host networking or tracing.

Original eight-element nft transaction/text read, followed by observational raw
netlink dumps. Raw order is retained because nft sorts elements before printing.
"""
import collections
import ipaddress
import json
import re
import socket
import struct
import subprocess
import sys
import time


def nft(script):
    return subprocess.check_output(['nft', '-f', '-'], input=script, text=True)


def duration(value):
    units = {'d': 86400000, 'h': 3600000, 'm': 60000, 's': 1000, 'ms': 1}
    return sum(int(n)*units[u] for n, u in re.findall(r'(\d+)(ms|[dhms])', value))


def attrs(data):
    result = []
    while data:
        size, kind = struct.unpack_from('=HH', data)
        assert 4 <= size <= len(data)
        result.append((kind & 0x3fff, data[4:size]))
        data = data[(size+3) & ~3:]
    return result


def attr(kind, payload):
    raw = struct.pack('=HH', len(payload)+4, kind)+payload
    return raw+b'\0'*((-len(raw)) % 4)


class Reader:
    def __init__(self):
        self.sock = socket.socket(socket.AF_NETLINK, socket.SOCK_RAW, 12)
        self.sock.bind((0, 0))
        self.sock.settimeout(2)
        self.seq = 0

    def dump(self):
        self.seq += 1
        payload = struct.pack('!BBH', 1, 0, 0)+attr(1, b'fvpn\0')+attr(2, b'fv_granularity\0')
        msg = struct.pack('=IHHII', 16+len(payload), (10 << 8) | 13, 0x301, self.seq, 0)+payload
        start = time.monotonic_ns()
        self.sock.send(msg)
        values, packets = [], []
        while True:
            data = self.sock.recv(65536)
            packets.append(data.hex())
            while data:
                size, kind, flags, seq, pid = struct.unpack_from('=IHHII', data)
                assert seq == self.seq and 16 <= size <= len(data)
                body, data = data[16:size], data[(size+3) & ~3:]
                if kind == 3:
                    return dict(start_ns=str(start), end_ns=str(time.monotonic_ns()),
                                wire_order=values, netlink_hex=packets)
                if kind == 2:
                    raise RuntimeError('netlink error: '+str(struct.unpack_from('=i', body)[0]))
                assert kind == (10 << 8) | 12
                outer = dict(attrs(body[4:]))
                for _, elem in attrs(outer[3]):
                    fields = dict(attrs(elem))
                    key = dict(attrs(fields[1]))[1]
                    timeout = struct.unpack('!Q', fields[4])[0]
                    expires = struct.unpack('!Q', fields[5])[0]
                    values.append([str(ipaddress.IPv4Address(key)), timeout, expires])


def entries(indices):
    return ', '.join('192.0.2.%d timeout %dms expires %dms' % (i, 1000+i, 700+i) for i in indices)


def create(padded=None):
    script = 'add set inet fvpn fv_granularity { type ipv4_addr; flags timeout; }\n'
    if padded:
        first, last = (range(1, 7), range(7, 9)) if padded == 'forward' else (range(7, 9), range(1, 7))
        script += 'add element inet fvpn fv_granularity { '+entries(first)+' }\n'
        script += 'add set inet fvpn fv_padding { type ipv4_addr; }\n'
        filler = ', '.join(str(ipaddress.IPv4Address(int(ipaddress.IPv4Address('100.64.0.0'))+i)) for i in range(5000))
        script += 'add element inet fvpn fv_padding { '+filler+' }\n'
        script += 'add element inet fvpn fv_granularity { '+entries(last)+' }\n'
    else:
        script += 'add element inet fvpn fv_granularity { '+entries(range(1, 9))+' }\n'
    start = time.monotonic_ns()
    nft(script)
    return dict(start_ns=str(start), end_ns=str(time.monotonic_ns()), padded=padded)


def ages(row):
    return [700+int(a.rsplit('.', 1)[1])-e for a, _, e in row['wire_order']]


def old_pass(pairs):
    return len(pairs) == 8 and [a for a, _ in pairs] == list(range(1001, 1009)) and all(pairs[i+1][1]-pairs[i][1] == 1 for i in range(7))


def trial(reader, index, padded=None, dumps=5):
    install = create(padded)
    start = time.monotonic_ns()
    text = subprocess.check_output(['nft', 'list', 'set', 'inet', 'fvpn', 'fv_granularity'], text=True)
    end = time.monotonic_ns()
    pairs = [(duration(a), duration(b)) for a, b in re.findall(r'timeout (\w+) expires (\w+)', text)]
    reads = []
    for _ in range(dumps):
        reads.append(reader.dump())
        time.sleep(.001)
    json_text = subprocess.check_output(['nft', '-j', 'list', 'set', 'inet', 'fvpn', 'fv_granularity'], text=True)
    nft('delete set inet fvpn fv_granularity\n'+('delete set inet fvpn fv_padding\n' if padded else ''))
    # With a common allocation base and exact integer conversion, age in dump
    # order can only increase. Any decrease proves distinct allocation bases.
    split = any(any(b < a for a, b in zip(ages(r), ages(r)[1:])) for r in reads)
    return dict(test_only=True, index=index, install=install,
                precision=dict(read_start_ns=str(start), read_end_ns=str(end),
                               text=text, timeout_expires_ms=pairs, json=json.loads(json_text)),
                old_pass=old_pass(pairs), allocation_split_proven=split, dumps=reads)


def main():
    mode = sys.argv[1]
    reader = Reader()
    nft('add table inet fvpn\n')
    try:
        print(json.dumps(dict(test_only=True, mode=mode,
              kernel=subprocess.check_output(['uname', '-r'], text=True).strip(),
              nft=subprocess.check_output(['nft', '--version'], text=True).strip(),
              nft_package=subprocess.check_output(['dpkg-query', '-W', '-f=${Version}', 'nftables'], text=True))), flush=True)
        if mode == 'normal':
            for i in range(1000):
                print(json.dumps(trial(reader, i)), flush=True)
        elif mode == 'padded':
            for i, direction in enumerate(['forward', 'reverse']):
                print(json.dumps(trial(reader, i, direction, 10)), flush=True)
        elif mode == 'dump':
            install = create()
            for i in range(2000):
                read = reader.dump()
                print(json.dumps(dict(test_only=True, index=i, install=install, dump=read)), flush=True)
            nft('delete set inet fvpn fv_granularity\n')
        else:
            raise ValueError(mode)
    finally:
        nft('delete table inet fvpn\n')
        reader.sock.close()


if __name__ == '__main__':
    main()
