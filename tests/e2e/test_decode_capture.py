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

if __name__=='__main__':
    unittest.main()
