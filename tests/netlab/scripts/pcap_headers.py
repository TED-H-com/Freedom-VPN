"""TEST ONLY pcap decoding; export headers and opaque UDP matching digests only.

Raw pcaps stay private. Reject truncated/unknown records rather than silently
claiming complete capture. No dependency on a packet payload decoder.
"""
import collections
import hashlib
from pathlib import Path
import socket
import struct


def decode(path):
    raw = Path(path).read_bytes()
    if raw[:4] != b'\xd4\xc3\xb2\xa1':
        raise ValueError('requires microsecond little-endian pcap')
    link = struct.unpack_from('<I', raw, 20)[0]
    at, rows = 24, []
    while at < len(raw):
        sec, usec, size, original = struct.unpack_from('<IIII', raw, at)
        at += 16
        frame = raw[at:at+size]
        at += size
        if len(frame) != size or size != original:
            raise ValueError('truncated pcap packet')
        row = dict(packet=len(rows)+1, wall_ns=str(sec*10**9+usec*1000), captured=size)
        if link == 276:  # Linux cooked v2: retains ifindex and direction.
            proto = struct.unpack_from('!H', frame)[0]
            row.update(ifindex=struct.unpack_from('!I', frame, 4)[0], direction='out' if frame[10] == 4 else 'in')
            p = frame[20:]
        elif link == 113:
            proto = struct.unpack_from('!H', frame, 14)[0]
            row['direction'] = 'out' if struct.unpack_from('!H', frame)[0] == 4 else 'in'
            p = frame[16:]
        else:
            raise ValueError('requires Linux cooked capture, got '+str(link))
        row['ethertype'] = proto
        if proto == 0x800:
            ihl = (p[0] & 15)*4
            frag = struct.unpack_from('!H', p, 6)[0]
            row.update(family=4, src=socket.inet_ntop(socket.AF_INET,p[12:16]), dst=socket.inet_ntop(socket.AF_INET,p[16:20]),
                       ip_length=struct.unpack_from('!H',p,2)[0], ip_id=struct.unpack_from('!H',p,4)[0],
                       df=bool(frag & 0x4000), more_fragments=bool(frag & 0x2000), fragment_offset=(frag & 8191)*8)
            nxt, body = p[9], p[ihl:row['ip_length']]
            if row['fragment_offset']:
                nxt = -1
        elif proto == 0x86dd:
            row.update(family=6, src=socket.inet_ntop(socket.AF_INET6,p[8:24]), dst=socket.inet_ntop(socket.AF_INET6,p[24:40]),
                       ip_length=40+struct.unpack_from('!H',p,4)[0])
            nxt, pos = p[6], 40
            while nxt in (0, 43, 60, 44):
                old = nxt
                nxt = p[pos]
                if old == 44:
                    frag = struct.unpack_from('!H',p,pos+2)[0]
                    row.update(fragment_offset=frag & 0xfff8, more_fragments=bool(frag & 1), fragment_id=struct.unpack_from('!I',p,pos+4)[0])
                    pos += 8
                    if row['fragment_offset']:
                        nxt = -1
                        break
                else:
                    pos += (p[pos+1]+1)*8
            body = p[pos:row['ip_length']]
        else:
            rows.append(row)
            continue
        row['protocol'] = nxt
        row['datagram_sha256'] = hashlib.sha256(p).hexdigest()
        if nxt == 17:
            sport,dport,length = struct.unpack_from('!HHH',body)
            row.update(sport=sport,dport=dport,udp_length=length,udp_sha256=hashlib.sha256(body).hexdigest())
            if dport == 9000 and sport == 40021 and len(body) == 16:
                row['probe_sequence'] = struct.unpack_from('!Q', body, 8)[0]
        elif nxt == 6:
            sport,dport,seq,ack = struct.unpack_from('!HHII',body)
            head = (body[12] >> 4)*4
            row.update(sport=sport,dport=dport,seq=seq,ack=ack,flags=body[13],window=struct.unpack_from('!H',body,14)[0],tcp_data_length=len(body)-head)
            opts, i = body[20:head], 0
            while i < len(opts):
                kind = opts[i]
                if kind == 0:
                    break
                if kind == 1:
                    i += 1
                    continue
                size = opts[i+1]
                if size < 2:
                    raise ValueError('malformed TCP option')
                if kind == 2:
                    row['mss'] = struct.unpack_from('!H',opts,i+2)[0]
                if kind == 5:
                    row['sack'] = [list(struct.unpack_from('!II',opts,j)) for j in range(i+2,i+size,8)]
                i += size
        elif nxt in (1,58):
            row.update(icmp_type=body[0],icmp_code=body[1])
            if nxt == 58 and body[0] == 2:
                row['ptb_mtu'] = struct.unpack_from('!I',body,4)[0]
        rows.append(row)
    if at != len(raw):
        raise ValueError('partial record')
    return rows


def compare_udp(sent, received):
    """Multiset match handles repeated packets without hiding duplicates."""
    available = collections.Counter(r.get('datagram_sha256',r.get('udp_sha256')) for r in received)
    sizes = {}
    absent = []
    for row in sent:
        item = sizes.setdefault(str(row['ip_length']), dict(sent=0,arrived=0,absent=0))
        item['sent'] += 1
        key = row.get('datagram_sha256',row.get('udp_sha256'))
        if available[key]:
            available[key] -= 1
            item['arrived'] += 1
        else:
            item['absent'] += 1
            absent.append(row['packet'])
    return dict(sizes=sizes,absent_sender_packets=absent,unmatched_receiver_packets=sum(available.values()))
