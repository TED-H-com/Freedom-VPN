import json
from pathlib import Path
import tempfile
import unittest

from reload_analysis import analyze, guards, parse_capture, position


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

    def test_timeout_uses_packet_ingress_not_late_read_completion(self):
        def stamp(ns):
            return dict(wall_ns=str(ns), mono_ns=str(ns))
        apply = dict(event='apply', generation='7', start=stamp(1_000_000_000),
                     end=stamp(1_100_000_000), verified=True,
                     deadline=dict(render_start_ns='1000000000', installed_by_ns='1010000000'),
                     batches=[dict(kind='guard_install', start=stamp(1_000_000_000), end=stamp(1_010_000_000), ok=True),
                              dict(kind='guard_release', start=stamp(1_020_000_000), end=stamp(1_030_000_000), ok=True)])
        events = [dict(event='udp_send', sequence=1, send_wall_ns='1015000000',
                       send_mono_ns='1015000000', source_port=40000, size=3),
                  dict(event='udp_read', sequence=1, wall_ns='2515000000', outcome='TimeoutError')]
        def header(iface):
            return ('1.015001 ' + iface + ' IP (tos 0x0, ttl 63, id 17, offset 0, flags [DF], proto UDP (17), length 31)\n'
                    '    100.64.1.1.40000 > 203.0.113.10.9000: UDP, length 3\n')
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root/'stream').write_text(''.join(json.dumps(e)+'\n' for e in events))
            (root/'apply').write_text(json.dumps(apply)+'\n')
            for role, text in [('origin', header('fv-wg0 Out')), ('node', header('fv-fh In')), ('responder', '')]:
                (root/('trace-'+role+'.txt')).write_text(text)
                count = int(bool(text))
                (root/('trace-'+role+'.err')).write_text(f'{count} packets captured\n0 packets dropped by kernel\n')
            result = analyze(root/'stream', root/'apply', root/'trace')
        self.assertEqual(result['summary']['timeouts_by_node_ingress'], {'inside': 1})
        self.assertEqual(result['summary']['timeouts_without_node_egress'], 1)
        self.assertEqual(result['datagrams'][0]['ingress_guard_generation'], '7')
        self.assertEqual(result['datagrams'][0]['read_wait_guard_generations'], ['7'])

    def test_incomplete_guard_is_not_a_measurement(self):
        with self.assertRaises(ValueError):
            guards([dict(event='apply', batches=[])])


if __name__ == '__main__':
    unittest.main()
