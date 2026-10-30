#!/usr/bin/env python3
"""Fresh native admission subset using the existing dev-stack hooks, not capacity."""
import json
import os
import tempfile
from run import ROOT, PRIVATE, PUBLIC, PREFIX, PORT_BASE, command, inventory, summarize_go, utc, write_json


def main():
    PRIVATE.mkdir(parents=True, exist_ok=True)
    (ROOT / '.agent/tmp').mkdir(parents=True, exist_ok=True)
    started = utc()
    record = {'started_at': started, 'scope': '20 native users × 3 peers; no serving node or capacity claim', 'node_incarnation': None}
    with tempfile.TemporaryDirectory(prefix='secops-scale.', dir=ROOT / '.agent/tmp') as scratch:
        env = dict(os.environ, GOTOOLCHAIN='local', CGO_ENABLED='0', TMPDIR=scratch, GOTMPDIR=scratch,
                   E2E_RESET='1', FVPN_E2E_PREFIX=PREFIX, FVPN_E2E_PORT_BASE=PORT_BASE,
                   FVPN_E2E_STATE=str(ROOT / '.agent/e2e-state-w4-secops'))
        before = inventory(env)
        record['inventory_before'] = before
        if any(x['exit_code'] or x['count'] for x in before.values()):
            raise SystemExit('assigned namespace occupied; refusing to attach or tear it down')
        try:
            up, _ = command('scale-e2e-up', ['make', '-f', 'mk/e2e.mk', 'e2e-up'], env, 1200)
            record['up'] = up
            if up['exit_code'] == 0:
                args = ['go', 'test', '-json', '-tags', 'e2e,integration', '-p', '1', '-parallel', '2', '-count=1', '-timeout', '45m', '-run', '^TestSECOPS10_NativeScale$', './tests/e2e/at/secops']
                result, raw = command('scale-probe', args, dict(env, FVPN_SECOPS_SCALE_ONLY='1'), 2760)
                record['probe'] = dict(result, **summarize_go(raw, ['TestSECOPS10_NativeScale'], result['exit_code']))
        finally:
            down, _ = command('scale-e2e-down', ['make', '-f', 'mk/e2e.mk', 'e2e-down'], env, 180)
            record['down'] = down
            record['inventory_after'] = inventory(env)
    record['finished_at'] = utc()
    path = PUBLIC / 'native-scale-live.json'
    live = json.loads(path.read_text()) if path.exists() else {}
    record['status'] = 'passed' if record.get('probe', {}).get('status') == 'passed' and live.get('native_scale', {}).get('status') == 'passed' else 'blocked'
    record['actual'] = live.get('native_scale')
    record['source_revision'] = live.get('source_revision')
    record['probe_source_sha256'] = live.get('probe_source_sha256')
    record['policy_revision'] = live.get('policy_revision')
    record['image_digests'] = live.get('image_digests', [])
    record['build_evidence'] = 'Final cross-compile hashes in validation.json; this subset makes no forwarding assertion.'
    write_json(PUBLIC / 'native-scale.json', record)
    print('native admission subset: '+record['status'], flush=True)
    return 0 if record['status'] == 'passed' else 2


if __name__ == '__main__':
    raise SystemExit(main())
