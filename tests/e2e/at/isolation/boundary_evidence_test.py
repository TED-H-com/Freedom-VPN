import unittest
from review_boundary import timeline, qualify_scope


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

    def test_d142_earlier_positive_overlap_recorded_only(self):
        rows,pairs=timeline(dict(deadline_mono_ns='1000',packets=[
            self.packet(1,100,120),self.packet(2,110,125),self.packet(3,140,150),self.packet(4,160,0)]))
        start,last,first=qualify_scope(rows,pairs)
        self.assertEqual(start,2)
        self.assertFalse(pairs[0]['d142_judged'])
        self.assertEqual([p['packet_id'] for p in pairs[0]['packets']],[1,2])
        self.assertEqual((last['packet_id'],first['packet_id']),(3,4))

    def test_d142_drop_before_later_positive_is_in_scope(self):
        rows,pairs=timeline(dict(deadline_mono_ns='1000',packets=[
            self.packet(1,100,0),self.packet(2,110,140),self.packet(3,130,0)]))
        start,_,_=qualify_scope(rows,pairs)
        self.assertEqual(start,0)
        self.assertTrue(pairs[0]['d142_judged'])

    def test_d142_equal_boundary_is_not_before(self):
        rows,pairs=timeline(dict(deadline_mono_ns='1000',packets=[self.packet(1,100,120),self.packet(2,120,0)]))
        qualify_scope(rows,pairs)
        self.assertTrue(pairs[0]['d142_judged'])
