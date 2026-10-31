#!/usr/bin/env python3
"""Verify immutable archives, current pins, first delivery and public hygiene."""
import hashlib
import json
import subprocess

from followup13_records import ARCHIVE, closure_guards, digest, load
from followup9_records import clean_inventory
from run import ROOT, PUBLIC, scan


def main():
    index = load(PUBLIC / 'history/index.json')
    total = 0
    for name, entry in index.items():
        for item in entry.get('snapshot_files', []):
            path = PUBLIC / 'history' / name.replace('_', '-') / item['file']
            assert digest(path) == item['sha256'], str(path)
            if name == 'follow_up_12':
                merged = subprocess.check_output(['git','show','8b5e809b:docs/e2e/at/secops/'+item['file']], cwd=ROOT)
                assert path.read_bytes() == merged, item['file']
            total += 1
    run, live = (load(PUBLIC / ('followup13-restore-' + n + '.json')) for n in ('run', 'live'))
    assert run['follow_up'] == live['follow_up'] == 13
    assert run['phase'] == 'restore' and run['finished_at']
    assert run['source_revision'] == live['source_revision']
    probe = hashlib.sha256()
    for path in sorted((ROOT/'tests/e2e/at/secops').glob('*.go')):
        probe.update(path.read_bytes())
    assert probe.hexdigest() == live['probe_source_sha256'], 'measured Go probe differs from current source'
    assert run['live']['status'] == 'passed' and live['go_test_failed'] is False
    assert clean_inventory(run['inventory_after']) and run['state_cleanup']['removed']
    assert all(not r['known_secret_matches'] and not r['device_flow_key_lines'] and r['read_ok'] for r in live['secret_scan'])
    expected = closure_guards(run, live)
    record = load(PUBLIC / 'SECOPS-21.json')
    assert record['closure_guards'] == expected
    assert record['request_status'] == ('closed' if all(expected.values()) else 'open')
    for at in ('AT29','AT31'):
        current, old = load(PUBLIC / (at+'.json')), load(ARCHIVE / (at+'.json'))
        assert current['status'] == 'blocked' and current['run_id'] == old['run_id']
        assert current['actual'] == old['actual'] and current['blockers'] == old['blockers']
    count = 0
    def verify(value):
        nonlocal count
        if isinstance(value, dict):
            if 'path' in value and 'sha256' in value:
                assert digest(ROOT/value['path']) == value['sha256'], value['path']
                count += 1
            for child in value.values(): verify(child)
        elif isinstance(value, list):
            for child in value: verify(child)
    verify(load(PUBLIC/'scope-review.json'))
    for path in PUBLIC.iterdir():
        if path.suffix in ('.json','.md'): assert not scan(path.read_bytes()), path.name
    # Scan each committed evidence blob, including archives newly added by this lane.
    paths = subprocess.check_output(['git','diff','--name-only','--diff-filter=AM','main','--','docs/e2e/at/secops'], cwd=ROOT, text=True).splitlines()
    for name in paths:
        raw = (ROOT/name).read_bytes()
        assert not scan(raw), name
    print(f'PASS: {total} archive pins (Follow-up 12 byte-identical to #301), {count} scope pins; one completed restore (pre-arm failure archived); first-delivery verdict {record["request_status"]}; AT29/AT31 unchanged blocked; cleanup zero; evidence hygiene')


if __name__ == '__main__': main()
