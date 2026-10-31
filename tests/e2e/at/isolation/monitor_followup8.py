#!/usr/bin/env python3
"""Passive nft event receipts across node process restarts; no mutations."""
from datetime import datetime, timezone
import json
from pathlib import Path
import subprocess
import time

output = Path('docs/e2e/at/isolation/nft-passive-receipts.jsonl')
end = time.monotonic() + 2400
with output.open('w') as f:
    while time.monotonic() < end:
        running = subprocess.run(['docker','inspect','-f','{{.State.Running}}','fvpn-w4-isolation-node'], capture_output=True, text=True)
        if running.returncode:
            break
        if running.stdout.strip() != 'true':
            time.sleep(.2)
            continue
        p = subprocess.Popen(['docker','exec','fvpn-w4-isolation-node','nft','-j','monitor'], stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True)
        try:
            for line in p.stdout:
                at = dict(mono_ns=str(time.monotonic_ns()), wall=datetime.now(timezone.utc).isoformat().replace('+00:00','Z'))
                try:
                    doc = json.loads(line)
                except ValueError:
                    continue
                for entry in doc.get('nftables', [doc]):
                    for action, kinds in entry.items():
                        if not isinstance(kinds, dict):
                            continue
                        for kind, body in kinds.items():
                            if not isinstance(body, dict) or body.get('table') != 'fvpn':
                                continue
                            if kind != 'rule' and body.get('name') not in ('allowed_v4', 'fv_deadline_guard'):
                                continue
                            row = dict(receipt=at, action=action, kind=kind, name=body.get('name'), chain=body.get('chain'))
                            if str(body.get('comment','')).startswith('fvpn_generation='):
                                row['generation_comment'] = body['comment']
                            f.write(json.dumps(row)+'\n');f.flush()
        finally:
            if p.poll() is None:
                p.terminate()
            p.wait()
        time.sleep(.2)
