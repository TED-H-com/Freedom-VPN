#!/usr/bin/env python3
"""Offline timing/continuity derivation; preserves printed and AT verdicts."""
import calendar
from datetime import datetime
import json
from pathlib import Path
import re

root = Path('docs/e2e/at/isolation')
def read(name, default=None):
    p=root/name
    return json.loads(p.read_text()) if p.exists() else default
def write(name, value):
    (root/name).write_text(json.dumps(value,indent=2)+'\n')
def ns(s):
    base, _, frac=s.rstrip('Z').partition('.')
    return calendar.timegm(datetime.strptime(base,'%Y-%m-%dT%H:%M:%S').timetuple())*10**9+int(frac.ljust(9,'0'))
def delta(a,b):
    return str(ns(a)-ns(b))

run=read('AT10.json',{}).get('run_id')
receiver=read('receiver-capture.json',{}).get('transactions',[])
leases=[]
seen=set()
for r in receiver:
    l=(r.get('response') or {}).get('signed',{})
    if l.get('lease_id') and l['lease_id'] not in seen:
        seen.add(l['lease_id']);leases.append(dict(received_at=r['received_at'],response_at=r['response_at'],lease=l))
events=[]
p=root/'nft-passive-receipts.jsonl'
if p.exists():
    for line in p.read_text().splitlines():
        if line.strip():events.append(json.loads(line))
guards=[];pending=None
for e in events:
    if e.get('kind')=='chain' and e.get('name')=='fv_deadline_guard':
        if e['action']=='add':pending=e
        if e['action']=='delete' and pending:
            guards.append(dict(add_receipt=pending['receipt'],delete_receipt=e['receipt'],receipt_interval_ns=str(int(e['receipt']['mono_ns'])-int(pending['receipt']['mono_ns']))));pending=None
renewal=read('renewal-probes.json',[]);packets=read('renewal-packets.json',[])
lo=min((int(p['send']['mono_ns']) for p in renewal),default=0)
hi=max((int(p['read_completion']['mono_ns']) for p in renewal),default=0)
ingress={};egress={}
for p in packets:
    if p.get('interface')=='fvmb0':ingress.setdefault(p['packet_id'],p['capture_read'])
    if p.get('interface')=='fvup0':egress.setdefault(p['packet_id'],p['capture_read'])
losses=[]
for p in renewal:
    if p['result']=='ok':continue
    at=int((ingress.get(p['packet_id']) or p['send'])['mono_ns'])
    near=min(guards,key=lambda g:abs(at-int(g['add_receipt']['mono_ns']))) if guards else None
    losses.append(dict(probe=p,ingress=ingress.get(p['packet_id']),egress=egress.get(p['packet_id']),nearest_guard_receipts=near))
renew_wall_start=min((p['send']['wall'] for p in renewal),default='9999')
renew_wall_end=max((p['read_completion']['wall'] for p in renewal),default='0000')
write('continuity-analysis.json',dict(run_id=run,scope='Repeated-renewal availability, separate from final expiry; D-120 is not applied to this 80ms dual-phase diagnostic stream.',probes=len(renewal),echo_ok=sum(p['result']=='ok' for p in renewal),losses=losses,unobserved_ingress=[p['packet_id'] for p in renewal if p['packet_id'] not in ingress],without_egress=[p['packet_id'] for p in renewal if p['packet_id'] not in egress],capture_completion=[p for p in packets if p.get('event')=='complete'],renewals=[l for l in leases if renew_wall_start<=l['response_at']<=renew_wall_end],guard_receipt_intervals=[g for g in guards if lo<=int(g['add_receipt']['mono_ns'])<=hi],timing_limit='Passive netlink receipts include Docker delivery; not executor brackets, kernel commit instants or final-render/install samples. Initial recorder interval missing.'))
expiry=read('deadline-verdict.json',{})
whole=read('whole-node-probes.json',{})
if expiry:
    write('expiry-timing-analysis.json',dict(run_id=run,packet_verdict=expiry,lease=whole.get('lease'),last_positive_margin_ns=str(int(expiry['last_positive_offset_ns'])+101000000),first_drop_margin_ns=str(100000000-int(expiry['first_drop_offset_ns'])),precision='1ms quantum; 100ms install bound; no extra cadence allowance',missing_producer_samples=['Apply start','final render start','installed-by','executor precision-read start/end'],available='signed envelopes, node SQLite anchor, monotonic sends/ingress/egress/read completion, periodic precision-read brackets, heartbeat, renewal wire, watchdog log and passive guard receipts'))
