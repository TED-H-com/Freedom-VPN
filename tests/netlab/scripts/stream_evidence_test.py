"""TEST ONLY: additive echo metadata must preserve every D-120 verdict."""
from copy import deepcopy
import unittest

from reload_analysis import acceptance_evidence
from stream import tcp_echo_fields
from trace_split import udp_projection


class StreamEvidenceTest(unittest.TestCase):
    def result(self, timeout):
        events, datagrams, logs = [], [], ['0 up']
        for seq in range(1, 4):
            events.append(dict(event='tcp_send', sequence=seq))
            if seq == 1 and timeout:
                events.append(dict(event='tcp_read', sequence=seq, outcome='TimeoutError'))
                logs.append('1 fail TimeoutError')
                continue
            response = f't{seq-1 if timeout else seq}\n'.encode()
            events.append(dict(event='tcp_read', sequence=seq, **tcp_echo_fields(response, seq)))
            events.extend([dict(event='udp_send', sequence=seq),
                           dict(event='udp_read', sequence=seq, outcome='ok', response_matches=True)])
            datagrams.append(dict(sequence=seq, send={}, read=dict(outcome='ok', response_matches=True),
                                  stages=dict(node_ingress={'wall_ns': 1}, origin_egress={'wall_ns': 1})))
            logs.append(f'{seq} ok')
        events.append(dict(event='trace_end', sequence=3))
        return dict(events=events, windows=[dict(generation=str(g), verified=True) for g in range(1000, 1101)],
                    datagrams=datagrams, summary={'incomplete_udp_reads': 0}), '\n'.join(logs)

    def test_timeout_then_stale_is_one_timeout_unchanged_verdict(self):
        result, log = self.result(True)
        reads = [e for e in result['events'] if e['event'] == 'tcp_read']
        self.assertEqual([e.get('echo_status') for e in reads], [None, 'stale', 'stale'])
        self.assertEqual([e.get('response_sequence') for e in reads], [None, 1, 2])
        verdict = acceptance_evidence(result, log)
        self.assertEqual(verdict['errors'], ['(iv): TCP failure: 1:TimeoutError'])
        self.assertFalse(verdict['passed'])
        self.assert_legacy_identical(result, log)

    def assert_legacy_identical(self, result, log):
        old = deepcopy(result)
        for e in old['events']:
            if e['event'] == 'tcp_read':
                for key in ('echo_status', 'response_matches', 'response_sequence'):
                    e.pop(key, None)
        self.assertEqual(acceptance_evidence(result, log), acceptance_evidence(old, log))

    def test_recovered_reads_keep_current_success_classification(self):
        result, log = self.result(False)
        # A retransmission recovered before the read bound remains a success.
        for e in result['events']:
            e['mono_ns'] = str(e['sequence'] * 2_000_000_000 + (416_739_885 if e['event'] == 'tcp_read' else 0))
        self.assertTrue(acceptance_evidence(result, log)['passed'])
        self.assert_legacy_identical(result, log)

    def test_unknown_partial_coalesced_and_eof_do_not_invent_identity(self):
        for response in (b't', b't1\nt2\n', b'unknown', b't99\n', b't01\n'):
            fields = tcp_echo_fields(response, 2)
            self.assertEqual(fields['echo_status'], 'mismatch')
            self.assertIsNone(fields['response_sequence'])
            self.assertFalse(fields['response_matches'])
        self.assertEqual(tcp_echo_fields(b'', 2)['echo_status'], 'eof')


class CombinedCaptureTest(unittest.TestCase):
    udp = ('1.000000 fv-fh In IP (tos 0x0, ttl 63, id 17, offset 0, flags [DF], proto UDP (17), length 31)\n'
           '    100.64.1.1.40000 > 203.0.113.10.9000: UDP, length 3\n')
    tcp = ('1.000001 fv-fh In IP (tos 0x0, ttl 63, id 18, offset 0, flags [DF], proto TCP (6), length 55)\n'
           '    100.64.1.1.40001 > 203.0.113.10.9001: Flags [P.], cksum 0x0000 (incorrect -> 0x1234), seq 1:4, ack 1, win 512, options [nop,nop,TS val 1 ecr 1], length 3\n')
    stats = '2 packets captured\n4 packets received by filter\n0 packets dropped by kernel\n'

    def test_complete_combined_capture_projects_exact_udp(self):
        text, stats, meta = udp_projection(self.udp + self.tcp, self.stats)
        self.assertEqual(text, self.udp)
        self.assertIn('1 packets captured', stats)
        self.assertEqual((meta['combined_packets'], meta['udp_packets'], meta['tcp_packets']), (2, 1, 1))
        self.assertTrue(meta['complete'])

    def test_lost_unparsed_or_truncated_tcp_cannot_pass_udp_projection(self):
        for text, stats in [(self.udp, self.stats),
                            (self.udp + self.tcp, self.stats.replace('0 packets dropped', '1 packets dropped')),
                            (self.udp + self.tcp.replace('Flags [P.]', '[|tcp]'), self.stats),
                            (self.udp + self.tcp.splitlines(keepends=True)[0], self.stats),
                            (self.udp + self.tcp + '    unexpected payload output\n', self.stats)]:
            with self.assertRaises(ValueError):
                udp_projection(text, stats)


if __name__ == '__main__':
    unittest.main()
