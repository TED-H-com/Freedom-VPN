import json
import struct
import unittest
import underlay_evidence as evidence


class CaptureTests(unittest.TestCase):
    def test_realistic_ipv6_handshake_and_sanitizer(self):
        # TEST ONLY synthetic IPv6 UDP initiation. Opaque key-like bytes are
        # deliberately present in the packet and must never leave the decoder.
        import ipaddress
        message=b'\x01\x00\x00\x00'+struct.pack('<I',7)+b'K'*140
        udp=struct.pack('!HHHH',51820,51820,len(message)+8,0)+message
        ip=struct.pack('!IHBB',6<<28,len(udp),17,64)+ipaddress.ip_address('2001:db8::1').packed+ipaddress.ip_address('2001:db8::2').packed+udp
        address=lambda b:'addr-'+__import__('hashlib').sha256(str(ipaddress.ip_address(b)).encode()).hexdigest()[:16]
        packet=evidence.decoder.decode_ip(ip,address,wireguard=True)
        packet.update(packet=1,timestamp='1793790000.123456',interface_index=2,packet_type=4,publicKey='TEST_ONLY_PUBLIC',endpoint='private.example.invalid',arbitrary={'address':'192.0.2.1'})
        view=evidence.packet_view(packet,{2:'eth1'},{51820})
        self.assertEqual(view['ip_length'],196)
        self.assertEqual(view['wireguard'],dict(type=1,length=148))
        for text in ('2001:db8','192.0.2','TEST_ONLY','example.invalid','KKKK'):
            self.assertNotIn(text,json.dumps(view))
        self.assertIsNone(evidence.packet_view(packet,{2:'fvnb0'},{51820}))
        packet['src']='192.0.2.1'
        self.assertIsNone(evidence.packet_view(packet,{2:'eth1'},{51820}))

    def test_icmp6_and_multiplicity(self):
        p=dict(packet=1,timestamp='1793790000.123456',interface_index=2,packet_type=0,protocol=58,ip_version=6,ip_length=128,src='addr-'+'a'*16,dst='addr-'+'b'*16,icmp=dict(type=2,code=0,mtu=1280,raw='DROP'))
        self.assertEqual(evidence.packet_view(p,{2:'eth1'},set())['icmp'],dict(type=2,code=0,mtu=1280))
        tx=[dict(packet=i,ip_sha256='a'*64) for i in [1,2]]
        self.assertEqual([p['receiver_seen'] for p in evidence.join_direction(tx,tx[:1])],[True,False])


if __name__=='__main__':
    unittest.main()
