#!/usr/bin/env python3
"""TEST ONLY: reproducible descriptive quartet summaries, no exclusion/retry."""
import gzip
import json
import math
from pathlib import Path

ROOT=Path(__file__).resolve().parent

def read(path):
    p=ROOT/path
    return p.read_bytes() if p.exists() else gzip.decompress(Path(str(p)+'.gz').read_bytes())

def doc(path):return json.loads(read(path))
def dist(values):
    values=sorted(values)
    if not values:return None
    return dict(n=len(values),**{key:values[min(len(values)-1,math.ceil(p*len(values))-1)]/1e6 for key,p in [('p50_ms',.5),('p90_ms',.9),('p99_ms',.99),('max_ms',1)]})
def duration(row):return int(row['end_ns'])-int(row['start_ns'])

def sizes(raw):
    out={}
    for item in raw['nftables']:
        s=item.get('set',{})
        if s.get('name') in ('denied_v4','host_v4'):
            out[s['name']]=len(s.get('elem',[]))
    return out

results=[];pools={}
for leg in ['A1','B1','A2','B2']:
    base=Path('comparison')/leg
    result=dict(leg=leg,cost={},reload={})
    for size in ['spec','expanded']:
        path=base/'cost/TestISO01_F27_VerifyCost'/size
        samples=doc(path/'samples.json')
        assert len(samples)==100
        vals=[];gaps=[]
        for sample in samples:
            v=sample['result']['timing']['verifications'][0]
            vals.append(duration(v))
            gap=int(v['tc_read']['start_ns'])-int(v['nft_read']['end_ns'])-duration(v['precision_read'])
            assert gap>=0
            gaps.append(gap)
        result['cost'][size]=dict(installed_elements=sizes(doc(path/'installed.json')),
            exit_codes=[x['exit'] for x in samples],full_verify=dist(vals),nft_parse_check_gap=dist(gaps))
        for name,items in [('full_verify',vals),('nft_parse_check_gap',gaps)]:
            pools.setdefault(leg[0]+'/'+size+'/'+name,[]).extend(items)
    path=base/'reload/TestAT10_AT13_ReloadRevokeBaseline'
    analysis=doc(path/'at10-analysis.json')
    windows=analysis['windows']
    applies=[json.loads(line) for line in read(path/'at10-apply.jsonl').decode().splitlines() if line.strip()]
    input_sizes=[]
    for row in applies:
        m=row['test_only_input_model']
        item=dict(denied_prefixes=len(m['denied_v4']),extra_denied_prefixes=len(m['extra_denied_v4']),
                  host_prefixes=len(m['host_v4']),snat_already_in_host=m['datapath']['snat']+'/32' in m['host_v4'])
        if item not in input_sizes:input_sizes.append(item)
    full=[]
    for row in applies:
        # The trace's nested timing is the production Apply instrumentation.
        for v in (row.get('timing') or {}).get('verifications',[]):
            full.append(duration(v))
    result['reload']=dict(exit_code=doc(base/'reload.json')['exit_code'],summary=analysis['summary'],
        input_sizes=input_sizes,
        guard=dist([w['upper_ns'] for w in windows]),render_to_installed=dist([w['render_to_installed_ns'] for w in windows]),
        apply_verification=dist(full),failed_apply_generations=[r['generation'] for r in applies if not r.get('verified')])
    for name,values in [('guard',[w['upper_ns'] for w in windows]),('render_to_installed',[w['render_to_installed_ns'] for w in windows]),('apply_verification',full)]:
        pools.setdefault(leg[0]+'/reload/'+name,[]).extend(values)
    admission=doc(base/'admission.json')
    pre=admission['rows'][-1];post=doc(base/'after-host.json')
    result['host']=dict(pre_setup_wait_seconds=doc(base/'pre-setup.json')['wait_seconds'],quiet_wait_seconds=admission['wait_seconds'],entry=pre,exit=post)
    results.append(result)
summary=dict(test_only=True,percentiles='nearest rank; no excluded attempts',legs=results,pools={k:dist(v) for k,v in pools.items()},
    gap_definition='tc_read.start - nft_read.end - precision_read duration: includes both nft parses, rule/set checks and local bookkeeping; not a direct CPU profiler',
    inference_limit='One same-host quartet. Full verifier includes subprocess and scheduling cost; descriptive differences do not alone prove a repeatable causal regression.')
(ROOT/'comparison-summary.json').write_text(json.dumps(summary,indent=2)+'\n')
for r in results:
    print(r['leg'],r['cost']['spec']['full_verify'],r['cost']['expanded']['full_verify'],r['reload']['exit_code'],r['reload']['guard'],r['reload']['render_to_installed'])
