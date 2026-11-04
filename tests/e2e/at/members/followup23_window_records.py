"""Index original follow-up 23 path/window evidence without replay or predicate changes."""
from pathlib import Path
import json,sys,hashlib,datetime
from collections import defaultdict, deque
from decimal import Decimal
sys.path.insert(0,'tests/e2e/at/members')
from evidence_io import load,publish
sys.path.insert(0,'tests/e2e')
import analyze_native_capture as native
R=Path('docs/e2e/at/members')
run='6a25221b-1dcf-4f4d-b866-cd04f57f58f6';source='8bde2b6acc0b116515148eeb0ff4448c65453a92'
def save(name,v):
 p=R/name;p.write_text(json.dumps(dict(run_id=run,source_commit=source,**v),indent=2)+'\n');publish(p)
def dt(s):return datetime.datetime.fromisoformat(s.replace('Z','+00:00'))
def stamp(s):return Decimal(native.wall_ns(s))/Decimal(10**9)
forward=[]
for name in ['AT01','AT08']:
 c=load(R/(name+'.json'));assert c['run_id']==run and c['complete']
 for i,f in enumerate(c.get('forwarding',[])):
  if not f.get('native_capture'):continue
  cap=f['native_capture'];rows=[];after=f['peer_status_after']['underlay']
  for role in ['member','node']:
   selected=after[role]['selected_remote']['address']
   views={}
   for phase,field in [('before','peer_status'),('after','peer_status_after')]:
    p=f[field]['underlay'][role]
    views[phase]=dict(started_at=p['started_at'],ended_at=p['ended_at'],selected_local=p['selected_local'],selected_remote=p['selected_remote'],route_to_eventually_selected_endpoint=[r for r in p['paths'] if r['destination_label']==selected],wireguard_command_ok=p['wireguard_command_ok'],wireguard=p['wireguard'])
   rows.append(dict(role=role,**views))
  pkt=f.get('underlay_capture',{})
  before=next((s for s in cap['samples'] if s['label']=='before-transfer'),None)
  aft=next((s for s in cap['samples'] if s['label']=='after-transfer'),None)
  start=max(p['end']['wall'] for p in before['points'].values()) if before else None
  end=min(p['start']['wall'] for p in aft['points'].values()) if aft else None
  window={}
  for role,p in pkt.get('points',{}).items():
   window[role]=[p for p in p['packets'] if start and end and stamp(start)<=Decimal(p['timestamp'])<=stamp(end)]
  directions={}
  for sender,receiver in [('member','node'),('node','member')]:
   tx=[p for p in window[sender] if p['direction']=='egress']
   rx=[p for p in window[receiver] if p['direction']=='ingress']
   queues=defaultdict(deque)
   for p in rx:
    if p.get('ip_sha256'):queues[p['ip_sha256']].append(p)
   joined_packets=[]
   for p in tx:
    q=queues[p.get('ip_sha256')].popleft() if queues[p.get('ip_sha256')] else None
    joined_packets.append(dict(sender_packet=p,receiver_packet=q))
   directions[sender+'_to_'+receiver]=dict(sender_packets=tx,receiver_packets=rx,exact_ip_joins=joined_packets,joined_occurrences=sum(x['receiver_packet'] is not None for x in joined_packets),scope='All filtered underlay traffic in the transfer window, including candidate discovery and ICMP. An unmatched unrelated egress is not a lost selected-peer packet.')
  loopback='addr-'+hashlib.sha256(b'::1').hexdigest()[:16]
  responses=[p for p in window.get('node',[]) if p['direction']=='egress' and p['src']==loopback and p.get('wireguard',{}).get('type') in [1,2,4]]
  joined=[dict(node_packet=p,member_packets=[q for q in window.get('member',[]) if q['direction']=='ingress' and q.get('ip_sha256')==p.get('ip_sha256')]) for p in responses]
  missing='none: original origin receipt and both device counter increments' if f['counted'] else 'unresolved; inspect original end-to-end packet and counter records'
  if not f['counted'] and any(j['member_packets'] for j in joined) and after['member']['wireguard']['received_bytes']=='0':missing='after member underlay ingress, before WireGuard receive accounting; loopback-source IP hash reaches member, but no per-packet kernel drop trace'
  forward.append(dict(case=name,forwarding_index=i,initial=i==0,counted=f['counted'],origin_requests=f['origin_requests'],origin_snat_matches=f['origin_snat_matches'],node_before=f['node_before'],node_after=f['node_after'],peer_views=rows,transfer_window=dict(started_after=start,ended_before=end,basis='between the existing before-transfer readback completion and after-transfer readback start; command scheduling lies inside'),underlay=pkt,window_packets=window,window_directions=directions,loopback_response_joins=joined,first_unconfirmed_hop=missing))
save('F23-initial-transfers.json',dict(transfers=forward,scope='Readback metadata and original passive captures only. Handshake, successful API calls and diagnostic fields do not substitute for the original origin/counter predicate.'))
c=load(R/'AT09.json');assert c['run_id']==run and c['complete'];windows=[];failed=[]
for i,s in enumerate(c['steps']):
 try:a=json.loads(s['actual'])
 except (TypeError,ValueError):continue
 if 'observation_window' in a:
  w=a['observation_window'];logs=w['agent_log'];first=logs['first_retained_log_time'];last=logs['last_retained_log_time']
  logs['row_duration_seconds']=str(Decimal(str((dt(w['ended_at'])-dt(w['started_at'])).total_seconds())))
  if first and logs['overwrite_or_replacement_detected']:
   logs['retained_suffix_from']=max(w['started_at'],first);logs['retained_suffix_seconds']=str(max(0,(dt(w['ended_at'])-max(dt(first),dt(w['started_at']))).total_seconds()))
  windows.append(dict(index=i,check=s['name'],status=s['status'],**w))
 if s['name'].endswith(' D-146 clamp evidence'):
  rules=a.get('rules',{})
  if not rules.get('serving_read_ok'):failed.append(dict(index=i,check=s['name'],status=s['status'],rules={k:v for k,v in rules.items() if k!='pairs'}))
save('F23-observation-windows.json',dict(rows=windows,failed_serving_reads=failed,scope='All control-stored node observations in each recorded half-open row window; only received_at, report ID, generation and incarnation. Agent logs lack per-submission outcomes, explicitly unavailable; no freshness or D-151 policy change.'))
print('initial captures',len(forward),'native windows',len(windows),'failed serving reads',len(failed))
