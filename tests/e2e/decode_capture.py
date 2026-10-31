#!/usr/bin/env python3
"""TEST ONLY pcap header export. No payload, MAC, hostname or peer key output.

Classic pcap, Ethernet or Linux cooked v1/v2. Address tokens are deterministic
within a supplied campaign using the same map. Header truncation is explicit;
wire lengths may exceed snaplen (and MTU under GSO), never infer wire packets.
"""
import argparse
import collections
import hashlib
import ipaddress
import json
from pathlib import Path
import struct


def u16(b, i):
    return struct.unpack_from('!H', b, i)[0]


def u32(b, i):
    return struct.unpack_from('!I', b, i)[0]


def decode_ip(b, address):
    version = b[0] >> 4
    out = {'ip_version': version}
    if version == 4:
        h = (b[0] & 15)*4
        if h < 20 or len(b) < h:
            raise ValueError('short IPv4 header')
        size, frag, proto = u16(b, 2), u16(b, 6), b[9]
        out.update(src=address(b[12:16]), dst=address(b[16:20]), ip_length=size,
                   ip_header_length=h, ttl=b[8], identification=u16(b, 4),
                   df=bool(frag & 0x4000), mf=bool(frag & 0x2000), fragment_offset=(frag & 8191)*8, protocol=proto)
        if frag & 8191:
            return out
    elif version == 6:
        if len(b) < 40:
            raise ValueError('short IPv6 header')
        size, proto, h = 40+u16(b, 4), b[6], 40
        out.update(src=address(b[8:24]), dst=address(b[24:40]), ip_length=size, hop_limit=b[7], extensions=[])
        while proto in (0, 43, 44, 51, 60):
            kind = proto
            proto = b[h]
            length = 8 if kind == 44 else (b[h+1]+2)*4 if kind == 51 else (b[h+1]+1)*8
            if len(b) < h+length:
                raise ValueError('short IPv6 extension')
            ext = dict(type=kind, length=length)
            if kind == 44:
                frag = u16(b, h+2)
                ext.update(fragment_offset=frag & 0xfff8, mf=bool(frag & 1), identification=u32(b, h+4), next_header=proto)
            out['extensions'].append(ext)
            h += length
            if kind == 44 and frag & 0xfff8:
                return out
        out['ip_header_length'] = h
    else:
        raise ValueError('non-IP')
    out['protocol'] = proto
    transport = b[h:]
    if proto == 6:
        if len(transport) < 20:
            raise ValueError('short TCP header')
        th = (transport[12] >> 4)*4
        if th < 20 or len(transport) < th:
            raise ValueError('short TCP options')
        tcp = dict(src_port=u16(transport, 0), dst_port=u16(transport, 2),
                   seq=u32(transport, 4), ack=u32(transport, 8), flags=transport[13],
                   window=u16(transport, 14), header_length=th, length=size-h-th, options=[])
        i = 20
        while i < th:
            kind = transport[i]
            if kind == 0:
                break
            if kind == 1:
                i += 1
                continue
            length = transport[i+1]
            if length < 2 or i+length > th:
                raise ValueError('malformed TCP option')
            option = dict(kind=kind, length=length)
            if kind == 2 and length == 4:
                option['mss'] = u16(transport, i+2)
            elif kind == 3 and length == 3:
                option['window_scale'] = transport[i+2]
            elif kind == 5 and (length-2) % 8 == 0:
                option['sack'] = [[u32(transport, j), u32(transport, j+4)] for j in range(i+2, i+length, 8)]
            elif kind == 8 and length == 10:
                option.update(tsval=u32(transport, i+2), tsecr=u32(transport, i+6))
            tcp['options'].append(option)
            i += length
        out['tcp'] = tcp
    elif proto == 17:
        if len(transport) < 8:
            raise ValueError('short UDP header')
        out['udp'] = dict(src_port=u16(transport, 0), dst_port=u16(transport, 2), length=u16(transport, 4))
    elif proto in (1, 58):
        if len(transport) < 8:
            raise ValueError('short ICMP header')
        out['icmp'] = dict(type=transport[0], code=transport[1])
        if proto == 1 and transport[:2] == b'\x03\x04':
            out['icmp']['mtu'] = u16(transport, 6)
        elif proto == 58 and transport[0] == 2:
            out['icmp']['mtu'] = u32(transport, 4)
    return out


def decode(path, output, addresses):
    raw = path.read_bytes()
    formats = {b'\xd4\xc3\xb2\xa1': ('<', 1000000), b'\xa1\xb2\xc3\xd4': ('>', 1000000),
               b'\x4d\x3c\xb2\xa1': ('<', 1000000000), b'\xa1\xb2\x3c\x4d': ('>', 1000000000)}
    endian, scale = formats[raw[:4]]
    _, _, _, _, snaplen, linktype = struct.unpack_from(endian+'HHIIII', raw, 4)
    stats = dict(file=path.name, bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest(), snaplen=snaplen,
                 linktype=linktype, records=0, snap_truncated_records=0, header_errors=0,
                 unparsed_records=0, file_complete=True, first=None, last=None)
    counts = collections.Counter()
    def address(b):
        key = str(ipaddress.ip_address(b))
        if key not in addresses:
            addresses[key] = f'endpoint-{len(addresses)+1}'
        return addresses[key]
    cursor = 24
    with output.open('w') as f:
        while cursor < len(raw):
            if len(raw)-cursor < 16:
                stats['file_complete'] = False
                break
            sec, frac, cap, wire = struct.unpack_from(endian+'IIII', raw, cursor)
            cursor += 16
            if len(raw)-cursor < cap:
                stats['file_complete'] = False
                break
            b = raw[cursor:cursor+cap]; cursor += cap
            stats['records'] += 1
            stats['snap_truncated_records'] += cap < wire
            stamp = f'{sec}.{frac:0{6 if scale==1000000 else 9}d}'
            row = dict(packet=stats['records'], timestamp=stamp, captured_length=cap, wire_length=wire)
            stats['first'] = stats['first'] or stamp; stats['last'] = stamp
            try:
                if linktype == 276:
                    proto, offset = u16(b, 0), 20
                    row.update(interface_index=u32(b, 4), packet_type=b[10])
                elif linktype == 113:
                    proto, offset = u16(b, 14), 16
                    row['packet_type'] = u16(b, 0)
                elif linktype == 1:
                    proto, offset = u16(b, 12), 14
                else:
                    raise ValueError('unsupported link type')
                if proto in (0x800, 0x86dd):
                    row.update(decode_ip(b[offset:], address))
                    counts[str(row.get('protocol', 'fragment'))] += 1
                else:
                    row['non_ip_ethertype'] = proto
                    stats['unparsed_records'] += 1
            except (IndexError, struct.error, ValueError) as e:
                row['header_error'] = str(e)
                stats['header_errors'] += 1
            f.write(json.dumps(row, separators=(',', ':'))+'\n')
    stats['protocol_records'] = dict(counts)
    return stats


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--out', type=Path, required=True)
    p.add_argument('captures', nargs='+', type=Path)
    args = p.parse_args(); args.out.mkdir(parents=True, exist_ok=True)
    addresses = {}; results = []
    for source in args.captures:
        results.append(decode(source, args.out/(source.stem+'.headers.jsonl'), addresses))
    # Address map stays private; only headers with campaign-stable tokens escape.
    print(json.dumps(results, indent=2))

if __name__ == '__main__':
    main()
