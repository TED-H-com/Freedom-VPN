from evidence_io import load
"""Derive header-only evidence; preserve original capture snapshots separately."""
from pathlib import Path
import importlib.util,json,subprocess,hashlib
root=Path.cwd()
spec=importlib.util.spec_from_file_location('native_analysis',root/'tests/e2e/analyze_native_capture.py')
mod=importlib.util.module_from_spec(spec);spec.loader.exec_module(mod)
cases={case:load(root/'docs/e2e/at/members'/(case+'.json')) for case in ['AT01','AT02','AT03','AT04','AT05','AT08','AT09','AT20','AT21','AT22']}
run=cases['AT01']['run_id']
assert cases['AT01']['source_commit'].startswith('d49b535a')
base=root/'.agent/tmp/f13-derived'/run
base.mkdir(parents=True,exist_ok=True)
entries=[]
for i,o in enumerate(cases['AT01'].get('forwarding',[])):
 if o.get('native_capture'): entries.append(('initial-selection' if i==0 else 'later-selection-'+str(i),o['native_capture'],o['transfer'], 'pass' if o['counted'] else 'blocked'))
for s in cases.get('AT09',{}).get('steps',[]):
 try:v=json.loads(s['actual'])
 except (ValueError,KeyError):continue
 if isinstance(v,dict) and v.get('native_capture'):
  name={'native ipv4 MTU 1500 transfer':'matrix4','native ipv6 MTU 1500 transfer':'matrix6','native dual MTU 1500 transfer':'matrixDual','native dual MTU 1280 transfer':'matrix1280','native ipv6 MTU 1280 transfer':'ipv6-asymmetric','D-137 supported known-PMTU 1280 transfer':'ipv6-explicit1280'}.get(s['name'])
  if name:entries.append((name,v['native_capture'],v['transfer'],s['status']))
results=[]
for name,capture,transfer,status in entries:
 directory=next(p.parent for p in (root/'.agent/tmp/f13-native'/run).glob('*/capture.json') if json.loads(p.read_text())['id']==capture['id'])
 header=base/'headers'/name
 decoder=['python3','-B','tests/e2e/decode_capture.py','--stable-addresses','--wireguard-headers','--out',str(header)]+[str(directory/(role+'.pcap')) for role in ['member','node','origin']]
 decoded=json.loads(subprocess.check_output(decoder,text=True))
 original=base/(name+'-original-capture.json')
 raw=json.dumps(capture,indent=2)+'\n'
 if original.exists():assert original.read_text()==raw
 else:original.write_text(raw)
 # Label normalization is confined to the derived analyzer input. Original above
 # retains exact labels. Before-transfer is the same owner's readback operation.
 derived=json.loads(json.dumps(capture))
 for sample in derived['samples']:
  if sample['label']=='before-transfer':sample['label']='before-first-transfer'
 doc={'id':name,'run_id':run,'source_commit':cases['AT01']['source_commit'],'status':status,'observations':[{'name':'capture','detail':json.dumps(derived)},{'name':'decode','detail':json.dumps(decoded)},{'name':'transfer','detail':json.dumps(transfer if name.startswith(('initial-selection','later-selection')) else {'bulk':transfer})}]}
 rec=base/(name+'.json');rec.write_text(json.dumps(doc,indent=2)+'\n')
 try:
  if name.startswith('later-selection'):
   result={'id':name,'run_id':run,'source_commit':cases['AT01']['source_commit'],'status':status,'scope':'Later original sleep/wake observation, TCP port 20002; not the first selection','capture_checks':{},'clock_offsets':{},'hops':{},'transfer':transfer}
   before=next(s for s in capture['samples'] if s['label']=='before-transfer')['points']
   for role,check in zip(['member','node','origin'],decoded):
    point=capture['points'][role];packets=[json.loads(line) for line in (header/(role+'.headers.jsonl')).read_text().splitlines()]
    complete=point['statistics_ok'] and point['process_stopped'] and point['kernel_drops']==0 and point['captured']==check['records']==len(packets) and check['header_errors']==0 and check['file_complete'] and check['snap_truncated_records']==0
    assert complete, 'incomplete later-selection capture'
    result['capture_checks'][role]={'complete':complete,'decoded':check['records'],'captured':point['captured'],'kernel_drops':0}
    offset=mod.offsets(capture,role);result['clock_offsets'][role]=offset
    (header/(role+'.bounds.jsonl')).write_text(''.join(json.dumps(mod.bound(packet,offset),separators=(',',':'))+'\n' for packet in packets))
    interfaces={link['ifindex']:link['ifname'] for link in before[role]['links']}
    result['hops'][role]=[dict(mod.bound(packet,offset),interface=interfaces.get(packet['interface_index'],'unknown'),packet_type=packet['packet_type'],tcp=packet['tcp']) for packet in packets if packet.get('tcp',{}).get('src_port')==20002 or packet.get('tcp',{}).get('dst_port')==20002]
  else:result=mod.analyze(base,rec)
 except (AssertionError,KeyError,StopIteration,ValueError) as e:
  result={'id':name,'analysis_unavailable':str(e),'capture':capture,'decode':decoded}
 results.append(result)
 (base/(name+'-analysis.json')).write_text(json.dumps(result,indent=2)+'\n')
 print(name, 'complete' if 'analysis_unavailable' not in result else result['analysis_unavailable'], 'transfer',transfer.get('bytes',transfer.get('origin_receipt')))
(base/'analysis.json').write_text(json.dumps({'run_id':run,'rows':results,'derivation':{'decoder_source_sha256':hashlib.sha256((root/'tests/e2e/decode_capture.py').read_bytes()).hexdigest(),'analyzer_source_sha256':hashlib.sha256((root/'tests/e2e/analyze_native_capture.py').read_bytes()).hexdigest(),'label_normalization':'Derived analyzer input maps before-transfer to before-first-transfer and wraps matrix transfer as bulk; original snapshot and AT rows stay unchanged. Raw pcaps remain private.'}},indent=2)+'\n')
