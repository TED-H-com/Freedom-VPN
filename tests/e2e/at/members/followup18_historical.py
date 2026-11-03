"""Additive explanation of run 81b638e8; never changes historical evidence."""
from collections import Counter
import hashlib
import json
from pathlib import Path
import sys

from d146_evidence import mss
sys.path.insert(0, str(Path.cwd()/'tests/e2e'))
from analyze_native_capture import wall_ns
from decimal import Decimal

root=Path('docs/e2e/at/members')
base=root/'history/follow-up-17-captures/81b638e8-be59-4996-9594-14dd402ebb86'
rows=[]
pins={}
for name in ['matrix1280','ipv6-asymmetric']:
    document=base/(name+'.json')
    doc=json.loads(document.read_text())
    notes={o['name']:json.loads(o['detail']) for o in doc['observations']}
    transfer=notes['transfer'];port=transfer['bulk']['endpoint']['local_port']
    capture=notes['capture']
    before=next(s['points'] for s in capture['samples'] if s['label']=='before-bulk')
    after=next(s['points'] for s in capture['samples'] if s['label']=='after-bulk')
    lo=min(wall_ns(v['end']['wall']) for v in before.values());hi=max(wall_ns(v['start']['wall']) for v in after.values())
    syns={};payloads={};headers={}
    for role in ['member','origin']:
        path=base/'headers'/name/(role+'.headers.jsonl')
        pins[str(path.relative_to(root))]=hashlib.sha256(path.read_bytes()).hexdigest()
        index={link['ifindex']:link['ifname'] for link in before[role]['links']}
        packets=[json.loads(line) for line in path.read_text().splitlines()]
        stream=[p for p in packets if 'tcp' in p and {p['tcp']['src_port'],p['tcp']['dst_port']}=={9446,port} and lo<=int(Decimal(p['timestamp'])*10**9)<=hi and (role!='member' or index[p['interface_index']]=='fvnb0')]
        syns[role]=[dict(packet=p['packet'],timestamp=p['timestamp'],sent=p['packet_type']==4,tcp=p['tcp'],mss=mss(p)) for p in stream if p['tcp']['flags']&2]
        reverse=[p for p in stream if p['tcp']['src_port']==9446 and p['tcp']['length']>0]
        payloads[role]=dict(Counter(str(p['tcp']['length']) for p in reverse))
        headers[role]=dict(Counter(f"IP {p['ip_length']}; IP header {p['ip_header_length']}; TCP header {p['tcp']['header_length']}" for p in reverse))
    analysis=json.loads((base/(name+'-analysis.json')).read_text())
    pins[str(document.relative_to(root))]=hashlib.sha256(document.read_bytes()).hexdigest()
    rows.append(dict(row=name,run_id=doc['run_id'],source_commit=doc['source_commit'],historical_status=doc['status'],
        selected=transfer['selected_exit_before'],bulk=transfer['bulk'],endpoint_handshakes=syns,
        reverse_tcp_payload_lengths=payloads,reverse_inner_packet_lengths=headers,reverse_outer=analysis['reverse'],
        ptb=analysis['ptb'],fragmentation_needed=analysis['fragmentation_needed'],
        explanation='Origin emits one ECHO prefix plus each 1001-byte fixture line per write (1006 bytes). Captured reverse segments are 1006 bytes, independent of the larger advertised MSS ceiling. Inner 20+32+1006=1058 bytes pads to 1072, then WireGuard/UDP/outer IPv4 adds 60 (1132) or IPv6 adds 80 (1152). The matching sizes/counts are consistent with line-sized writes; opaque ciphertext is not mapped to inner segments by timing. No clamp was present or inferred, and this finite transfer did not exercise an oversized reverse segment.' ))
for name in ['tests/e2e/cmd/fvpn-e2e-fixture-probe/main.go','tests/e2e/cmd/fvpn-e2e-origin/member_echo.go']:
    pins[name]=hashlib.sha256(Path(name).read_bytes()).hexdigest()
(root/'F18-historical-81b638e8-MSS.json').write_text(json.dumps(dict(scope='Additive header analysis. Every historical outcome, including failed F8-02 cases and blocked receive-only rows, is unchanged.',pins=pins,rows=rows),indent=2)+'\n')
for r in rows:print(r['row'],r['reverse_tcp_payload_lengths'],{role:[s['mss'] for s in values] for role,values in r['endpoint_handshakes'].items()})
