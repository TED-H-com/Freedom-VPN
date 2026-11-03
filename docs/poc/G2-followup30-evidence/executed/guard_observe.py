#!/usr/bin/env python3
"""TEST ONLY NFLOG guard verdict evidence. Raw pcaps stay in lab scratch."""
import json
import os
from pathlib import Path
import re
import signal
import struct
import subprocess
import sys
import time
from at12_observe import headers, stamp
from tcp_observe import validate_stats


def decode(raw):
    precision={b'\x4d\x3c\xb2\xa1':1,b'\xd4\xc3\xb2\xa1':1000}.get(raw[:4])
    if len(raw)<24 or precision is None or struct.unpack_from('<I',raw,20)[0]!=239:
        raise ValueError('requires little-endian NFLOG pcap')
    pos=24; rows=[]
    while pos<len(raw):
        if len(raw)-pos<16: raise ValueError('partial pcap record')
        sec,nano,size,original=struct.unpack_from('<IIII',raw,pos);pos+=16
        packet=raw[pos:pos+size];pos+=size
        if size!=original or len(packet)!=size or nano*precision>=10**9 or size<4:raise ValueError('truncated pcap')
        family,version,group=struct.unpack_from('!BBH',packet)
        if family not in (2,10) or version!=0 or group!=29:raise ValueError('unexpected NFLOG group/family')
        at=4;attrs={}
        while at<len(packet):
            if len(packet)-at<4:raise ValueError('partial NFLOG attribute')
            length,kind=struct.unpack_from('<HH',packet,at)
            if length<4 or at+length>len(packet) or kind in attrs:raise ValueError('invalid NFLOG attribute')
            attrs[kind]=packet[at+4:at+length];at+=(length+3)&~3
        if not all(k in attrs for k in (1,9,10)):raise ValueError('missing verdict identity/payload')
        prefix=attrs[10].rstrip(b'\0').decode('ascii')
        match=re.fullmatch(r'f29g(\d+)-(iifname|oifname)',prefix)
        if not match:raise ValueError('unexpected log prefix')
        proto,hook,_=struct.unpack('!HBB',attrs[1])
        if hook!=2:raise ValueError('not the forward hook')
        row=headers(attrs[9],proto)
        row.update(packet=len(rows)+1,pcap_wall_ns=str(sec*10**9+nano*precision),pcap_precision_ns=precision,
                   generation=match[1],rule=match[2],verdict='drop',group=group,hook=hook)
        if 3 in attrs:
            seconds,micros=struct.unpack('!QQ',attrs[3])
            if micros>=10**6:raise ValueError('invalid kernel timestamp')
            row['kernel_wall_ns']=str(seconds*10**9+micros*1000)
        for kind,key in ((4,'input_index'),(5,'output_index'),(12,'nflog_sequence'),(13,'nflog_global_sequence')):
            if kind in attrs:row[key]=struct.unpack('!I',attrs[kind])[0]
        rows.append(row)
    return rows


def start(base):
    assert not Path(base+'.pid').exists(), 'never reuse a diagnostic capture'
    meta=dict(test_only=True,start=stamp(),group=29)
    # NFLOG does not support nanosecond pcap delivery timestamps on the lab's
    # libpcap backend. Preserve its default microsecond precision explicitly.
    cmd=['tcpdump','--immediate-mode','-U','-nn','-s','0','-i','nflog:29','-w',base+'.pcap']
    with Path(base+'.err').open('x') as err:
        proc=subprocess.Popen(cmd,stdout=subprocess.DEVNULL,stderr=err,start_new_session=True)
    Path(base+'.pid').write_text(str(proc.pid))
    until=time.monotonic()+5
    while 'listening on' not in Path(base+'.err').read_text():
        if proc.poll() is not None or time.monotonic()>until:raise RuntimeError('NFLOG capture not ready')
        time.sleep(.01)
    meta['ready']=stamp();Path(base+'.meta.json').write_text(json.dumps(meta,indent=2)+'\n')


def stop(base):
    os.kill(int(Path(base+'.pid').read_text()),signal.SIGINT)
    until=time.monotonic()+5
    while 'packets dropped by kernel' not in Path(base+'.err').read_text():
        if time.monotonic()>until:raise RuntimeError('NFLOG capture did not finish')
        time.sleep(.01)
    raw=Path(base+'.pcap').read_bytes()
    rows=decode(raw)
    Path(base+'.jsonl').write_text(''.join(json.dumps(r)+'\n' for r in rows))
    meta=json.loads(Path(base+'.meta.json').read_text());meta.update(end=stamp(),stats=validate_stats(Path(base+'.err').read_text(),len(rows)),raw_bytes=len(raw))
    # pcap delivery statistics do not prove kernel NFLOG never omitted a log.
    meta['completeness_limit']='Positive logs identify a rule; absence alone is not proof of no drop. Match all missing TCP segments independently.'
    Path(base+'.meta.json').write_text(json.dumps(meta,indent=2)+'\n');Path(base+'.pcap').unlink()


if __name__=='__main__':
    {'start':start,'stop':stop}[sys.argv[1]](sys.argv[2])
