import struct
import unittest
from decode_capture import decode_ip, wireguard_header

class HeaderOnly(unittest.TestCase):
    def test_wireguard_fixed_fields_only_and_truncation(self):
        import json
        message = struct.pack('<III', 2, 123, 456) + b'PRIVATE!' * 10
        got = wireguard_header(message)
        self.assertEqual((got['type'], got['sender_index'], got['receiver_index']), (2, 123, 456))
        self.assertNotIn('PRIVATE', json.dumps(got))
        self.assertEqual(set(got), {'type', 'length', 'sender_index', 'receiver_index', 'message_sha256'})
        self.assertIsNone(wireguard_header(message[:-1]))
        self.assertIsNone(wireguard_header(bytes([2, 1]) + message[2:]))
        data = struct.pack('<IIQ', 4, 456, 2**64-1) + bytes(16)
        self.assertEqual(wireguard_header(data)['counter'], '18446744073709551615')

    def test_payload_is_never_exported(self):
        import json
        ip = bytes([0x45,0])+struct.pack('!H',48)+bytes(5)+bytes([6])+bytes(10)
        tcp = struct.pack('!HHIIHHHH',1111,9446,123,456,0x5018,42,0,0)
        got=decode_ip(ip+tcp+b'PRIVATE!',lambda _: 'endpoint')
        self.assertEqual(got['tcp']['seq'],123)
        self.assertEqual(got['tcp']['ack'],456)
        self.assertEqual(got['tcp']['window'],42)
        self.assertEqual(got['tcp']['length'],8)
        self.assertNotIn('PRIVATE',json.dumps(got))
    def test_short_options_refused(self):
        ip=bytes([0x45,0])+struct.pack('!H',60)+bytes(5)+bytes([6])+bytes(10)
        tcp=struct.pack('!HHIIHHHH',1111,9446,1,2,0xf010,42,0,0)
        with self.assertRaisesRegex(ValueError,'short TCP options'):
            decode_ip(ip+tcp,lambda _: 'endpoint')

    def test_ipv6_noninitial_fragment_and_ptb(self):
        ip6=bytes([0x60,0,0,0])+struct.pack('!H',16)+bytes([44,64])+bytes(32)
        fragment=bytes([17,0])+struct.pack('!H',1232)+struct.pack('!I',789)
        got=decode_ip(ip6+fragment+b'PRIVATE!',lambda _: 'endpoint')
        self.assertEqual(got['extensions'][0]['fragment_offset'],1232)
        self.assertNotIn('udp',got)
        ptb=bytes([0x60,0,0,0])+struct.pack('!H',8)+bytes([58,64])+bytes(32)+bytes([2,0,0,0])+struct.pack('!I',1280)
        self.assertEqual(decode_ip(ptb,lambda _: 'endpoint')['icmp']['mtu'],1280)

if __name__=='__main__':
    unittest.main()

class DNSHeaderOnly(unittest.TestCase):
    def test_closed_idna_wire_and_nonfixture_rejection(self):
        from decode_capture import dns_header
        import json
        def message(labels):
            return struct.pack('!6H',42,0x8180,1,0,0,0)+b''.join(bytes([len(x)])+x for x in labels)+b'\0'+struct.pack('!HH',1,1)
        got=dns_header(message([b'xn--bcher-kva',b'example',b'invalid']))
        self.assertEqual((got['fixture_case'],got['wire_id'],got['rcode']),('idna_positive',42,0))
        self.assertNotIn('xn--',json.dumps(got))
        self.assertIsNone(dns_header(message([b'private',b'example',b'invalid'])))
        self.assertIsNone(dns_header(message([b'rr',b'example',b'invalid'])[:-1]))

class NativeDebugAllowlist(unittest.TestCase):
    def test_no_arbitrary_log_or_identity_export(self):
        import json
        from decode_native_debug import decode
        raw=(b'2026-11-01T08:18:15.195Z INFO [peer: PRIVATE] client/internal/peer/conn.go:472: configure WireGuard endpoint to: [2001:db8::1]:51820\n'
             b'2026-11-01T08:18:15.196Z DEBG client/internal/peer/conn.go:484: redirect packets from relayed conn to WireGuard PRIVATE\n'
             b'2026-11-01T08:18:15.197Z INFO client/unknown.go:1: PRIVATE\n')
        got=decode(raw)
        self.assertEqual(len(got),2)
        self.assertEqual(got[0]['family'],'ipv6')
        self.assertNotIn('PRIVATE',json.dumps(got))
        self.assertNotIn('2001:db8',json.dumps(got))
