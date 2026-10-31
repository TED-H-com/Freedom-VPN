import unittest

from mtu_measure import capture_check


class CaptureCheckTest(unittest.TestCase):
    def test_counts_multicast_and_both_ip_families_without_hiding_drops(self):
        text = ('1.000001 fv-fh In IP (header)\n'
                '1.000002 fv-fh M IP6 (header)\n'
                '1.000003 fv-ih Out IP (header)\n')
        stats = '3 packets captured\n0 packets dropped by kernel\n'
        self.assertTrue(capture_check(text, stats)['ok'])
        self.assertEqual(capture_check(text, stats)['parsed_headers'], 3)
        for bad in (stats.replace('3 packets captured', '4 packets captured'),
                    stats.replace('0 packets dropped', '1 packets dropped'), ''):
            self.assertFalse(capture_check(text, bad)['ok'])
        self.assertFalse(capture_check(text+'1.000004 fv-fh unexpected IP\n',
                                       stats.replace('3 packets captured', '4 packets captured'))['ok'])


if __name__ == '__main__':
    unittest.main()
