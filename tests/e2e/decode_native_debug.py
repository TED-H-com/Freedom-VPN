#!/usr/bin/env python3
"""TEST ONLY allowlisted native events. No free text, peer keys or DNS names."""
import argparse
import hashlib
import ipaddress
import json
from pathlib import Path
import re

EVENTS = {
 'WireGuard Proxy Factory will produce loopback proxy': 'loopback_factory',
 'WireGuard Proxy Factory will produce UDP proxy': 'udp_factory',
 'relayed conn added to wg proxy store:': 'relay_proxy_added',
 'configure WireGuard endpoint to:': 'p2p_endpoint_configure',
 'redirect packets from relayed conn to WireGuard': 'relay_redirect_as_endpoint',
 'configure up WireGuard and wait for handshake': 'wg_configure_wait',
 'remove relayed conn from store by address:': 'relay_proxy_removed',
 'failed to read from relayed conn (endpoint:': 'relay_read_closed_or_failed',
 'upstreams failed for domain=': 'dns_upstreams_failed',
 'write error response for domain=': 'dns_error_write_failed',
 'failed to write DNS response for question domain=': 'dns_success_write_failed',
}

def decode(raw):
    out=[]
    for number,line in enumerate(raw.decode(errors='replace').splitlines(),1):
        prefix=re.match(r'^(\d{4}-\d\d-\d\dT[0-9:.]+Z) (\w+) ',line)
        source=re.search(r' (client/[a-zA-Z0-9/_]+\.go:\d+): ',line)
        if not prefix or not source: continue
        for text,event in EVENTS.items():
            if text not in line: continue
            item=dict(line=number,at=prefix[1],level=prefix[2],source=source[1],event=event)
            if event=='p2p_endpoint_configure':
                endpoint=line.split(text,1)[1].strip()
                try:
                    host,port=endpoint.rsplit(':',1); ip=ipaddress.ip_address(host.strip('[]'))
                    item.update(family='ipv'+str(ip.version),address='addr-'+hashlib.sha256(str(ip).encode()).hexdigest()[:16],port=int(port))
                except ValueError: item['endpoint_unparsed']=True
            if event=='dns_upstreams_failed':
                item['timeout_mentions']=line.count('timeout')
                item['wire_id']=re.search(r'dns_id=([0-9a-f]+)',line).group(1) if re.search(r'dns_id=([0-9a-f]+)',line) else None
            out.append(item)
    return out

if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('logs',type=Path,nargs='+');p.add_argument('--out',type=Path,required=True);p.add_argument('--record',type=Path,required=True)
    args=p.parse_args(); source=json.loads(args.record.read_text())
    observations=[]
    for path in args.logs:
        raw=path.read_bytes()
        observations.append(dict(name=path.stem,detail=json.dumps(dict(bytes=len(raw),sha256=hashlib.sha256(raw).hexdigest(),events=decode(raw)))))
    source.update(id='native-debug-events',title='Pinned native client allowlisted debug events',status='pass',observations=observations)
    args.out.write_text(json.dumps(source,indent=2)+'\n')
