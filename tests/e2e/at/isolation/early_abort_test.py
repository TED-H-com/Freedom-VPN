"""Synthetic current-run setup abort shaped like 70beb28e; no stack or replay."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from review_populated_holds import review as holds
from review_holds import review as general_holds
from review_stops import review as stops

HERE = Path(__file__).resolve().parent
FIXTURE = json.loads((HERE/'testdata/early-abort.json').read_text())


class EarlyAbort(unittest.TestCase):
    def setUp(self):
        disk = HERE.parents[3]/'.agent/tmp'
        disk.mkdir(parents=True, exist_ok=True)
        self.temp = tempfile.TemporaryDirectory(dir=disk)
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for name, value in {
            'AT10.json': FIXTURE, 'AT13.json': FIXTURE,
            'receiver-capture.json': dict(run_id=FIXTURE['run_id'], transactions=[]),
            'container-lifecycle-events.json': dict(events=[]),
            'receiver-closure.json': [dict(container='fvpn-w4-isolation-node-clone', status='not created', stage='clone_creation', reason='not created: setup exited before clone creation', ok=False)],
        }.items():
            self.write(name, value)

    def write(self, name, value):
        (self.root/name).write_text(json.dumps(value))

    def test_populated_holds_and_every_reader_unavailable(self):
        for review in (holds, general_holds, stops):
            with self.subTest(reader=review.__module__):
                result = review(self.root)
                self.assertEqual(result['run_id'], FIXTURE['run_id'])
                self.assertEqual(result['status'], 'unavailable')
                self.assertIn('unavailable: populated holds were not executed', json.dumps(result))
        self.assertFalse(stops(self.root)['all_parent_receipts_matched'])
        run = subprocess.run(['python3', '-B', str(HERE/'review_populated_holds.py'), str(self.root)], capture_output=True, text=True)
        self.assertEqual(run.returncode, 1)
        self.assertEqual(run.stderr, '')
        self.assertIn('populated_holds', run.stdout)
        self.assertIn('unavailable: populated holds were not executed', run.stdout)

    def test_recovery_unavailable_and_no_creator(self):
        run = subprocess.run(['python3', '-B', str(HERE/'review_followup8.py'), str(self.root)], capture_output=True, text=True)
        self.assertEqual(run.returncode, 0, run.stderr)
        self.assertEqual(run.stderr, '')
        result = json.loads((self.root/'recovery-verdict.json').read_text())
        self.assertEqual(result['run_id'], FIXTURE['run_id'])
        self.assertEqual(result['reason'], 'injection not reached')
        self.assertEqual(result['status'], 'unavailable')
        self.assertFalse(result['passed'])
        self.assertTrue(all(not c['passed'] for c in result['assertions']))
        self.assertNotIn('injected_creator', result)

    def test_creator_requires_current_run_injection(self):
        for run_id, ok, creator in [('previous-run', True, False), (FIXTURE['run_id'], False, False), (FIXTURE['run_id'], True, True)]:
            with self.subTest(run_id=run_id, ok=ok):
                self.write('foreign-redirect.json', dict(run_id=run_id, rows=[dict(label='inject clsact mirred', ok=ok)]))
                run = subprocess.run(['python3', '-B', str(HERE/'review_followup8.py'), str(self.root)], capture_output=True, text=True)
                self.assertEqual(run.returncode, 0, run.stderr)
                result = json.loads((self.root/'recovery-verdict.json').read_text())
                self.assertEqual('injected_creator' in result, creator)
                self.assertFalse(result['passed'])

    def test_operation_collector_without_lab_record(self):
        from capture_hold_operations import capture
        stopped = subprocess.CompletedProcess([], 1, '', 'TEST ONLY database unavailable')
        # No lab record exists. The runner sets the invocation ID before any AT
        # record is emitted, so the collector can start before acceptance.
        (self.root/'AT10.json').unlink()
        (self.root/'AT13.json').unlink()
        (self.root/'receiver-capture.json').unlink()
        with patch.dict(os.environ, {'FVPN_AT_RUN_ID': FIXTURE['run_id']}), patch('capture_hold_operations.subprocess.run', return_value=stopped):
            result = capture(self.root)
        self.assertEqual(result['run_id'], FIXTURE['run_id'])
        self.assertEqual(result['snapshots'], [])
        self.assertEqual(json.loads((self.root/'publication-operation-history.json').read_text())['run_id'], FIXTURE['run_id'])

    def test_operation_history_failure_is_unavailable(self):
        self.write('generation-holds-central.json', dict(run_id=FIXTURE['run_id'], reports=[], final={}))
        result = general_holds(self.root)
        self.assertEqual(result['run_id'], FIXTURE['run_id'])
        self.assertEqual(result['status'], 'unavailable')
        self.assertEqual(result['stage'], 'publication_operation_history')
        self.assertFalse(result['passed'])

    def test_old_populated_evidence_cannot_pass_current_abort(self):
        self.write('populated-holds.json', dict(run_id='previous-run', attempts=[]))
        self.assertEqual(holds(self.root)['status'], 'unavailable')


if __name__ == '__main__':
    unittest.main()
