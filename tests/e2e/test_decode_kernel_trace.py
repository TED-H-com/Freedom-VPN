import json
import unittest
from decode_kernel_trace import decode, token


class KernelTraceTest(unittest.TestCase):
    def test_closed_metadata(self):
        raw = b'trace id deadbeef netdev fvtrace ingress_eth1 packet: iif "eth1" ip6 saddr ::1 ip6 daddr 2001:db8::41 ip6 length 92 udp sport 51820 udp dport 51820\ntrace id deadbeef netdev fvtrace ingress_eth1 rule ip6 saddr ::1 comment "TEST ONLY private expression" (verdict continue)\n'
        result = decode(raw)
        self.assertEqual(result['unparsed_lines'], 0)
        self.assertEqual(result['events'][0]['saddr'], token('::1'))
        self.assertEqual(result['events'][0]['length'], 92)
        self.assertEqual(result['events'][1]['verdict'], 'continue')
        self.assertNotIn('private expression', json.dumps(result))
        self.assertNotIn('2001:', json.dumps(result))

    def test_unparsed_is_not_silence(self):
        result = decode(b'unexpected\n')
        self.assertEqual(result['unparsed_lines'], 1)
        self.assertEqual(result['events'], [])


if __name__ == '__main__':
    unittest.main()
