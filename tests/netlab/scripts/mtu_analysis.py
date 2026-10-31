#!/usr/bin/env python3
"""Recompute F8-02 header observations and counter deltas from retained files."""
import argparse
from decimal import Decimal
import json
from pathlib import Path
import re

from mtu_measure import capture_check


def packets(text):
    rows = []
    for chunk in re.split(r'(?=^\d+\.\d+ )', text, flags=re.M):
        h = re.match(r'^(\d+\.\d+)\s+(\S+)\s+(In|Out|M|B|P)\s+IP(6)? ', chunk)
        if h:
            rows.append(dict(wall_ns=str(int(Decimal(h[1])*10**9)), interface=h[2], direction=h[3],
                             family=6 if h[4] else 4, text=chunk))
    return rows


def classes(text):
    return {m[1]: dict(bytes=int(m[2]), packets=int(m[3]), dropped=int(m[4])) for m in
            re.finditer(r'class htb (\S+)[^\n]*\n Sent (\d+) bytes (\d+) pkt \(dropped (\d+)', text)}


def counters(text):
    return {m[1]: dict(packets=int(m[2]), bytes=int(m[3])) for m in
            re.finditer(r'counter (\w+) \{[^}]*?packets (\d+) bytes (\d+)', text, re.S)}


def diff(before, after):
    return {k: {n: v - before.get(k, {}).get(n, 0) for n, v in value.items()}
            for k, value in after.items() if value != before.get(k)}


def analyze(root):
    cap = {}
    for role in ('member', 'node', 'origin'):
        text, stats = (root/(role+'.txt')).read_text(), (root/(role+'.err')).read_text()
        if not capture_check(text, stats)['ok']:
            raise ValueError('incomplete capture: '+role)
        cap[role] = packets(text)
    rows = json.loads((root/'measurements.json').read_text())
    previous = json.loads((root/'initial.json').read_text())
    out = []
    for r in rows:
        before, after = r.get('before', previous), r['after']
        previous = after
        observations, mss = [], []
        for role, ps in cap.items():
            for p in ps:
                if not int(r['start_ns']) <= int(p['wall_ns']) <= int(r['end_ns']):
                    continue
                base = dict(role=role, **{k: v for k, v in p.items() if k != 'text'})
                if 'unreachable - need to frag' in p['text'] or 'packet too big' in p['text']:
                    observations.append(base)
                match = re.search(r'options \[mss (\d+)', p['text'])
                if match:
                    mss.append(dict(base, mss=int(match[1]), flags=re.search(r'Flags \[([^]]+)\]', p['text'])[1]))
        result = r['result']['stdout']
        try:
            result = json.loads(result)
        except ValueError:
            pass
        out.append(dict(name=r['name'], result=result, errors=observations, mss=mss,
                        nft_delta=diff(counters(before['nft']), counters(after['nft'])),
                        tc_delta={dev: diff(classes(before['tc'][dev]['class']), classes(after['tc'][dev]['class']))
                                  for dev in ('fv-fh', 'fv-ih')}))
    return dict(test_only=True, phases=out)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('directory', type=Path)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    result = analyze(args.directory)
    args.output.write_text(json.dumps(result, indent=2)+'\n')
    for row in result['phases']:
        print(row['name'])
        if isinstance(row['result'], dict):
            print(' result:', json.dumps(row['result'], sort_keys=True))
        else:
            print(' result:', row['result'].strip().replace('\n', ' | '))
        for role in ('member', 'node', 'origin'):
            hops = sorted({f"{p['interface']}:{p['direction']}" for p in row['errors'] if p['role'] == role})
            advertised = sorted({f"{p['interface']}:{p['direction']}:{p['flags']}={p['mss']}" for p in row['mss'] if p['role'] == role})
            if hops or advertised:
                print(' ', role, 'PMTU errors:', ','.join(hops) or 'none', 'MSS:', ','.join(advertised) or 'none')
        print(' nft delta:', json.dumps(row['nft_delta'], sort_keys=True))
        print(' tc delta:', json.dumps(row['tc_delta'], sort_keys=True))
