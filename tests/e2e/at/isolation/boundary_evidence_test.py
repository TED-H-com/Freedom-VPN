import unittest
from review_boundary import timeline


class BoundaryQualification(unittest.TestCase):
    def packet(self, n, ingress, egress):
        return dict(ingress_mono_ns=str(ingress),egress_mono_ns=str(egress),probe=dict(
            packet_id=n,phase='dense_boundary',result='ok' if egress else 'timeout',
            scheduled_mono_ns=str(ingress-2),send=dict(mono_ns=str(ingress-1))))

    def test_interleaving_positive_cannot_qualify(self):
        rows,errors=timeline(dict(deadline_mono_ns='1000',packets=[self.packet(1,100,120),self.packet(2,110,0)]))
        self.assertEqual(errors[0]['positive_packet'],1)
        self.assertEqual(errors[0]['next_packet'],2)
        self.assertEqual(rows[1]['result'],'timeout')

    def test_noninterleaving_preserves_drop_and_latency(self):
        rows,errors=timeline(dict(deadline_mono_ns='1000',packets=[self.packet(2,120,0),self.packet(1,100,110)]))
        self.assertEqual(errors,[])
        self.assertEqual(rows[0]['transit_ns'],'10')
        self.assertEqual(rows[0]['latency_ns'],'1')
        self.assertIsNone(rows[1]['transit_ns'])
        self.assertEqual(rows[1]['egress_mono_ns'],'0')
