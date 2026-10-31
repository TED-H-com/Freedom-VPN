"""A later success, compiler row or probe cannot replace the first delivery."""
import copy
import unittest

from followup13_records import closure_guards


def fixture():
    run = {'live': {'status': 'passed'}, 'd099_roadmap_present': True}
    stamp = lambda n: f'2026-10-31T00:00:{n:02d}Z'
    live = {'go_test_failed': False, 'D-095': {'passed': True},
            'SECOPS-21': {'manual_generation_adjustment': False},
            'first_desired_offer': {
                'arm_count': 1, 'capture_id': 'TEST ONLY capture', 'armed_at': stamp(0),
                'agent_stopped_before_arm': True, 'retained_generation_ordering_proved': True,
                'retained_before_restore': {'node_accepted_desired': '15', 'node_applied': '15'},
                'retained_generation_observation': {'report_id': 'TEST ONLY retained report', 'applied_generation': '15',
                                                  'central_applied': '15', 'received_at': stamp(1)},
                'release_started_at': stamp(2),
                'capture': {'capture_id': 'TEST ONLY capture',
                    'first': {'http_status': 200, 'generation': '16', 'bytes': 100, 'write_complete': True,
                              'body_sha256': 'a'*64, 'delivered_at': stamp(3)},
                    'local': {'accepted_generation': '16', 'applied_generation': '16',
                              'envelope_sha256': 'a'*64, 'sampled_at': stamp(5)},
                    'ack': {'report_id': 'TEST ONLY applied report', 'applied_generation': '16',
                            'http_status': 202, 'received_at': stamp(5)}},
                'central_receipt': {'report_id': 'TEST ONLY applied report', 'node_id': 'TEST ONLY node',
                                    'applied_generation': '16', 'received_at': stamp(4)}}}
    return run, live


class FirstDeliveryTests(unittest.TestCase):
    def test_full_first_delivery_proof(self):
        run, live = fixture()
        self.assertTrue(all(closure_guards(run, live).values()))

    def test_missing_or_mismatched_evidence_keeps_open(self):
        changes = [
            ('capture.first.http_status', 304), ('capture.first.http_status', 503),
            ('capture.first.generation', '15'), ('capture.first.generation', '14'),
            ('capture.first.generation', ''), ('capture.first.generation', 16),
            ('capture.first.generation', '016'), ('capture.first.write_complete', False),
            ('capture.local.envelope_sha256', 'b'*64), ('capture.local.applied_generation', '17'),
            ('capture.ack.applied_generation', '17'), ('capture.ack.http_status', 503),
            ('central_receipt.report_id', 'TEST ONLY unrelated'), ('central_receipt.applied_generation', '17'),
            ('capture.capture_id', 'TEST ONLY later capture'), ('arm_count', 2),
            ('retained_before_restore.node_applied', '14'),
            ('retained_generation_observation.received_at', '2026-10-31T00:00:06Z'),
            ('capture.local.sampled_at', '2026-10-31T00:00:01Z'),
        ]
        run, live = fixture()
        for field, value in changes:
            with self.subTest(field=field, value=value):
                changed = copy.deepcopy(live)
                row = changed['first_desired_offer']
                parts = field.split('.')
                for key in parts[:-1]:
                    row = row[key]
                row[parts[-1]] = value
                self.assertFalse(all(closure_guards(run, changed).values()))
        for key in ('first', 'local', 'ack'):
            with self.subTest(missing=key):
                changed = copy.deepcopy(live)
                del changed['first_desired_offer']['capture'][key]
                self.assertFalse(all(closure_guards(run, changed).values()))

    def test_later_probe_and_compiler_success_do_not_close_first_304(self):
        run, live = fixture()
        live['first_desired_offer']['capture']['first'].update(http_status=304, generation='')
        live['first_http_probe_after_fresh_observation'] = {'http_200': True, 'generation': '17'}
        live['SECOPS-21'].update(offers=[{'generation':'17'}], final={'node_applied':'17'}, observations=[{'applied':'17'}])
        self.assertFalse(all(closure_guards(run, live).values()))

    def test_first_delivery_need_not_equal_separate_probe_or_compiler_row(self):
        run, live = fixture()
        live['first_http_probe_after_fresh_observation'] = {'generation': '17'}
        live['SECOPS-21']['offers'] = [{'generation': '14'}, {'generation':'16'}]
        self.assertTrue(all(closure_guards(run, live).values()))

    def test_uint64_generation_comparison_is_exact(self):
        run, live = fixture()
        row = live['first_desired_offer']
        prior, new = '9007199254740992', '9007199254740993'
        row['retained_before_restore'].update(node_accepted_desired=prior, node_applied=prior)
        row['retained_generation_observation'].update(applied_generation=prior, central_applied=prior)
        row['capture']['first']['generation'] = new
        row['capture']['local'].update(accepted_generation=new, applied_generation=new)
        row['capture']['ack']['applied_generation'] = new
        row['central_receipt']['applied_generation'] = new
        self.assertTrue(all(closure_guards(run, live).values()))
