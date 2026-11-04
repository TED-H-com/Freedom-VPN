import copy
import unittest

from followup17_records import qualifying_outage


class Followup17Records(unittest.TestCase):
    def test_resolution_requires_qualifying_outage(self):
        row = dict(lease_setup_qualified=True,control_stop_completed_at='2026-11-04T06:00:05Z',
                   lease_setup=dict(cap_changed_at='2026-11-04T06:00:00Z'),
                   signed_lease=dict(lease_id='TEST ONLY',max_offline_seconds=60,
                                     issued_at='2026-11-04T06:00:01Z',expires_at='2026-11-04T06:01:01Z'))
        self.assertTrue(qualifying_outage(row))
        for key in ('lease_setup_qualified','control_stop_completed_at','signed_lease','lease_setup'):
            bad=copy.deepcopy(row)
            del bad[key]
            self.assertFalse(qualifying_outage(bad))
        for change in (dict(max_offline_seconds=86400),dict(issued_at='2026-11-04T05:59:59Z',expires_at='2026-11-04T06:00:59Z'),dict(expires_at='2026-11-04T06:02:01Z')):
            bad=copy.deepcopy(row)
            bad['signed_lease'].update(change)
            self.assertFalse(qualifying_outage(bad))
