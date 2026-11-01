#!/usr/bin/env python3
"""Offline follow-up 10 review; preserves printed AT and D-120 verdicts."""
import json
from pathlib import Path
import runpy
import sys
from producer_timing import analyze, read_operations

root=Path(sys.argv[1] if len(sys.argv)>1 else 'docs/e2e/at/isolation')
# Refuse inherited packet files when this attempt did not execute expiry.
current=json.loads((root/'AT13.json').read_text())
expiry=next((c for c in current.get('checks',[]) if c['name']=='short whole-node lease expiry'),{})
if 'D-104 exact packet deadline verdict=' not in expiry.get('actual',''):
    from producer_timing import validate_operations
    rows=read_operations(root)
    v=validate_operations(rows)
    incomplete=dict(run_id=current['run_id'],status='unavailable',reason='This attempt did not execute the whole-node packet window; inherited packet files are not evidence for it.',
        check=expiry,source_counts=v['source_counts'],snapshots=v['snapshots'],brackets=v['brackets'],failed_applies=v['failed_applies'])
    (root/'producer-timing-unavailable.json').write_text(json.dumps(incomplete,indent=2)+'\n')
    raise SystemExit('Current expiry measurement unavailable; refusing inherited packet evidence')

sys.argv=[sys.argv[0],str(root)]
runpy.run_path(str(Path(__file__).with_name('summarize_followup8.py')),run_name='__main__')
r=analyze(root)
def write(name,value):
    (root/name).write_text(json.dumps(value,indent=2)+'\n')
write('continuity-analysis.json',dict(run_id=r['run_id'],scope=r['scope'],historical=r['historical'],
    **r['renewal'],producer_timing='producer-timing-analysis.json',timing_limit=r['interpretation']))
old=json.loads((root/'expiry-timing-analysis.json').read_text())
old.update(missing_producer_samples=[],available='Exact producer snapshots plus signed envelopes, node SQLite anchor, same-boot/time-namespace packet clocks and periodic precision/heartbeat reads',
           producer_timing='producer-timing-analysis.json',producer_brackets=r['brackets'],
           last_accepted_renewal=r['expiry']['last_accepted_renewal'],loss_correlation=r['expiry']['losses'])
write('expiry-timing-analysis.json',old)

# Focused failure index, derived only from this run. Baseline source rows name
# agent writes; the separate watchdog log retains its own decisions and writes.
verdict=json.loads((root/'deadline-verdict.json').read_text())
positives=[p for p in verdict['packets'] if int(p['egress_mono_ns'])>0]
first_positive=min((int(p['ingress_mono_ns']) for p in positives),default=0)
losses=r['expiry']['losses']; deadline=int(r['expiry']['deadline_ns'])
early=[p for p in losses if first_positive and int(p['packet']['ingress_ns'])<first_positive]
final=[p for p in losses if int(p['packet']['ingress_ns'])>int(verdict['last_positive_mono_ns'])]
producer_run=r['expiry']['last_accepted_renewal']['run_id']
rows=read_operations(root)
process=[p for p in rows if p['run_id']==producer_run]
selected=[]
if early:
    start=int(early[0]['packet']['ingress_ns'])
    applies=[(i,p) for i,p in enumerate(process) if p.get('timing',{}).get('operation')=='apply' and p['timing']['succeeded']]
    before=[i for i,p in applies if int(p['timing']['end_ns'])<start]
    after=[i for i,p in applies if int(p['timing']['start_ns'])>=start]
    if before and after:
        selected=process[max(before[-1]-2,0):after[0]+3]
wire=json.loads((root/'receiver-capture.json').read_text())['transactions']
reports={}
if selected:
    begin,end=selected[0]['anchor']['wall'],selected[-1]['anchor']['wall']
    for w in wire:
        b=w.get('body') or {}
        if b.get('report_id') and begin<=b.get('observed_at','')<=end:
            item=reports.setdefault(b['report_id'],dict(report=b,attempts=[]))
            item['attempts'].append({k:w[k] for k in ('received_at','response_at','status')})
