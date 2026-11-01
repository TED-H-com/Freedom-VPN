import unittest
from analyze_followup26 import closing_reports, relative_ms

class ServingJoinTest(unittest.TestCase):
    def test_closing_report_is_after_admission_not_boot_failure(self):
        rows=[dict(observed_at='2026-11-01T12:00:00Z',member_forwarding_enabled=False),dict(observed_at='2026-11-01T12:00:02Z',member_forwarding_enabled=True),dict(observed_at='2026-11-01T12:00:03Z',member_forwarding_enabled=False)]
        self.assertEqual(closing_reports(rows,'2026-11-01T12:00:02Z','2026-11-01T12:00:04Z'),rows[2:])
    def test_wall_report_is_not_an_invented_close_instant(self):
        self.assertIsNone(relative_ms('2026-11-01T12:00:00Z',None))
        self.assertEqual(Decimal(relative_ms('2026-11-01T12:00:00.500000+00:00','1793534401')),Decimal('-500'))

from decimal import Decimal
if __name__=='__main__':unittest.main()
