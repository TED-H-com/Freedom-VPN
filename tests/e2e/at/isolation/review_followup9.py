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
