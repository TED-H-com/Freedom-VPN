"""Publish safe receipts for original failed withdrawal reads; never re-read nft."""
from pathlib import Path
import json,collections,hashlib,uuid
root=Path('docs/e2e/at/members')
case=json.loads((root/'AT04.json').read_text())
failures=[];samples={}
for step in case['steps']:
 if not step['name'].startswith('persistent withdrawal '):continue
 value=json.loads(step['actual']);ticks=value['samples'];bad=[]
 for i,tick in enumerate(ticks):
  p=tick['counters']['read_diagnostic'].get('parse_failure')
  if p:
   samples[p['id']]=p
   bad.append({'sample_index':i,'sample':tick,'previous':ticks[i-1] if i else None,'next':ticks[i+1] if i+1<len(ticks) else None})
 if bad:
  failures.append({'original_check':{k:v for k,v in step.items() if k!='actual'},'reported_result':{k:v for k,v in value.items() if k!='samples'},'failed_samples_and_neighbors':bad})
rows=[]
for sample_id,metadata in samples.items():
 assert str(uuid.UUID(sample_id))==sample_id
 p=Path('.agent/tmp/counterread')/sample_id/'output.raw'
 row={'metadata':metadata,'private_verified':False}
 try:
  raw=p.read_bytes();error=p.with_name('parser-error.txt').read_bytes()
  row.update(raw_sha256=hashlib.sha256(raw).hexdigest(),raw_bytes=str(len(raw)),raw_lines=len(raw.splitlines()),exact_parser_error_sha256=hashlib.sha256(error).hexdigest(),exact_parser_error_bytes=str(len(error)),private_files_mode_0600=all((f.stat().st_mode&0o777)==0o600 for f in [p,p.with_name('parser-error.txt')]))
  row['private_verified']=row['raw_sha256']==metadata['sha256'] and row['raw_bytes']==metadata['bytes'] and row['private_files_mode_0600']
  try:d=json.loads(raw);row['json_valid']=True
  except ValueError:d={};row['json_valid']=False
  chains=collections.defaultdict(list)
  for i,o in enumerate(d.get('nftables',[])):
   if 'chain' in o:
    c=o['chain'];chains[(c.get('family'),c.get('table'),c.get('name'))].append((i,c))
  duplicates=[]
  for key,cs in chains.items():
   if len(cs)<2:continue
   duplicates.append({'object_indices_zero_based':[i for i,c in cs],'family':key[0],'owned_table':key[1]=='fvpn','chain_class':key[2] if key[2] in ['input','output','forward','prerouting','postrouting'] else 'other_fixture_chain','identical':all(c==cs[0][1] for i,c in cs)})
  row.update(duplicate_chain_identities=duplicates,safe_parser_reason='duplicate chain identity' if error.startswith(b'duplicate chain ') else 'unclassified')
 except OSError:row['private_error']='retained private files unavailable'
 row.update(parser_site=('node/enforce/parse_nft.go:400 noteChain rejects a second (family, table, name), including identical definitions' if row.get('safe_parser_reason')=='duplicate chain identity' else 'unattributed'),parser_defect_status='candidate compatibility issue; duplicate-output origin and permitted snapshot contract unproven',no_retry=True,raw_and_exact_error='private .agent/tmp/counterread/<id>/output.raw and parser-error.txt; not published')
 rows.append(row)
producer_path=Path('.agent/tmp/f14-parser-producer.jsonl')
producer_export=None;windows=[]
if producer_path.exists():
 raw=producer_path.read_bytes();records=[];complete=True
 for line in raw.splitlines():
  try:records.append(json.loads(line))
  except ValueError:complete=False
 producer_export={'complete':complete,'bytes':str(len(raw)),'sha256':hashlib.sha256(raw).hexdigest(),'records':len(records),'scope':'read-only stream export after original sample; private full JSONL retained'}
 for withdrawal in failures:
  for bad in withdrawal['failed_samples_and_neighbors']:
   d=bad['sample']['counters']['read_diagnostic'];before=d['enforcement_before']['record'];seq=int(before['sequence'])
   selected=[r for r in records if r['run_id']==before['run_id'] and seq-2<=int(r['sequence'])<=seq+8]
   overlap=[r['sequence'] for r in selected if 'timing' in r and max(int(d['read_start_mono_ns']),int(r['timing']['start_ns']))<=min(int(d['read_end_mono_ns']),int(r['timing']['end_ns']))]
   windows.append({'parse_failure_id':d['parse_failure']['id'],'producer_records':selected,'overlapping_operation_sequences':overlap,'cause_status':'unproven; overlap is not cause'})
value={'run_id':case['run_id'],'source_commit':case['source_commit'],'samples':rows,'withdrawals':failures,'producer_export':producer_export,'producer_windows':windows,'historical_F13_01':'both original failed rows remain failed and unattributed; their raw bytes were never retained'}
(root/'F14-F13-01-private-receipts.json').write_text(json.dumps(value,indent=2)+'\n')
print('F13-01 safe receipts:',len(rows),'verified:',sum(r['private_verified'] for r in rows))
