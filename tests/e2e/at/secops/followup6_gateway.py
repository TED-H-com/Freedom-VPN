#!/usr/bin/env python3
"""Read-only socket/FD samples, only the assigned gateway; never raw addresses."""
import subprocess
import time
from run import PUBLIC, write_json, utc

SCRIPT = r'''printf 'workers '; for p in /proc/[0-9]*/cmdline; do tr '\000' ' ' < "$p" 2>/dev/null; printf '\n'; done | grep -c '^nginx: worker process'
awk 'FNR>1 {states[$4]++;total++} END {print "tcp_total",total;for(s in states) print "tcp_state_"s,states[s]}' /proc/net/tcp /proc/net/tcp6
printf 'fd_entries '; for p in /proc/[0-9]*/comm; do if [ "$(cat "$p" 2>/dev/null)" = nginx ]; then d=${p%/comm}; ls "$d/fd" 2>/dev/null; fi; done | wc -l'''


def main():
    rows=[]
    record={'started_at':utc(),'scope':'30-second samples, with actual observer start/end times recorded; boundaries are also recorded by each Go probe; socket states include gateway namespace connections, no private addresses','command':['docker','exec','fvpn-w4-secops-gateway','sh','-c','aggregate process, TCP state and nginx FD inventory'],'samples':rows}
    for _ in range(100):
        names=subprocess.run(['docker','ps','--filter','name=^fvpn-w4-secops-member-','--format','{{.Names}}'],capture_output=True,text=True,timeout=10)
        container=subprocess.run(['docker','inspect','--format','{{.Id}}','fvpn-w4-secops-gateway'],capture_output=True,text=True,timeout=10)
        raw=subprocess.run(['docker','exec','fvpn-w4-secops-gateway','sh','-c',SCRIPT],capture_output=True,text=True,timeout=15)
        fields=raw.stdout.split()
        data=dict(zip(fields[::2],fields[1::2])) if len(fields)%2==0 else {}
        rows.append({'utc':utc(),'gateway_instance':container.stdout.strip() if container.returncode==0 else None,'native_peer_containers':len(names.stdout.splitlines()),'read_ok':raw.returncode==0,'aggregates':data})
        write_json(PUBLIC/'gateway-inventory.json',record)
        runpath=PUBLIC/'followup6-run.json'
        if runpath.exists():
            import json
            run=json.loads(runpath.read_text())
            if run.get('finished_at'):
                break
        time.sleep(30)
    record['finished_at']=utc();write_json(PUBLIC/'gateway-inventory.json',record)


if __name__=='__main__':main()
