"""TEST ONLY synthetic agent CLI fixtures, using agent-core 52's field names."""
import json
import unittest
import observation_records as evidence

A = '00000000-0000-4000-8000-000000000001'
B = '00000000-0000-4000-8000-000000000002'
C = '00000000-0000-4000-8000-000000000003'


def enqueue(rid, at, reason):
    return f'time=2026-11-04T10:00:{at}Z level=INFO msg="observation enqueued" report_id={rid} observed_at=2026-11-04T10:00:{at}Z due_reason={reason}\n'


def submit(rid, at, attempt, outcome='accepted', process=42):
    accepted = rid if outcome == 'accepted' else ''
    status = 202 if outcome == 'accepted' else 503
    retry = '<nil>' if outcome == 'accepted' else '2026-11-04T10:00:04Z'
    code = '' if outcome == 'accepted' else 'DEPENDENCY_UNAVAILABLE'
    return f'time=2026-11-04T10:00:{at}.100Z level=INFO msg="observation submission" report_ids=[{rid}] attempt_numbers=[{attempt}] attempt_scope=process process_id={process} boot_id="TEST ONLY HOST BOOT" started_at=2026-11-04T10:00:{at}.000000001Z duration_ms=99 outcome={outcome} accepted_report_ids=[{accepted}] http_status={status} error_code="{code}" transport_error_class="" next_retry_at={retry} retry_after_ms=0\n'


def observation(rid, at):
    return dict(report_id=rid, received_at=f'2026-11-04T10:00:{at}Z', generation='9007199254740993')


def extract(before, after, stored):
    return evidence.extract(before, after, '2026-11-04T10:00:00Z', '2026-11-04T10:00:10Z', stored)


class ObservationTests(unittest.TestCase):
    def test_accepted_retry_and_half_open_window(self):
        before = 'time=2026-11-04T09:59:59Z level=INFO msg=started\n'
        after = before + enqueue(A, '00', 'closed_recovery_binding') + submit(A, '01', 1)
        after += enqueue(B, '02', 'interval_elapsed') + submit(B, '03', 1, 'rejected') + submit(B, '05', 2)
        after += enqueue(C, '09', 'state_changed')
        # Boundary excluded; unrelated secret-bearing lines are never emitted.
        after += enqueue(C, '10', 'state_changed')
        after += 'time=2026-11-04T10:00:06Z level=ERROR msg="TEST ONLY SECRET" token=TEST_ONLY_TOKEN url=https://example.invalid/secret\n'
        v = extract(before, after, [observation(A, '01.050'), observation(B, '05.050')])
        self.assertEqual(len(v['enqueues']), 3)
        self.assertEqual(len(v['submission_attempts']), 3)
        self.assertEqual(v['submission_attempts'][1]['next_retry_at'], '2026-11-04T10:00:04Z')
        self.assertEqual(v['submission_attempts'][2]['attempt_numbers'], [2])
        self.assertEqual(v['stored_observation_joins'][0]['enqueue_to_stored_seconds'], '1.05')
        self.assertEqual(v['stored_observation_joins'][1]['enqueue_to_attempt_seconds'], '3.000000001')
        self.assertEqual(v['enqueued_not_stored_in_window'][0]['report_id'], C)
        self.assertEqual(v['stored_observations_missing_records'], [])
        self.assertEqual(v['malformed_receipts'], [])
        for secret in ['TEST ONLY HOST BOOT', 'TEST_ONLY_TOKEN', 'example.invalid', 'TEST ONLY SECRET']:
            self.assertNotIn(secret, json.dumps(v))

    def test_overwrite_retains_start_export_and_process_attempt_reset(self):
        before = enqueue(A, '00', 'first_report') + submit(A, '01', 1, 'rejected')
        after = 'time=2026-11-04T10:00:07Z level=INFO msg=started\n' + submit(A, '08', 1, process=99) + submit(B, '09', 1, process=99)
        v = extract(before, after, [observation(A, '08.050'), observation(B, '09.050')])
        self.assertTrue(v['overwrite_or_replacement_detected'])
        self.assertEqual(v['missing_prefix_seconds'], '7')
        self.assertEqual(v['surviving_end_export_window_seconds'], '3')
        joined = v['stored_observation_joins'][0]
        self.assertEqual(joined['enqueue']['source'], 'start_export')
        self.assertEqual([r['process_id'] for r in joined['attempts']], [42, 99])
        self.assertEqual([r['attempt_numbers'] for r in joined['attempts']], [[1], [1]])
        self.assertEqual(v['stored_observations_missing_records'], [dict(report_id=B, missing=['enqueue'])])

    def test_invalid_records_are_counted_without_leaking_fields(self):
        raw = enqueue(A, '01', 'TEST_ONLY_SECRET') + submit(A, '02', 1).replace('attempt_numbers=[1]', 'attempt_numbers=[]')
        v = extract('', raw, [observation(A, '02.050')])
        self.assertFalse(v['submission_events_available'])
        self.assertEqual(len(v['malformed_receipts']), 2)
        self.assertEqual(v['stored_observations_missing_records'][0]['missing'], ['enqueue', 'accepted_attempt'])
        self.assertNotIn('TEST_ONLY_SECRET', json.dumps(v))


if __name__ == '__main__':
    unittest.main()
