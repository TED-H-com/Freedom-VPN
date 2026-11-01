"""Derive public, payload-free evidence from this lane's original captures."""
from pathlib import Path
import sys,json,subprocess,hashlib,copy
root=Path.cwd()
sys.dont_write_bytecode=True
sys.path.insert(0,str(root/'tests/e2e'))
import analyze_native_capture as native
import analyze_router_capture as router
import analyze_dns_capture as dns
cases={p.stem:json.loads(p.read_text()) for p in (root/'docs/e2e/at/members').glob('AT??.json')}
run=cases['AT01']['run_id'];source=cases['AT01']['source_commit']
base=root/'docs/e2e/at/members/history/follow-up-14-captures'/run
base.mkdir(parents=True,exist_ok=True)
private=root/'.agent/tmp/f14-native'/run
entries=[]
for i,o in enumerate(cases['AT01'].get('forwarding',[])):
 if o.get('native_capture'):entries.append(('initial-selection' if i==0 else 'later-selection-'+str(i),o['native_capture'],{'transfer':o['transfer']},'pass' if o['counted'] else 'blocked'))
names={'native ipv4 MTU 1500 transfer':'matrix4','native ipv6 MTU 1500 transfer':'matrix6','native dual MTU 1500 transfer':'matrixDual','native dual MTU 1280 transfer':'matrix1280','native ipv6 MTU 1280 transfer':'ipv6-asymmetric','D-137 supported known-PMTU 1280 transfer':'ipv6-explicit1280','D-137 routed ipv6 1280 transfer':'router-ipv6','D-137 routed ipv4 1280 transfer':'router-ipv4'}
for step in (cases['AT09']['steps'] if cases['AT09']['run_id']==run else []):
 if step['name'] in names:
  v=json.loads(step['actual'])
  entries.append((names[step['name']],v['native_capture'],{'transfer':dict(v,bulk=v['transfer'])},step['status']))
original=json.loads((root/'docs/e2e/at/members/F14-F12-04-original-dns.json').read_text())
queries={'rr':original['original_query']}
for s in cases['AT20']['steps']:
 if s['name'].startswith('positive IDNA TCP='):queries['idna-tcp' if s['name'].endswith('true') else 'idna-udp']=json.loads(s['actual'])
if original.get('native_capture'):entries.append(('native-dns-delivery',original['native_capture'],queries,'diagnostic'))
results=[]
for name,capture,notes,status in entries:
 if not capture:
  results.append({'id':name,'analysis_unavailable':'capture absent'});continue
 (base/(name+'-original-capture.json')).write_text(json.dumps(capture,indent=2)+'\n')
 try:
  directory=next(p.parent for p in private.glob('*/capture.json') if json.loads(p.read_text())['id']==capture['id'])
  roles=['member','node','origin']+[r for r in sorted(capture['points']) if r not in ('member','node','origin')]
  for role in roles:
   raw=directory/(role+'.pcap');receipt=capture['points'][role]['private_pcap_receipt']
   assert raw.stat().st_size==int(receipt['bytes']) and hashlib.sha256(raw.read_bytes()).hexdigest()==receipt['sha256']
 except (StopIteration,FileNotFoundError,KeyError,AssertionError):
  result={'id':name,'capture_id':capture['id'],'analysis_unavailable':'private capture missing or receipt mismatch','original_capture':capture}
  results.append(result)
  (base/(name+'-analysis.json')).write_text(json.dumps(result,indent=2)+'\n')
  continue
 header=base/'headers'/name
 decoded=json.loads(subprocess.check_output(['python3','-B','tests/e2e/decode_capture.py','--stable-addresses','--wireguard-headers','--dns-headers','--out',str(header)]+[str(directory/(r+'.pcap')) for r in roles],text=True))
 for d in decoded:d['file']=Path(d['file']).name
 (base/(name+'-original-capture.json')).write_text(json.dumps(capture,indent=2)+'\n')
 derived=copy.deepcopy(capture)
 for sample in derived['samples']:
  if sample['label']=='before-transfer':sample['label']='before-first-transfer'
 notes=dict(notes,capture=derived,decode=decoded)
 if name not in ('initial-selection','native-dns-delivery') and not name.startswith('later-'):
  # Only the owner analyzer adapter; original AT results remain untouched.
  transfer=notes['transfer'];transfer.pop('native_capture',None)
 doc={'id':name,'run_id':run,'source_commit':source,'status':status,'observations':[{'name':k,'detail':json.dumps(v)} for k,v in notes.items()]}
 rec=base/(name+'.json');rec.write_text(json.dumps(doc,indent=2)+'\n')
 try:
  if name.startswith('router-'):
   family=name.split('-')[1];router.analyze(base,family)
   result=json.loads(json.loads((base/('router-analysis-'+family+'.json')).read_text())['observations'][0]['detail']);result['id']=name
  elif name=='native-dns-delivery':
   dns.analyze(base)
   result=json.loads(json.loads((base/'dns-delivery-analysis.json').read_text())['observations'][0]['detail']);result['id']=name
  elif name.startswith('later-'):
   result={'id':name,'scope':'Original later sleep/wake transfer; no first-selection attribution','capture_checks':{},'transfer':notes['transfer']}
   for role,check in zip(roles,decoded):
    p=capture['points'][role]
    complete=p['statistics_ok'] and p['process_stopped'] and p['kernel_drops']==0 and p['captured']==check['records'] and check['header_errors']==check['snap_truncated_records']==0 and check['file_complete']
    result['capture_checks'][role]={'complete':complete,'records':check['records'],'kernel_drops':p['kernel_drops']}
    assert complete,role+' capture incomplete'
  else:result=native.analyze(base,rec)
 except (AssertionError,KeyError,StopIteration,ValueError) as e:
  result={'id':name,'analysis_unavailable':str(e),'capture_id':capture['id'],'decode':decoded}
 (base/(name+'-analysis.json')).write_text(json.dumps(result,indent=2)+'\n')
 results.append(result)
 print(name,'complete' if 'analysis_unavailable' not in result else result['analysis_unavailable'])
provenance={name:hashlib.sha256((root/'tests/e2e'/name).read_bytes()).hexdigest() for name in ['decode_capture.py','analyze_native_capture.py','analyze_router_capture.py','analyze_dns_capture.py']}
(base/'analysis.json').write_text(json.dumps({'run_id':run,'source_commit':source,'rows':results,'derivation':{'source_sha256':provenance,'adapter':'Original captures retained unchanged; derived input renames before-transfer to before-first-transfer and wraps bulk; header-only decode; raw pcaps remain private.'}},indent=2)+'\n')
files=sorted(p for p in base.rglob('*') if p.is_file() and p.name!='SHA256SUMS')
(base/'SHA256SUMS').write_text(''.join(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+str(p.relative_to(base))+'\n' for p in files))
