"""Checked UDP projection of a combined TCP/UDP header-only tcpdump trace.

The D-120 analyzer stays byte-identical. Retain the combined original and its
real capture statistics; reject any unparsed TCP or UDP packet before deriving
the legacy UDP input. No payload decoder, pcap file, or payload output is used.
"""
import hashlib
import re

from reload_analysis import parse_capture


def udp_projection(text, stats):
    blocks = []
    for line in text.splitlines(keepends=True):
        if re.match(r'^\d+\.\d+\s+\S+\s+(In|Out)\s+IP \(', line):
            blocks.append([line])
        elif blocks and line[:1].isspace():
            blocks[-1].append(line)
        elif line.strip():
            raise ValueError('unrecognized combined capture line')
    captured = re.search(r'(\d+) packets captured', stats)
    drops = re.search(r'(\d+) packets dropped by kernel', stats)
    if not captured or not drops or int(drops[1]) or int(captured[1]) != len(blocks):
        raise ValueError('incomplete combined capture')
    udp, tcp = [], []
    for lines in blocks:
        block = ''.join(lines)
        if len(lines) != 2:
            raise ValueError('incomplete or unexpected packet header')
        if 'proto UDP (17)' in lines[0]:
            parse_capture(block, '1 packets captured\n0 packets dropped by kernel\n')
            udp.append(block)
        elif 'proto TCP (6)' in lines[0] and re.fullmatch(
                r'\s*\d+\.\d+\.\d+\.\d+\.\d+ > \d+\.\d+\.\d+\.\d+\.\d+: '
                r'Flags \[[^\]]*\], .*\blength \d+\s*', lines[1]):
            tcp.append(block)
        else:
            raise ValueError('unparsed combined packet header')
    projected = ''.join(udp)
    projected_stats = ('# Derived UDP projection; original combined statistics in -headers.err\n'
                       f'{len(udp)} packets captured\n0 packets dropped by kernel\n')
    # The unchanged consumer independently checks every UDP record again.
    parse_capture(projected, projected_stats)
    metadata = dict(complete=True, combined_packets=len(blocks), udp_packets=len(udp),
                    tcp_packets=len(tcp), dropped_by_kernel=0,
                    headers_sha256=hashlib.sha256(text.encode()).hexdigest(),
                    stats_sha256=hashlib.sha256(stats.encode()).hexdigest(),
                    udp_sha256=hashlib.sha256(projected.encode()).hexdigest())
    return projected, projected_stats, metadata
