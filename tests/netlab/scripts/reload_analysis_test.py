import json
from pathlib import Path
import tempfile
import unittest

from reload_analysis import analyze, guards, parse_capture, position, classify_d120, acceptance_evidence
from copy import deepcopy


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


class D120Test(unittest.TestCase):
    def window(self):
        return dict(generation='1001', verified=True, upper_ns=30_000_000,
                    render_to_installed_ns=20_000_000,
                    install_start={'wall_ns': '1000000000'},
                    install_end={'wall_ns': '1010000000'},
                    release_start={'wall_ns': '1020000000'},
                    release_end={'wall_ns': '1030000000'})

    def datagram(self, sequence=1, loss=True, at=1_015_000_000):
        return dict(sequence=sequence, send={'sequence': sequence},
                    read=dict(outcome='TimeoutError') if loss else dict(outcome='ok', response_matches=True),
                    stages={'node_ingress': {'wall_ns': at} if at is not None else None})

    def verdict(self, **kw):
        ds = kw.pop('datagrams', [self.datagram(), self.datagram(2, False)])
        return classify_d120(ds, kw.pop('windows', [self.window()]), **kw)

    def assertFails(self, result, reason):
        self.assertFalse(result['passed'])
        self.assertIn(reason, json.dumps(result))

    def test_pass_no_loss_inside_boundary_and_inclusive_limits(self):
        self.assertTrue(self.verdict(datagrams=[self.datagram(loss=False)])['passed'])
        for at in (1_015_000_000, 1_005_000_000, 999_998_000, 1_030_002_000):
            with self.subTest(at=at):
                w = self.window()
                w['upper_ns'] = w['render_to_installed_ns'] = 100_000_000
                result = self.verdict(datagrams=[self.datagram(at=at), self.datagram(2, False)], windows=[w])
                self.assertTrue(result['passed'], result)

    def test_outside_unobserved_and_two_matching_brackets(self):
        for at, reason in [(999_997_999, 'outside'), (None, 'unobserved')]:
            self.assertFails(self.verdict(datagrams=[self.datagram(at=at), self.datagram(2, False)]), reason)
        # An inside match cannot hide another window's outer-boundary overlap.
        w2 = self.window()
        w2['generation'] = '1002'
        w2['install_end']['wall_ns'] = '1016000000'
        self.assertFails(self.verdict(windows=[self.window(), w2]), 'ambiguous')

    def test_upper_and_install_over_limit_or_invalid(self):
        for field in ('upper_ns', 'render_to_installed_ns'):
            for value in (100_000_001, -1):
                w = self.window()
                w[field] = value
                self.assertFails(self.verdict(windows=[w]), '(ii)')

    def test_unverified_apply(self):
        w = self.window()
        w['verified'] = False
        self.assertFails(self.verdict(windows=[w]), 'Apply not verified')

    def test_next_datagram_must_succeed_and_exist(self):
        self.assertFails(self.verdict(datagrams=[self.datagram(), self.datagram(2), self.datagram(3, False)]), '(iii)')
        self.assertFails(self.verdict(datagrams=[self.datagram()]), 'next datagram missing')

    def test_tcp_timeout_even_without_udp_loss(self):
        for ds in ([self.datagram(loss=False)], [self.datagram(), self.datagram(2, False)]):
            self.assertFails(self.verdict(datagrams=ds, tcp_errors=['TimeoutError']), '(iv)')

    def test_five_losses_pass_six_fail(self):
        ds = [self.datagram(i+1, i % 2 == 0) for i in range(10)]
        self.assertTrue(self.verdict(datagrams=ds)['passed'])
        ds.extend([self.datagram(11), self.datagram(12, False)])
        result = self.verdict(datagrams=ds)
        self.assertFails(result, '(v)')
        self.assertTrue(all(any('(v)' in f for f in r['failures']) for r in result['losses']))

    def test_missing_sidecar_trace_and_failed_capture(self):
        for field in ('send', 'read'):
            d = self.datagram()
            d[field] = None
            self.assertFails(self.verdict(datagrams=[d, self.datagram(2, False)]), 'missing sidecar')
        self.assertFails(self.verdict(evidence_errors=['capture count mismatch']), 'capture count mismatch')
        self.assertFails(self.verdict(datagrams=[]), 'no datagrams')
        self.assertFails(self.verdict(reloads=99), 'requires 100 reloads')

    def test_non_timeout_and_wrong_echo_are_not_tolerated(self):
        d = self.datagram()
        for read in (dict(outcome='OSError'), dict(outcome='ok', response_matches=False), dict(outcome='ok')):
            d['read'] = read
            self.assertFails(self.verdict(datagrams=[d, self.datagram(2, False)]), 'not a lost echo')

    def test_sidecar_text_integrity(self):
        ds = [self.datagram(loss=False)]
        ds[0]['stages']['origin_egress'] = {'wall_ns': 1}
        events = [dict(event=k, sequence=1, **fields) for k, fields in (
            ('tcp_send', {}), ('tcp_read', {'size': 3}), ('udp_send', {}),
            ('udp_read', {'outcome': 'ok', 'response_matches': True}), ('trace_end', {}))]
        windows = [dict(self.window(), generation=str(g)) for g in range(1000, 1101)]
        result = dict(events=events, windows=windows, datagrams=ds, summary={'incomplete_udp_reads': 0})
        self.assertTrue(acceptance_evidence(result, '1 up\n2 ok\n')['passed'])
        self.assertFails(acceptance_evidence(result, '1 up\n2 fail TimeoutError\n'), 'text/sidecar')
        for index in range(len(events)):
            broken = deepcopy(result)
            del broken['events'][index]
            self.assertFalse(acceptance_evidence(broken, '1 up\n2 ok\n')['passed'])
        for field in ('origin_egress', 'node_ingress'):
            broken = deepcopy(result)
            broken['datagrams'][0]['stages'][field] = None
            self.assertFails(acceptance_evidence(broken, '1 up\n2 ok\n'), 'missing trace')
        broken = deepcopy(result)
        broken['windows'].pop()
        self.assertFails(acceptance_evidence(broken, '1 up\n2 ok\n'), 'generations')


if __name__ == '__main__':
    unittest.main()