trace=read('foreign-redirect.json',{})
rows=trace.get('rows',[])
def first(label):return next((r for r in rows if r['label']==label),{})
injected=first('inject clsact mirred');removed=first('remove all injected objects')
contained=next((r for r in rows if r['label']=='containment wait' and 'serving 0' in r.get('readback','') and any('TC_FOREIGN_FILTER'==e.get('code') for e in r.get('observation',{}).get('errors',[]))),{})
serving=next((r for r in rows if r['label']=='recovery wait' and 'serving 1' in r.get('readback','') and r.get('observation',{}).get('member_forwarding_enabled')), {})
positive=first('final counted control')
rejected=[];unsolicited=[]
for r in receiver:
    b=r.get('body',{});t=b.get('tc_evidence') or {}
    if t.get('error_code')=='TC_FOREIGN_FILTER':
        item=dict(report_id=b['report_id'],read_at=t['read_at'],candidate_generation=t['candidate_generation'],rejected=t.get('rejected'),attachments=t.get('attachments'))
        rejected.append(item)
        if t['read_at'] < injected.get('start','9999') or t['read_at'] > removed.get('end','0000'):
            # Retained historical reads are classified by their own read time,
            # never by the later observation emission time.
            unsolicited.append(item)
write('foreign-redirect-analysis.json',dict(run_id=run,assisted=trace.get('assisted'),restoration_owner=trace.get('restoration_owner'),injection=injected,removal=removed,first_contained_read=contained,first_serving_read=serving,final_counted=positive,injection_to_contained_upper_ns=delta(contained['read_end'],injected['start']) if contained and injected else None,removal_to_serving_observed_upper_ns=delta(serving['read_end'],removed['end']) if serving and removed else None,removal_to_counted_upper_ns=delta(positive['at'],removed['end']) if positive and removed else None,rejected_reads=rejected,unsolicited_rejected_reads=unsolicited,creator='The harness created the labelled clsact/flower/mirred injection. A rejected row alone does not encode a process creator; any unsolicited rows remain unattributed.'))
# Complete node-any capture and containment window, including pre-containment bypass.
analysis=read('foreign-redirect-analysis.json',{})
text=(root/'redirect-node-capture.txt').read_text()
headers=[line for line in text.splitlines() if re.match(r'^\d+\.\d+ ',line)]
reported=re.search(r'(?m)^(\d+) packets? captured$',text)
def packet_ns(line):
    sec,frac=line.split()[0].split('.')
    return int(sec)*10**9+int(frac.ljust(9,'0'))
if injected and contained and removed:
    uplink=[line for line in headers if 'fvup0 Out ' in line and '.9: UDP' in line]
    before=[line for line in uplink if ns(injected['start'])<=packet_ns(line)<ns(contained['read_end'])]
    during=[line for line in uplink if ns(contained['read_end'])<=packet_ns(line)<ns(removed['end'])]
    analysis['packet_evidence']=dict(parsed=len(headers),reported=int(reported[1]) if reported else None,complete=bool(reported and int(reported[1])==len(headers) and '0 packets dropped by kernel' in text),uplink_between_injection_start_and_confirmed_down=before,uplink_after_confirmed_down_until_removal=during)
    raised=next((r for r in rows if r['label']=='recovery wait' and any(link.get('ifname')=='fvmb0' and 'UP' in link.get('flags',[]) for link in json.loads(r['readback'].splitlines()[0]))),None)
    if raised:
        analysis['first_link_up_read']=raised
        analysis['removal_to_link_up_upper_ns']=delta(raised['read_end'],removed['end'])
write('foreign-redirect-analysis.json',analysis)
