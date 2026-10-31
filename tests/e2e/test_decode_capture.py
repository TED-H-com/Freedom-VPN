import struct
import unittest
from decode_capture import decode_ip

class HeaderOnly(unittest.TestCase):
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
