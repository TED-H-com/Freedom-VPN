import unittest
from analyze_followup26 import closing_reports, relative_ms, syn_interval

class ServingJoinTest(unittest.TestCase):
    def test_closing_report_is_after_admission_not_boot_failure(self):
        rows=[dict(observed_at='2026-11-01T12:00:00Z',member_forwarding_enabled=False),dict(observed_at='2026-11-01T12:00:02Z',member_forwarding_enabled=True),dict(observed_at='2026-11-01T12:00:03Z',member_forwarding_enabled=False)]
        self.assertEqual(closing_reports(rows,'2026-11-01T12:00:02Z','2026-11-01T12:00:04Z'),rows[2:])
    def test_wall_report_is_not_an_invented_close_instant(self):
        self.assertIsNone(relative_ms('2026-11-01T12:00:00Z',None))
        self.assertEqual(Decimal(relative_ms('2026-11-01T12:00:00.500000+00:00','1793534401')),Decimal('-500'))
    def test_syn_match_uses_headers_and_ingress_and_preserves_absence(self):
        member=dict(packet=1,mono_lower_ns='100',mono_upper_ns='110',tcp={'seq':1,'flags':2})
        node=dict(packet=2,mono_lower_ns='200',mono_upper_ns='210',tcp=member['tcp'],interface='fvmb0',packet_type=0)
        self.assertEqual(syn_interval(member,[node]),dict(member_packet=1,node_packet=2,lower_ns='90',upper_ns='110'))
        self.assertIsNone(syn_interval(member,[dict(node,interface='fvup0')]))
        self.assertIsNone(syn_interval(member,[dict(node,tcp={'seq':2,'flags':2})]))
        self.assertIsNone(syn_interval(None,[node]))

from decimal import Decimal
if __name__=='__main__':unittest.main()
