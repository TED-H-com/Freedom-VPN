"""Persist a current-run block before opening any possibly inherited evidence."""
import json
from evidence_io import read_json


def unexecuted_expiry(root, output, current=None):
    current = current if current is not None else read_json(root/'AT13.json')
    expiry = next((c for c in current.get('checks', []) if c['name']=='short whole-node lease expiry'), {})
    if 'D-104 exact packet deadline verdict=' in expiry.get('actual', ''):
        return None
    result = dict(run_id=current['run_id'], status='blocked', reason='expiry not executed',
                  owner=expiry.get('owner', 'acceptance'), check=expiry,
                  inherited_evidence_used=False, schedule_qualified=False,
                  scope='Current attempt only; no packet, schedule or producer evidence read.')
    (root/output).write_text(json.dumps(result, indent=2)+'\n')
    return result
