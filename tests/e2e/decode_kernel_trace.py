#!/usr/bin/env python3
"""TEST ONLY closed nftrace metadata export; never copies rule expressions."""
import hashlib
import ipaddress
import re


def token(value):
    return 'addr-'+hashlib.sha256(str(ipaddress.ip_address(value)).encode()).hexdigest()[:16]


def decode(raw):
    events = []
    unparsed = 0
    for number, line in enumerate(raw.decode(errors='replace').splitlines(), 1):
        match = re.match(r'^trace id ([0-9a-f]+) (netdev|inet|ip6|ip) ([\w-]+) ([\w-]+) (packet|rule|policy|verdict|return|continue|enter)', line)
        if not match:
            unparsed += 1
            continue
        item = dict(line=number, trace_id=match[1], family=match[2], table=match[3], chain=match[4], kind=match[5])
        for key, pattern in {
            'input_interface': r' iif "([\w.-]+)"',
            'output_interface': r' oif "([\w.-]+)"',
            'verdict': r'(?:\(verdict |policy )([a-z]+)',
        }.items():
            value = re.search(pattern, line)
            if value:
                item[key] = value[1]
        if match[5] == 'packet':
            for field in ['saddr', 'daddr']:
                value = re.search(r' ip6 '+field+r' ([0-9a-fA-F:]+)', line)
                if value:
                    item[field] = token(value[1])
            for field in ['sport', 'dport', 'length', 'hoplimit']:
                value = re.search(r' (?:udp |ip6 |th )?'+field+r' (\d+)', line)
                if value:
                    item[field] = int(value[1])
        events.append(item)
    return dict(bytes=str(len(raw)), sha256=hashlib.sha256(raw).hexdigest(), lines=len(raw.splitlines()), unparsed_lines=unparsed, events=events)
