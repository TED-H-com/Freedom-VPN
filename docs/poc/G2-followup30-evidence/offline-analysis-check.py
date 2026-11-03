#!/usr/bin/env python3
"""TEST ONLY synthetic guard logs against retained TCP paths; never a live claim."""
import gzip
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
D=Path(__file__).resolve().parent;ROOT=D.parents[2]
sys.path.insert(0,str(ROOT/'tests/netlab/scripts'))
import tcp_analysis
source=ROOT/'docs/poc/G2-followup28-evidence/window/reload/TestAT10_AT13_ReloadRevokeBaseline'
analysis=tcp_analysis.analyze(source)

def read(name):
    p=source/name
    return p.read_bytes() if p.exists() else gzip.decompress(Path(str(p)+'.gz').read_bytes())

with tempfile.TemporaryDirectory(prefix='f30-analysis-') as temp:
    work=Path(temp)
    for name in ('at10-apply.jsonl','stream-a1.log.jsonl','at10-tcp-meta.json',*[f'at10-tcp-{role}{ext}' for role in ('origin','node','responder') for ext in ('.jsonl','.err')]):
        (work/name).write_bytes(read(name))
    applies=[json.loads(x) for x in read('at10-apply.jsonl').splitlines()]
    for a in applies:
        for b in a['batches']:
            if b['kind']!='guard_install':continue
            lines=[]
            for line in b['test_only_script'].splitlines():
                if line.startswith('add rule inet fvpn fv_deadline_guard ') and line.endswith(' drop'):
                    direction='iifname' if ' iifname ' in line else 'oifname'
                    line=line[:-5]+f' log prefix "f29g{a["generation"]}-{direction}" group 29 snaplen 128 queue-threshold 1 drop'
                lines.append(line)
            b['test_only_script']='\n'.join(lines)+'\n'
            b['test_only_sha256']=hashlib.sha256(b['test_only_script'].encode()).hexdigest()
    (work/'at10-apply.jsonl').write_text(''.join(json.dumps(a)+'\n' for a in applies))
    logs=[]
    for s in analysis['segments']:
        if s['direction']!='request' or 'node_in_fv-fh' not in s['stages'] or 'node_out_fv-ih' in s['stages']:continue
        r=dict(s['stages']['node_in_fv-fh'])
        bracket=next(h for h in r['guard_positions'] if h['phase'] in ('install','guard_certain','release'))
        r.update(protocol=6,generation=bracket['generation'],rule='iifname',packet=len(logs)+1,pcap_wall_ns=r['wall_ns'],pcap_precision_ns=1000,kernel_wall_ns=str(int(r['wall_ns'])//1000*1000))
        logs.append(r)
    assert len(logs)==11
    (work/'at10-guard.meta.json').write_text(json.dumps(dict(test_only=True,synthetic=True,stats=dict(captured=11,received_by_filter=11,dropped_by_kernel=0))))
    cases=[]
    def run(name,records):
        (work/'at10-guard.jsonl').write_text(''.join(json.dumps(r)+'\n' for r in records))
        target=work/(name+'.json')
        subprocess.run([sys.executable,'-B',str(D/'guard-analysis.py'),str(work),str(target)],check=True,stdout=subprocess.DEVNULL)
        return json.loads(target.read_text())
    r=run('complete',logs);assert r['all_missing_attributed'] and r['exact_matches']==11 and r['logged_tcp_with_egress']==0
    cases.append('11 synthetic positive logs map one-to-one to the 11 retained missing egresses')
    r=run('absent',logs[:-1]);assert not r['all_missing_attributed']
    cases.append('missing log leaves missing segment unattributed')
    r=run('duplicate',logs+[dict(logs[0],packet=12)]);assert not r['all_missing_attributed']
    cases.append('duplicate logs cannot silently become one attribution')
    reply=next(s for s in analysis['segments'] if s['direction']=='reply' and 'node_in_fv-ih' in s['stages'] and any(h['phase']=='release' for h in s['stages']['node_in_fv-ih']['guard_positions']))
    r=dict(reply['stages']['node_in_fv-ih']);bracket=next(h for h in r['guard_positions'] if h['phase']=='release')
    r.update(protocol=6,destination='100.64.1.1',generation=bracket['generation'],rule='oifname',packet=12,pcap_wall_ns=r['wall_ns'],pcap_precision_ns=1000)
    result=run('contradictory-reply',logs+[r]);assert result['exact_matches']==12 and result['logged_tcp_with_egress']==1
    cases.append('reverse DNAT identity matches; contradictory egress remains explicit')
    out=dict(test_only=True,synthetic=True,not_a_measurement=True,cases=cases,passed=len(cases))
    (D/'offline-analysis-check.json').write_text(json.dumps(out,indent=2)+'\n')
    print(json.dumps(out,indent=2))
