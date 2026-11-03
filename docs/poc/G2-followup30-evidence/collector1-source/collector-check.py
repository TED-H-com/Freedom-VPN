#!/usr/bin/env python3
"""TEST ONLY run inside owned lab: eight known logs and one negative control."""
import json
from pathlib import Path
import socket
import struct
import subprocess
import sys
import time
sys.path.insert(0,'/opt/netlab/scripts')
import guard_observe as guard
from at12_observe import stamp

BASE='/scratch/collector'

def inject(direction):
    source,dest,sport,dport=('100.64.1.1','203.0.113.10',49030,49031) if direction=='up' else ('203.0.113.10','100.64.1.1',49031,49030)
    count=5 if direction=='up' else 3
    rows=[]
    with socket.socket(socket.AF_INET,socket.SOCK_RAW,socket.IPPROTO_RAW) as sock:
        for n in range(count+(direction=='up')):
            negative=n==count
            ident=(30001 if direction=='up' else 30011)+n
            port=49032 if negative else dport
            udp=struct.pack('!HHHH',sport,port,9+n,0)+bytes(n+1)
            ip=struct.pack('!BBHHHBBH4s4s',0x45,0,20+len(udp),ident,0,64,17,0,socket.inet_aton(source),socket.inet_aton(dest))
            start=stamp();sent=sock.sendto(ip+udp,(dest,0));end=stamp()
            rows.append(dict(source=source,destination=dest,source_port=sport,destination_port=port,ip_id=ident,ip_length=sent,protocol=17,data_length=n+1,expected_log=not negative,rule='iifname' if direction=='up' else 'oifname',start=start,end=end))
    print(json.dumps(rows))

if len(sys.argv)>1:
    inject(sys.argv[1]);sys.exit(0)
script='''add chain inet fvpn fv_collector_check { type filter hook forward priority -400; policy accept; }
add rule inet fvpn fv_collector_check iifname "fv-fh" ip saddr 100.64.1.1 ip daddr 203.0.113.10 udp sport 49030 udp dport 49031 counter log prefix "f29g30000-iifname" group 29 snaplen 128 queue-threshold 1 drop
add rule inet fvpn fv_collector_check oifname "fv-fh" ip saddr 203.0.113.10 ip daddr 100.64.1.1 udp sport 49031 udp dport 49030 counter log prefix "f29g30000-oifname" group 29 snaplen 128 queue-threshold 1 drop
add rule inet fvpn fv_collector_check ip saddr 100.64.1.1 udp sport 49030 udp dport 49032 counter drop
'''
Path(BASE+'-rules.nft').write_text(script)
subprocess.run(['nft','-f','-'],input=script,text=True,check=True)
guard.start(BASE)
injected=[]
try:
    for direction,ns in (('up','fv-a1'),('down','fv-inet')):
        injected.extend(json.loads(subprocess.check_output(['ip','netns','exec',ns,sys.executable,'-B',__file__,direction],text=True)))
    Path(BASE+'-injected.json').write_text(json.dumps(injected,indent=2)+'\n')
    time.sleep(.3)
finally:
    guard.stop(BASE)
    Path(BASE+'-readback.json').write_bytes(subprocess.check_output(['nft','-j','list','chain','inet','fvpn','fv_collector_check']))
    subprocess.run(['nft','delete','chain','inet','fvpn','fv_collector_check'],check=True)
rows=[json.loads(x) for x in Path(BASE+'.jsonl').read_text().splitlines()]
keys=('source','destination','source_port','destination_port','ip_id','ip_length','protocol','data_length','rule')
expected=[r for r in injected if r['expected_log']]
matched=[dict(injected=r,records=[x['packet'] for x in rows if all(x.get(k)==r[k] for k in keys)]) for r in expected]
meta=json.loads(Path(BASE+'.meta.json').read_text())
result=dict(test_only=True,not_a_measurement=True,expected=8,decoded=len(rows),matches=matched,
            precision_ns=[r['pcap_precision_ns'] for r in rows],kernel_timestamp_present=['kernel_wall_ns' in r for r in rows],
            receiver=meta['stats'],pass_check=len(rows)==8 and all(len(r['records'])==1 for r in matched))
Path(BASE+'-result.json').write_text(json.dumps(result,indent=2)+'\n')
assert result['pass_check'],result
print(json.dumps(result,indent=2))
