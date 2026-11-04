import unittest

from followup16_records import verdict


class Followup16Records(unittest.TestCase):
    def test_failed_subject_is_not_promoted_by_other_passes(self):
        cases = {name: dict(status='passed') for name in
                 ('grant_expiry', 'lease_expiry', 'control_lost_agent_alive', 'device_revoke', 'manual_grant_withdrawal')}
        cases['guest_clock_rollback'] = dict(status='blocked')
        self.assertEqual(verdict(cases), 'blocked')
        cases['control_lost_agent_alive']['status'] = 'failed'
        self.assertEqual(verdict(cases), 'failed')

    def test_missing_subject_is_blocked(self):
        self.assertEqual(verdict({}), 'blocked')
