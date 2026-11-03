"""Retained packet joins, preserving both successful and failed original attempts."""
import sys,json
from pathlib import Path
sys.path.insert(0,'tests/e2e/at/members')
from evidence_io import load,publish,read
root=Path('docs/e2e/at/members');case=load(root/'AT01.json');run=case['run_id'];base=root/'history/follow-up-17-captures'/run
rows=[]
for directory in sorted((base/'headers').iterdir()):
 packets={}
 for role in ['member','node']:
  p=directory/(role+'.headers.jsonl')
  if p.exists() or Path(str(p)+'.gz').exists():packets[role]=[json.loads(line) for line in read(p).splitlines()]
 cap=load(base/(directory.name+'-original-capture.json'))
 messages=sorted({p['wireguard']['message_sha256'] for pp in packets.values() for p in pp if p.get('ip_version')==6 and p.get('wireguard',{}).get('source_loopback')})
 for digest in messages:
  hops=[dict(role=role,packet=p) for role,pp in packets.items() for p in pp if p.get('wireguard',{}).get('message_sha256')==digest]
  samples=[dict(label=s['label'],points={r:dict(start=p['start'],end=p['end'],latest_handshake_unix_seconds=p.get('latest_handshake_unix_seconds'),wireguard_endpoint=p.get('wireguard_endpoint'),routes=p.get('routes'),ipv6_counters=p.get('kernel',{}).get('snmp6',{})) for r,p in s['points'].items() if r in packets}) for s in cap['samples']]
  rows.append(dict(capture=directory.name,capture_id=cap['id'],message_sha256=digest,matching_hops=hops,independent_samples=samples,cause_status='unproven'))
first=load(base/'initial-selection-analysis.json');syns=first.get('member_syns',[]);syn=syns[0] if syns else None
arrival=next((p for p in first.get('hops',{}).get('node',[]) if syn and p['interface']=='fvmb0' and p['packet_type']!=4 and p.get('tcp')==syn['tcp']),None)
value=dict(run_id=run,source_commit=case['source_commit'],cause_status='unproven',ipv6_loopback_messages=rows,original_first_forwarding=case.get('forwarding',[None])[0],member_first_syn=syn,node_matching_first_syn=arrival,transit_lower_ns=str(int(arrival['mono_lower_ns'])-int(syn['mono_upper_ns'])) if arrival else None,transit_upper_ns=str(int(arrival['mono_upper_ns'])-int(syn['mono_lower_ns'])) if arrival else None,scope='Headers and opaque hashes join observations; they do not establish a kernel discard site or causal attribution. A passing first selection closes neither F11-01 nor F14-01. All historical failures remain unchanged.')
p=root/'F17-loopback-and-first-selection.json';p.write_text(json.dumps(value,indent=2)+'\n');publish(p)
print('first selection counted',value['original_first_forwarding']['counted'],'transit ns',value['transit_lower_ns'],value['transit_upper_ns'],'loopback messages',len(rows))