reads=json.loads((root/'whole-node-continuous-reads.json').read_text())['reads']
baselines=[p for p in reads if 'fvpn_generation=0 fvpn_baseline=1' in p['readback']]
write('expiry-failure-correlation.json',dict(run_id=r['run_id'],status='pass' if verdict['passed'] else 'fail',
    owner='w2-agent-core: node/agent/engine.go Step/finishServingStep/verifyCurrentServing and node/watchdog/watch.go Watch.needsBaseline; netlab supports verifier diagnostics' if early else None,
    premise='Fresh agent baseline/heartbeat rows and watchdog reason logs are retained. A generation-0 read alone does not name a writer; historical follow-up 9 remains unattributed.',
    producer_sources=r['source_counts'],
    baseline_rows=[p for p in rows if p['source']=='baseline'],
    watchdog_reason_log='watchdog.log.txt',agent_reason_log='agent.log.txt',
    deadline_ns=str(deadline),producer_run_id=producer_run,
    early_dropout=dict(count=len(early),packet_ids=[p['packet']['packet_id'] for p in early],
        first_ingress_offset_ns=str(int(early[0]['packet']['ingress_ns'])-deadline) if early else None,
        first_subsequent_positive_offset_ns=str(first_positive-deadline) if first_positive else None,
        ingress_inside_guard=sum(bool(p['ingress_in_outer_guard']) for p in early),outside_guard=sum(not p['ingress_in_outer_guard'] for p in early)),
    final_expiry_drops=dict(count=len(final),first_ingress_offset_ns=str(int(final[0]['packet']['ingress_ns'])-deadline) if final else None,
        post_deadline_positives=sum(int(p['egress_mono_ns'])>=deadline for p in positives)),
    exact_producer_sequence=selected,wire_reports=list(reports.values()),
    first_baseline_read=baselines[0] if baselines else None,last_baseline_read=baselines[-1] if baselines else None,
    writer_evidence='Actual agent baseline calls are bracketed in baseline_rows; watchdog decisions/writes are in the retained reason log. No historical attribution is inferred.',
    all_loss_windows='producer-timing-analysis.json and expiry-timing-analysis.json retain every loss, with command and verifier references.'))

# Independently recheck original D-087 wire bytes and acceptance ordering.
from grant_evidence import validate as validate_grants
try:
    grant_audit=validate_grants(json.loads((root/'member-grants-transition.json').read_text()),
                               json.loads((root/'member-grants-wire.json').read_text()))
except (AssertionError,KeyError,ValueError,IndexError) as error:
    write('member-grants-audit.json',dict(run_id=r['run_id'],status='fail',error_kind=type(error).__name__,
        reason='The captured transition does not qualify. Retain its AT10 failure, invariant snapshots and original wire; incidental reports cannot supply a fixed-generation transition.'))
    raise SystemExit('D-087 captured transition failed independent replay')
write('member-grants-audit.json',grant_audit)

# Join original node acceptance records to both genuine response/report pairs.
transition=json.loads((root/'member-grants-transition.json').read_text())
joined={}
for phase,lease_key,report_key,flip_key in [
    ('empty','empty_lease','provisional_receipt','false_flip'),
    ('populated','populated_lease','serving_receipt','true_flip')]:
    event=transition[lease_key]
    lease_id=event['response']['signed']['lease_id']
    accepted=[p for p in rows if p.get('renewal',{}).get('new_lease') and p.get('lease',{}).get('lease_id')==lease_id]
    if len(accepted)!=1:
        raise ValueError('missing or duplicate actual node lease acceptance: '+phase)
    joined[phase]=dict(flip=transition[flip_key],server_renewal=dict(started=event['started'],completed=event['completed'],response_sha256=event['response_sha256']),
        node_acceptance=accepted[0],accepted_report=dict(report_id=transition[report_key]['request']['report_id'],started=transition[report_key]['started'],completed=transition[report_key]['completed']))
write('member-grants-timing.json',dict(run_id=r['run_id'],phases=joined,counted_forwarding=transition['counted_forwarding'],
    timing_basis='Server completion, original detached node renewal completion and actual accepted observation receipts are separate samples. Counted native packet timestamps are in the original captures; no scheduled slot is substituted.'))
