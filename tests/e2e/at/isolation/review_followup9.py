#!/usr/bin/env python3
"""Offline follow-up 9 review; preserves printed AT and D-120 verdicts."""
import json
from pathlib import Path
import runpy
import sys
from producer_timing import analyze

root=Path(sys.argv[1] if len(sys.argv)>1 else 'docs/e2e/at/isolation')
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

# Focused failure index, derived from this run. A baseline's presence does not
# identify its writer; exact engine timing does not cover watchdog ApplyBaseline.
verdict=json.loads((root/'deadline-verdict.json').read_text())
positives=[p for p in verdict['packets'] if int(p['egress_mono_ns'])>0]
first_positive=min((int(p['ingress_mono_ns']) for p in positives),default=0)
losses=r['expiry']['losses']; deadline=int(r['expiry']['deadline_ns'])
early=[p for p in losses if first_positive and int(p['packet']['ingress_ns'])<first_positive]
final=[p for p in losses if int(p['packet']['ingress_ns'])>int(verdict['last_positive_mono_ns'])]
producer_run=r['expiry']['last_accepted_renewal']['run_id']
rows=[json.loads(s) for s in (root/'producer-operations.jsonl').read_text().splitlines()]
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
    premise='A generation-0 baseline read is distinct from a timed guard. No baseline-writer timestamp/identity is exported; watchdog involvement remains an inference.',
    deadline_ns=str(deadline),producer_run_id=producer_run,
    early_dropout=dict(count=len(early),packet_ids=[p['packet']['packet_id'] for p in early],
        first_ingress_offset_ns=str(int(early[0]['packet']['ingress_ns'])-deadline) if early else None,
        first_subsequent_positive_offset_ns=str(first_positive-deadline) if first_positive else None,
        ingress_inside_guard=sum(bool(p['ingress_in_outer_guard']) for p in early),outside_guard=sum(not p['ingress_in_outer_guard'] for p in early)),
    final_expiry_drops=dict(count=len(final),first_ingress_offset_ns=str(int(final[0]['packet']['ingress_ns'])-deadline) if final else None,
        post_deadline_positives=sum(int(p['egress_mono_ns'])>=deadline for p in positives)),
    exact_producer_sequence=selected,wire_reports=list(reports.values()),
    first_baseline_read=baselines[0] if baselines else None,last_baseline_read=baselines[-1] if baselines else None,
    missing_writer_evidence='Engine Apply/Verify timing does not timestamp external ApplyBaseline invocations, including the independent watchdog. Its retained log is separate.',
    all_loss_windows='producer-timing-analysis.json and expiry-timing-analysis.json retain every loss, with command and verifier references.'))
