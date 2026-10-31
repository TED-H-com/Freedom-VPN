import unittest

from reload_analysis import guards, parse_capture, position


class ReloadAnalysisTest(unittest.TestCase):
    def test_header_only_capture_requires_complete_zero_drop_record(self):
        text = ('1793440000.123456 fv-fh In IP (tos 0x0, ttl 63, id 17, offset 0, flags [DF], proto UDP (17), length 31)\n'
                '    100.64.1.1.40000 > 203.0.113.10.9000: UDP, length 3\n')
        stats = '1 packets captured\n1 packets received by filter\n0 packets dropped by kernel\n'
        row = parse_capture(text, stats)[0]
        self.assertEqual(row['wall_ns'], 1793440000123456000)
        self.assertEqual(row['ip_id'], 17)
        self.assertEqual(row['direction'], 'In')
        for invalid in (stats.replace('0 packets dropped', '1 packets dropped'), stats.replace('1 packets captured', '2 packets captured'), ''):
            with self.assertRaises(ValueError):
                parse_capture(text, invalid)

    def test_guard_boundary_is_not_claimed_as_proven_inside(self):
        window = dict(generation='23', install_start={'wall_ns': '1000000'},
                      install_end={'wall_ns': '2000000'}, release_start={'wall_ns': '3000000'},
                      release_end={'wall_ns': '4000000'})
        for at, want in [(500000, 'outside'), (1500000, 'boundary'), (2000001, 'boundary'),
                         (2500000, 'inside'), (3500000, 'boundary'), (5000000, 'outside'), (None, 'unobserved')]:
            self.assertEqual(position(at, [window])[0], want)

    def test_incomplete_guard_is_not_a_measurement(self):
        with self.assertRaises(ValueError):
            guards([dict(event='apply', batches=[])])


if __name__ == '__main__':
    unittest.main()
